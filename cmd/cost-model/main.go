// Command cost-model audits compaction vs replay economics against local
// usage telemetry (no API calls): it aggregates the stats history by model
// and usage source, calibrates real spend against list prices, and simulates
// no-compaction / periodic-compaction / replay strategies with the observed
// parameters to answer whether compaction pays for itself.
//
// Usage: go run ./cmd/cost-model [-days 7] [-model <id>]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"reasonix/internal/config"
	"reasonix/internal/provider"
)

type usageRecord struct {
	ts         time.Time
	model      string
	source     string
	prompt     int
	completion int
	hit        int
	miss       int
	cost       float64
}

type statsAgg struct {
	requests   int
	prompt     int64
	completion int64
	hit        int64
	miss       int64
	cost       float64
}

func (a *statsAgg) add(r usageRecord) {
	a.requests++
	a.prompt += int64(r.prompt)
	a.completion += int64(r.completion)
	a.hit += int64(r.hit)
	a.miss += int64(r.miss)
	a.cost += r.cost
}

func (a *statsAgg) hitRate() float64 {
	total := a.hit + a.miss
	if total <= 0 {
		return 0
	}
	return float64(a.hit) / float64(total)
}

type audit struct {
	byModel     map[string]*statsAgg
	execByModel map[string]*statsAgg
	compByModel map[string]*statsAgg
	executor    *statsAgg
	compaction  *statsAgg
	execGaps    []int
	firstTS     time.Time
	modelIDs    []string
}

func statsDir() string {
	for _, env := range []string{"REASONIX_STATE_HOME", "REASONIX_HOME"} {
		if root := os.Getenv(env); root != "" {
			return filepath.Join(root, "stats")
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".reasonix", "stats")
}

func parseRecord(line string) (usageRecord, bool) {
	var raw struct {
		Ts         string `json:"ts"`
		Model      string `json:"model"`
		Source     string `json:"usage_source"`
		Prompt     int    `json:"prompt"`
		Completion int    `json:"completion"`
		CacheHit   int    `json:"cache_hit"`
		CacheMiss  int    `json:"cache_miss"`
		CostAmount string `json:"cost_amount"`
	}
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return usageRecord{}, false
	}
	if strings.TrimSpace(raw.Model) == "" {
		return usageRecord{}, false
	}
	ts, err := time.Parse(time.RFC3339Nano, raw.Ts)
	if err != nil {
		return usageRecord{}, false
	}
	cost, _ := strconv.ParseFloat(raw.CostAmount, 64)
	return usageRecord{
		ts: ts, model: raw.Model, source: raw.Source,
		prompt: raw.Prompt, completion: raw.Completion,
		hit: raw.CacheHit, miss: raw.CacheMiss, cost: cost,
	}, true
}

// modelID normalizes a stats model ref ("deepseek-responses/deepseek-v4-flash")
// to the config price key ("deepseek-v4-flash").
func modelID(ref string) string {
	if i := strings.LastIndex(ref, "/"); i >= 0 {
		return ref[i+1:]
	}
	return ref
}

func collect(dir string, cutoff time.Time, filter string) (*audit, error) {
	a := &audit{
		byModel:     map[string]*statsAgg{},
		execByModel: map[string]*statsAgg{},
		compByModel: map[string]*statsAgg{},
		executor:    &statsAgg{},
		compaction:  &statsAgg{},
		firstTS:     time.Now(),
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	var lastPrompt int
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			rec, ok := parseRecord(line)
			if !ok || rec.ts.Before(cutoff) {
				continue
			}
			if rec.ts.Before(a.firstTS) {
				a.firstTS = rec.ts
			}
			id := modelID(rec.model)
			if filter != "" && id != filter {
				continue
			}
			agg := a.byModel[id]
			if agg == nil {
				agg = &statsAgg{}
				a.byModel[id] = agg
			}
			agg.add(rec)
			switch rec.source {
			case "compaction":
				a.compaction.add(rec)
				cagg := a.compByModel[id]
				if cagg == nil {
					cagg = &statsAgg{}
					a.compByModel[id] = cagg
				}
				cagg.add(rec)
			case "executor":
				a.executor.add(rec)
				eagg := a.execByModel[id]
				if eagg == nil {
					eagg = &statsAgg{}
					a.execByModel[id] = eagg
				}
				eagg.add(rec)
				if lastPrompt > 0 && rec.prompt > lastPrompt {
					a.execGaps = append(a.execGaps, rec.prompt-lastPrompt)
				}
				lastPrompt = rec.prompt
			}
		}
	}
	for id := range a.byModel {
		a.modelIDs = append(a.modelIDs, id)
	}
	sort.Strings(a.modelIDs)
	return a, nil
}

