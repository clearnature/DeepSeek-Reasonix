package agent

import (
	"testing"

	"reasonix/internal/event"
	"reasonix/internal/provider"
	"reasonix/internal/tool"
)

func TestCompactRatioPriceAware(t *testing.T) {
	cases := []struct {
		name  string
		price *provider.Pricing
		ratio float64
		want  float64
	}{
		{name: "cheap-hit model defers compaction", price: &provider.Pricing{CacheHit: 0.025, Input: 3, Output: 6}, want: 0.90},
		{name: "flash-style ratio keeps default", price: &provider.Pricing{CacheHit: 0.1, Input: 3, Output: 9}, want: defaultCompactRatio},
		{name: "user-configured ratio wins", price: &provider.Pricing{CacheHit: 0.025, Input: 3}, ratio: 0.75, want: 0.75},
		{name: "no pricing keeps default", price: nil, want: defaultCompactRatio},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts := Options{Pricing: tc.price, CompactRatio: tc.ratio}
			a := New(nil, tool.NewRegistry(), NewSession("sys"), opts, event.Discard)
			if a.CompactRatio() != tc.want {
				t.Fatalf("CompactRatio = %v, want %v", a.CompactRatio(), tc.want)
			}
		})
	}
}
