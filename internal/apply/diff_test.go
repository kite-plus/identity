package apply

import (
	"slices"
	"testing"
)

func TestDiffTreatsOmittedAsZero(t *testing.T) {
	want := obj{
		"allowRegister":       true,
		"forceMfa":            false,
		"minLength":           "10",
		"maxOtpAttempts":      "0",
		"mfaInitSkipLifetime": "0s",
		"redirectUris":        []string{"b", "a"},
		"restWebhook":         obj{"interruptOnError": false},
	}
	// What the gateway returns: zero values omitted, numbers as JSON
	// numbers or strings, lists in any order.
	have := obj{
		"allowRegister": true,
		"minLength":     float64(10),
		"redirectUris":  []any{"a", "b"},
		"restWebhook":   obj{},
	}
	if d := diff(want, have); len(d) != 0 {
		t.Errorf("unexpected differences %v", d)
	}
	have["allowRegister"] = false
	have["redirectUris"] = []any{"a"}
	if d := diff(want, have); !slices.Equal(d, []string{"allowRegister", "redirectUris"}) {
		t.Errorf("differences %v", d)
	}
}
