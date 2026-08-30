package billing

import (
	"testing"
	"time"
)

func beijingTime(day string, h, m int) time.Time {
	loc := time.FixedZone("CST", 8*3600)
	t, err := time.ParseInLocation("2006-01-02", day, loc)
	if err != nil {
		panic(err)
	}
	return t.Add(time.Duration(h)*time.Hour + time.Duration(m)*time.Minute)
}

func TestIsPeakHour(t *testing.T) {
	cases := []struct {
		name string
		day  string
		h, m int
		want bool
	}{
		{"mon 09:00", "2026-08-31", 9, 0, true},   // Monday
		{"mon 11:59", "2026-08-31", 11, 59, true}, // inside 09-12
		{"mon 12:00", "2026-08-31", 12, 0, false}, // lunch break
		{"mon 13:59", "2026-08-31", 13, 59, false},
		{"mon 14:00", "2026-08-31", 14, 0, true}, // afternoon window
		{"mon 17:59", "2026-08-31", 17, 59, true},
		{"mon 18:00", "2026-08-31", 18, 0, false}, // after peak
		{"fri 10:00", "2026-09-04", 10, 0, true},  // Friday
		{"sat 10:00", "2026-09-05", 10, 0, false}, // weekend off-peak
		{"sun 10:00", "2026-09-06", 10, 0, false}, // Sunday (today: off-peak all day)
		{"mon 00:00", "2026-08-31", 0, 0, false},
		{"mon 22:00", "2026-08-31", 22, 0, false},
	}
	for _, c := range cases {
		got := IsPeakHour(beijingTime(c.day, c.h, c.m))
		if got != c.want {
			t.Errorf("%s (%s %02d:%02d): got %v, want %v", c.name, c.day, c.h, c.m, got, c.want)
		}
	}
}

func TestSelectRatesPeakAndOffPeak(t *testing.T) {
	rates := RateCard{
		CacheHit: 0.05, Input: 1.5, Output: 4.5, Currency: "CNY",
		PeakCacheHit: 0.10, PeakInput: 3, PeakOutput: 9,
	}
	// Sunday 10:00 (off-peak) — today's actual billing window.
	off := SelectRates(rates, beijingTime("2026-08-30", 10, 0))
	if off.CacheHit != 0.05 || off.Input != 1.5 || off.Output != 4.5 {
		t.Fatalf("off-peak rates = %+v", off)
	}
	// Monday 10:00 (peak).
	on := SelectRates(rates, beijingTime("2026-08-31", 10, 0))
	if on.CacheHit != 0.10 || on.Input != 3 || on.Output != 9 {
		t.Fatalf("peak rates = %+v", on)
	}
}

func TestSelectRatesNoPeakScheduleUnchanged(t *testing.T) {
	rates := RateCard{CacheHit: 0.025, Input: 3, Output: 6, Currency: "¥"} // mimo-style, no peak
	got := SelectRates(rates, beijingTime("2026-08-31", 10, 0))            // peak hour
	if got.CacheHit != 0.025 || got.Input != 3 || got.Output != 6 {
		t.Fatalf("no-peak card must stay unchanged, got %+v", got)
	}
}

func TestDeepSeekRateBandWeekendOffPeak(t *testing.T) {
	// Sunday 10:00 Beijing = UTC 02:00 — upstream rule (UTC window) would say
	// peak, but the official schedule is Mon–Fri only.
	if got := DeepSeekRateBand(beijingTime("2026-08-30", 10, 0)); got != RateBandOffPeak {
		t.Fatalf("Sunday 10:00 = %q, want off_peak", got)
	}
	// Saturday 15:00 Beijing also off-peak.
	if got := DeepSeekRateBand(beijingTime("2026-09-05", 15, 0)); got != RateBandOffPeak {
		t.Fatalf("Saturday 15:00 = %q, want off_peak", got)
	}
	// Monday 10:00 still peak.
	if got := DeepSeekRateBand(beijingTime("2026-08-31", 10, 0)); got != RateBandPeak {
		t.Fatalf("Monday 10:00 = %q, want peak", got)
	}
}
