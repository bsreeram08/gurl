package storage

import (
	"testing"

	"github.com/sreeram/gurl/pkg/types"
)

func TestMatchesPattern(t *testing.T) {
	req := &types.SavedRequest{
		Name:    "create order",
		URL:     "https://api.example.com/orders",
		Headers: []types.Header{{Key: "X-Merchant-Id", Value: "M-4242"}},
	}
	for pattern, want := range map[string]bool{"order": true, "example.com": true, "M-4242": true, "nope": false} {
		if got := matchesPattern(req, pattern); got != want {
			t.Errorf("matchesPattern(%q) = %v, want %v", pattern, got, want)
		}
	}
}
