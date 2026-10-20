package commands

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/kballard/go-shellquote"
	"github.com/sreeram/gurl/internal/redact"
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

func TestShowCommandCurlFormatOneArgumentPerHeader(t *testing.T) {
	db := newMockDB()
	db.requests["req-1"] = &types.SavedRequest{
		ID:     "req-1",
		Name:   "charge",
		URL:    "https://api.example.com/charges?a=1&b=2",
		Method: "POST",
		Headers: []types.Header{
			{Key: "Authorization", Value: "Bearer sk_live_123"},
			{Key: "Content-Type", Value: "application/json"},
			{Key: "X-Note", Value: "it's quoted"},
		},
		Body: `{"amount": 5}`,
	}
	db.names["charge"] = "req-1"

	for _, tt := range []struct {
		name string
		args []string
		auth string
	}{
		{name: "masked", args: []string{"show", "--format", "curl", "charge"}, auth: "Authorization: " + redact.Mask},
		{name: "reveal", args: []string{"show", "--format", "curl", "--reveal", "charge"}, auth: "Authorization: Bearer sk_live_123"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var runErr error
			out := captureStdout(t, func() {
				runErr = ShowCommand(db).Run(context.Background(), tt.args)
			})
			if runErr != nil {
				t.Fatalf("unexpected error: %v", runErr)
			}
			args, err := shellquote.Split(out)
			if err != nil {
				t.Fatalf("output is not valid shell: %v\n%s", err, out)
			}
			want := []string{"curl", "-X", "POST",
				"-H", tt.auth,
				"-H", "Content-Type: application/json",
				"-H", "X-Note: it's quoted",
				"-d", `{"amount": 5}`,
				"https://api.example.com/charges?a=1&b=2"}
			if !reflect.DeepEqual(args, want) {
				t.Fatalf("curl args:\n got %q\nwant %q\noutput:\n%s", args, want, out)
			}
			for _, line := range strings.Split(out, "\n") {
				if strings.TrimSpace(line) == `-H \` {
					t.Fatalf("-H split from its value:\n%s", out)
				}
			}
		})
	}
}
