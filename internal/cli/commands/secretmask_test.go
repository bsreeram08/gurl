package commands

import (
	"context"
	"strings"
	"testing"

	"github.com/sreeram/gurl/pkg/types"
)

func TestShowCommandMasksSecrets(t *testing.T) {
	db := newMockDB()
	db.requests["req-1"] = &types.SavedRequest{
		ID:      "req-1",
		Name:    "charge",
		URL:     "https://api.example.com/charges",
		Method:  "POST",
		Headers: []types.Header{{Key: "Authorization", Value: "Bearer sk_live_123"}},
	}
	db.names["charge"] = "req-1"

	tests := []struct {
		name       string
		args       []string
		wantSecret bool
	}{
		{name: "pretty masks", args: []string{"show", "charge"}},
		{name: "json masks", args: []string{"show", "--json", "charge"}},
		{name: "curl masks", args: []string{"show", "--format", "curl", "charge"}},
		{name: "reveal shows", args: []string{"show", "--json", "--reveal", "charge"}, wantSecret: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var runErr error
			out := captureStdout(t, func() {
				runErr = ShowCommand(db).Run(context.Background(), tt.args)
			})
			if runErr != nil {
				t.Fatalf("unexpected error: %v", runErr)
			}
			if got := strings.Contains(out, "sk_live_123"); got != tt.wantSecret {
				t.Errorf("secret present = %v, want %v; output:\n%s", got, tt.wantSecret, out)
			}
		})
	}
}