func listPrices(cfg *config.Config) map[string]*provider.Pricing {
	out := map[string]*provider.Pricing{}
	for _, entry := range cfg.Providers {
		for id, p := range entry.Prices {
			out[id] = p
		}
		if entry.Price != nil {
			out["*"] = entry.Price
		}
	}
	return out
}

func priceFor(prices map[string]*provider.Pricing, id string) *provider.Pricing {
	if p := prices[id]; p != nil {
		return p
	}
	return prices["*"]
}

func expectedCost(p *provider.Pricing, a *statsAgg) float64 {
	if p == nil {
		return 0
	}
	return float64(a.hit)/1e6*p.CacheHit + float64(a.miss)/1e6*p.Input + float64(a.completion)/1e6*p.Output
}

func printUsageHeader(a *audit, dir string, days int) {
	fmt.Printf("cost-model audit: %d day(s) of stats under %s\n", days, dir)
	fmt.Printf("span: %s .. %s\n", a.firstTS.Format("2006-01-02 15:04"), time.Now().Format("2006-01-02 15:04"))
	fmt.Printf("\n%-30s %7s %12s %12s %12s %10s %10s %9s\n",
		"model", "reqs", "input", "output", "hit", "miss", "cost(actual)", "hitRate")
	for _, id := range a.modelIDs {
		agg := a.byModel[id]
		fmt.Printf("%-30s %7d %12d %12d %12d %10d %10.4f %8.1f%%\n",
			id, agg.requests, agg.prompt, agg.completion, agg.hit, agg.miss, agg.cost, agg.hitRate()*100)
	}
	fmt.Printf("\nusage by source:\n")
	for _, s := range []struct {
		name string
		agg  *statsAgg
	}{{"executor", a.executor}, {"compaction", a.compaction}} {
		if s.agg.requests == 0 {
			continue
		}
		fmt.Printf("  %-10s reqs=%-5d input=%-12d hit=%-10d miss=%-10d actual=¥%-8.4f hitRate=%5.1f%%\n",
			s.name, s.agg.requests, s.agg.prompt, s.agg.hit, s.agg.miss, s.agg.cost, s.agg.hitRate()*100)
	}
}

func printCalibration(a *audit, prices map[string]*provider.Pricing) {
	fmt.Printf("\ncalibration (actual spend vs list price):\n")
	for _, id := range a.modelIDs {
		agg := a.byModel[id]
		p := priceFor(prices, id)
		if p == nil || agg.miss+agg.hit == 0 {
			continue
		}
		expected := expectedCost(p, agg)
		ratio := 0.0
		if expected > 0 {
			ratio = agg.cost / expected * 100
		}
		fmt.Printf("  %-28s list=hit¥%.3f/in¥%.3f/out¥%.3f  expected=¥%.3f  actual=¥%.3f  (%.0f%% of list)\n",
			id, p.CacheHit, p.Input, p.Output, expected, agg.cost, ratio)
	}
}

