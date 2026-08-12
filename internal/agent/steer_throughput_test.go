package agent

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// R4 (P13): steer message throughput baseline. Team mid-turn messages are
// delivered through the per-agent steer queue (agent.go steerQueue/steerMu);
// the queue must sustain bursty multi-caller injection without becoming the
// bottleneck (jobs/mailbox wakeups, P1 envelopes, P8 injection).
func BenchmarkSteerConcurrentInject(b *testing.B) {
	for _, workers := range []int{1, 8, 32} {
		b.Run(fmt.Sprintf("workers=%d", workers), func(b *testing.B) {
			a := &Agent{}
			a.steerQueue = make([]steerEntry, 0, 64)
			b.SetBytes(100)
			var done atomic.Int64
			var wg sync.WaitGroup
			stop := make(chan struct{})
			// Consumer drains at loop pace (one per "turn").
			go func() {
				for {
					select {
					case <-stop:
						return
					default:
						a.consumeSteer()
						done.Add(1)
					}
				}
			}()
			b.ResetTimer()
			for w := 0; w < workers; w++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for i := 0; i < b.N/workers; i++ {
						a.Steer("[mid-turn guidance] step 1 of 4: inspect the draft and fold in the feedback.")
					}
				}()
			}
			wg.Wait()
			close(stop)
		})
	}
}

// TestSteerQueueThroughputFloor guards the R4 baseline: concurrent injection
// plus per-turn consumption must not degrade below a coarse floor (measured
// on a 2026 dev box; team traffic is ~1-10 messages per turn, so 5k msg/s is
// two orders of headroom). Failure here means the queue regressed to a copy
// or lock-hold path, not a real-world ceiling.
func TestSteerQueueThroughputFloor(t *testing.T) {
	a := &Agent{}
	a.steerQueue = make([]steerEntry, 0, 64)
	a.steerRunActive = true // Steer rejects when the agent is not running.
	const total = 50_000
	start := time.Now()
	var wg sync.WaitGroup
	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < total/8; i++ {
				a.Steer("mid-turn steer #" + fmt.Sprint(i))
			}
		}()
	}
	wg.Wait()
	injectDur := time.Since(start)

	consumed := 0
	for {
		_, _, ok := a.consumeSteer()
		if !ok {
			break
		}
		consumed++
	}
	if consumed != total {
		t.Fatalf("consumed = %d, want %d", consumed, total)
	}
	rate := float64(total) / injectDur.Seconds()
	t.Logf("R4 baseline: %d messages injected by 8 goroutines in %v (%.0f msg/s), all consumed",
		total, injectDur, rate)
	if rate < 5_000 {
		t.Fatalf("injection throughput %.0f msg/s below 5000 floor", rate)
	}
}
