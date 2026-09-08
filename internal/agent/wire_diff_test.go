package agent

import (
	"strings"
	"testing"

	"reasonix/internal/provider"
)

func TestFirstWireDiff(t *testing.T) {
	base := []provider.Message{
		{Role: provider.RoleUser, Content: "one"},
		{Role: provider.RoleAssistant, Content: "two"},
	}
	cases := []struct {
		name    string
		frozen  []provider.Message
		rebuilt []provider.Message
		want    string
	}{
		{"identical", base, base, ""},
		{"field drift", base, []provider.Message{base[0], {Role: provider.RoleAssistant, Content: "two", Origin: "host"}}, "i1:origin"},
		{"content drift", base, []provider.Message{{Role: provider.RoleUser, Content: "changed"}, base[1]}, "i0:content"},
		{"length drift", base, base[:1], "len2/1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := firstWireDiff(tc.frozen, tc.rebuilt)
			if tc.want == "" {
				if got != "" {
					t.Fatalf("firstWireDiff = %q, want empty", got)
				}
				return
			}
			if !strings.Contains(got, tc.want) {
				t.Fatalf("firstWireDiff = %q, want it to contain %q", got, tc.want)
			}
		})
	}
}