func printSimulation(a *audit, prices map[string]*provider.Pricing, filter, hitOverride string) {
	rounds := 200
	grow := 8000
	if len(a.execGaps) > 0 {
		sort.Ints(a.execGaps)
		grow = a.execGaps[len(a.execGaps)/2]
		if grow <= 0 {
			grow = 8000
		}
	}
	rate := a.executor.hitRate()
	if rate <= 0 {
		rate = 0.95
	}
	fmt.Printf("\nsimulation (observed params):\n")
	hitP := 0.0
	if hitOverride != "" {
		hitP, _ = strconv.ParseFloat(hitOverride, 64)
	}
	for _, id := range a.modelIDs {
		if filter != "" && id != filter {
			continue
		}
		p := priceFor(prices, id)
		if p == nil {
			continue
		}
		useHit := p.CacheHit
		if hitP > 0 {
			useHit = hitP
		}
		// Effective prices from history: the executor's blended ¥/M (mostly
		// cache hits) prices retention; the compaction source's blended ¥/M
		// prices a fold, exposing miss-penalty differences between providers.
		effHit, effFold := effectivePrices(a, id)
		base := simulate(rounds, grow, rate, useHit, p.Input, p.Output, 0)
		c20 := simulate(rounds, grow, rate, useHit, p.Input, p.Output, 20)
		c40 := simulate(rounds, grow, rate, useHit, p.Input, p.Output, 40)
		mid := rounds / 2
		replayed := float64(mid*grow) / 1e6 * useHit
		suffix := ""
		if hitP > 0 {
			suffix = " (hit-price override)"
		}
		fmt.Printf("  %-20s grow=%-5d hit=%.1f%% list=hit¥%.3f/in¥%.3f  no-comp ¥%.3f | every20 ¥%.3f (%+.1f%%) | every40 ¥%.3f (%+.1f%%) | C1-replay +¥%.4f%s\n",
			id, grow, rate*100, useHit, p.Input, base, c20, (1-c20/base)*100, c40, (1-c40/base)*100, replayed, suffix)
		if effHit > 0 && effFold > 0 {
			baseE := simulate(rounds, grow, rate, effHit, effFold, p.Output, 0)
			c20E := simulate(rounds, grow, rate, effHit, effFold, p.Output, 20)
			fmt.Printf("  %-20s effective: hit=¥%.4f/M fold=¥%.4f/M  no-comp ¥%.3f | every20 ¥%.3f (%+.1f%%)  ← historical prices\n",
				"", effHit, effFold, baseE, c20E, (1-c20E/baseE)*100)
		}
	}
}

// effectivePrices derives blended per-M prices from the history: the executor
// record prices history retention (cache-heavy), the compaction record prices
// a fold request (miss-heavy, exposing miss-penalty differences between
// providers).
func effectivePrices(a *audit, id string) (hit, fold float64) {
	if e := a.execByModel[id]; e != nil && e.prompt > 0 {
		hit = e.cost / float64(e.prompt) * 1e6
	}
	if c := a.compByModel[id]; c != nil && c.prompt > 0 {
		fold = c.cost / float64(c.prompt) * 1e6
	}
	return hit, fold
}

// simulate prices one 200-round session with per-turn growth at the given
// hit rate; compactEvery>0 folds 80% of history into a 3K summary every N
// rounds (a summarize request pays full input price for the fold).
func simulate(rounds, grow int, hitRate, hitP, inP, outP float64, compactEvery int) float64 {
	hist := float64(grow)
	total := 0.0
	for i := 1; i <= rounds; i++ {
		prev := hist - float64(grow)
		total += prev / 1e6 * hitP * hitRate
		total += float64(grow) / 1e6 * inP
		hist += float64(grow)
		if compactEvery > 0 && i%compactEvery == 0 && hist > float64(grow*3) {
			fold := hist * 0.8
			total += fold / 1e6 * inP
			hist = hist*0.2 + 3000
		}
	}
	return total
}

func main() {
	days := flag.Int("days", 7, "how many days of stats to audit")
	modelFilter := flag.String("model", "", "restrict to one model id (default: all)")
	hitPrice := flag.String("hit-price", "", "override cache-hit price per M tokens (sensitivity: e.g. 2.5 for a GLM-style expensive cache)")
	flag.Parse()

	dir := statsDir()
	if dir == "" {
		fmt.Fprintln(os.Stderr, "cost-model: cannot locate stats dir (set REASONIX_STATE_HOME/REASONIX_HOME)")
		os.Exit(1)
	}
	cutoff := time.Now().AddDate(0, 0, -*days)
	a, err := collect(dir, cutoff, *modelFilter)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cost-model:", err)
		os.Exit(1)
	}
	if len(a.byModel) == 0 {
		fmt.Printf("cost-model: no usage records in %s within the last %d day(s)\n", dir, *days)
		return
	}
	cfg, err := config.LoadUserConfigReadOnly()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cost-model: cannot load config for list prices:", err)
		os.Exit(1)
	}
	prices := listPrices(cfg)

	printUsageHeader(a, dir, *days)
	printCalibration(a, prices)
	printSimulation(a, prices, *modelFilter, *hitPrice)
}
