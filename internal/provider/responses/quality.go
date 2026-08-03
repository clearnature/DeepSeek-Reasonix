package responses

import (
	"net/url"
	"strings"
)

// Gate-3 quality filtering for knowledge-cache sources. The score weights
// follow the retrieval-tier design (5.1):
//
//	whitelist (verified professional source)   +0.40
//	multi-source cross-check (>=2 distinct)    +0.30
//	authority domain (gov/edu/mil/official)    +0.20
//	spam pattern hit                           -0.10 each
//
// Scores are clamped to [0, 1]. Callers decide a cutoff; FilterSources
// defaults to 0.50 (a whitelisted or cross-checked authoritative source
// passes; unknown junk does not).

// authorityTLDs are registrable top-level segments that imply an official
// operator. 中国政府/机构 and edu/research institutions.
var authorityTLDs = []string{"gov.cn", "gov", "edu.cn", "edu", "mil", "ac.cn", "org"}

// trustedDomains is a compact built-in whitelist of verified professional
// sources (subset of deep-research sources.md). Extend as sources.md grows.
var trustedDomains = map[string]bool{
	// 国际权威媒体
	"reuters.com": true, "apnews.com": true, "bbc.com": true, "bbc.co.uk": true,
	"nytimes.com": true, "wsj.com": true, "economist.com": true, "ft.com": true,
	"bloomberg.com": true, "theguardian.com": true, "aljazeera.com": true,
	// 中文权威
	"people.com.cn": true, "xinhuanet.com": true, "cctv.com": true, "gov.cn": true,
	"chinadaily.com.cn": true, "caixin.com": true, "yicai.com": true, "thepaper.cn": true,
	// 学术/知识库
	"wikipedia.org": true, "arxiv.org": true, "nature.com": true, "science.org": true,
	"springer.com": true, "ieee.org": true, "acm.org": true, "semanticscholar.org": true,
	"github.com": true,
	// 官方/组织
	"who.int": true, "un.org": true, "imf.org": true, "worldbank.org": true,
	"nasa.gov": true, "noaa.gov": true, "fda.gov": true,
}

// spamDomains are ad/farm/low-quality sources that pollute results.
var spamDomains = map[string]bool{
	"medium.com": true, "quora.com": true, "reddit.com": true, "baidu.com": true,
	"zhihu.com": true, "sohu.com": true, "toutiao.com": true, "weibo.com": true,
	"163.com": true, "sina.com.cn": true, "qq.com": true, "bilibili.com": true,
	"douyin.com": true, "spam-site.com": true, "advertorial.com": true,
}

// spamURLHints are substrings that mark aggregator/advertorial junk even on
// unknown domains.
var spamURLHints = []string{"utm_source=ad", "sponsored", "affiliate", "advertorial", "/ads/", "click.php"}

// domainOf extracts the normalized registrable domain (host minus scheme,
// port, and www. prefix) from a URL. Empty on unparsable input.
func domainOf(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	host := u.Hostname()
	host = strings.ToLower(host)
	host = strings.TrimPrefix(host, "www.")
	return host
}

// isAuthorityDomain reports whether host ends in an official TLD segment.
func isAuthorityDomain(host string) bool {
	if host == "" {
		return false
	}
	for _, tld := range authorityTLDs {
		if host == tld || strings.HasSuffix(host, "."+tld) {
			return true
		}
	}
	return false
}

// hasSpamHint reports whether the raw URL carries an ad/affiliate marker.
func hasSpamHint(rawURL string) bool {
	lower := strings.ToLower(rawURL)
	for _, h := range spamURLHints {
		if strings.Contains(lower, h) {
			return true
		}
	}
	return false
}

// scoreSource computes the gate-3 credibility for one source. Cross-check
// evidence (nDistinct distinct domains among allSources) is folded in so a
// claim backed by several independent outlets scores higher than a lone one.
func scoreSource(s Source, allSources []Source) float64 {
	score := 0.0
	host := domainOf(s.URL)
	if host != "" {
		if trustedDomains[host] {
			score += 0.40
		}
		if isAuthorityDomain(host) {
			score += 0.20
		}
		if spamDomains[host] {
			score -= 0.10
		}
	}
	if hasSpamHint(s.URL) {
		score -= 0.10
	}
	// Cross-check: at least two distinct domains among all sources.
	seen := map[string]bool{}
	for _, o := range allSources {
		d := domainOf(o.URL)
		if d != "" {
			seen[d] = true
		}
	}
	if len(seen) >= 2 {
		score += 0.30
	}
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}
	return score
}

// ScoreAndTagSources fills Domain/Credibility on every source of an entry.
func ScoreAndTagSources(entry *KnowledgeEntry) {
	if entry == nil {
		return
	}
	for i := range entry.Sources {
		s := &entry.Sources[i]
		s.Domain = domainOf(s.URL)
		s.Credibility = scoreSource(*s, entry.Sources)
	}
}

// FilterSources keeps sources scoring at or above minScore (default 0.5 when
// minScore <= 0). A source is also kept when it is whitelisted regardless of
// score — verified professional outlets beat a generic cutoff.
func FilterSources(sources []Source, minScore float64) []Source {
	if minScore <= 0 {
		minScore = 0.5
	}
	out := make([]Source, 0, len(sources))
	for _, s := range sources {
		if s.Credibility >= minScore || (s.Domain != "" && trustedDomains[s.Domain]) {
			out = append(out, s)
		}
	}
	return out
}
