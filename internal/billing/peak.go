package billing

import "time"

// IsPeakHour reports whether now falls in DeepSeek's peak billing window
// (Beijing time Monday–Friday 09:00–12:00 and 14:00–18:00). It delegates to
// DeepSeekRateBand so both the catalog resolution and the config dual-rate
// selection share one schedule source of truth.
func IsPeakHour(now time.Time) bool {
	return DeepSeekRateBand(now) == RateBandPeak
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
