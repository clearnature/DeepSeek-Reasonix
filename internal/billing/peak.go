package billing

import "time"

// IsPeakHour reports whether now falls in DeepSeek's peak billing window:
// Beijing time Monday–Friday 09:00–12:00 and 14:00–18:00 (all other times,
// including weekends, are off-peak). The pricing table's 2x peak multiplier
// follows this exact schedule.
func IsPeakHour(now time.Time) bool {
	loc := time.FixedZone("CST", 8*3600) // Asia/Shanghai, UTC+8
	t := now.In(loc)
	if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
		return false
	}
	h, m := t.Hour(), t.Minute()
	mins := h*60 + m
	return (mins >= 9*60 && mins < 12*60) || (mins >= 14*60 && mins < 18*60)
}

// SelectRates picks the peak or base rate card for the given occurrence time.
// A card without peak rates (PeakCacheHit <= 0) is returned unchanged, so
// providers without a peak schedule keep their single price.
func SelectRates(rates RateCard, occurredAt time.Time) RateCard {
	if rates.PeakCacheHit <= 0 && rates.PeakInput <= 0 && rates.PeakOutput <= 0 {
		return rates
	}
	if !IsPeakHour(occurredAt) {
		return rates
	}
	peak := rates
	peak.CacheHit = rates.PeakCacheHit
	peak.Input = rates.PeakInput
	peak.Output = rates.PeakOutput
	return peak
}
