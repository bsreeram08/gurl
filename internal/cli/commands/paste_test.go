package commands

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/sreeram/gurl/internal/redact"
	"github.com/sreeram/gurl/pkg/types"
)

const clipboardTestSecret = "sk-test-123"

// clipboardTestDB holds a GET request, a POST request with a body, and a
// request with a secret header.
func clipboardTestDB() *mockDB {
	db := newMockDB()
	for _, req := range []*types.SavedRequest{
		{ID: "req-1", Name: "test-request", URL: "https://example.com", Method: "GET"},
		{ID: "req-2", Name: "post-request", URL: "https://example.com/api", Method: "POST", Body: `{"key":"value"}`},
		{ID: "req-3", Name: "secret-request", URL: "https://example.com/me", Method: "GET",
			Headers: []types.Header{{Key: "X-Api-Key", Value: clipboardTestSecret}}},
	} {
		db.requests[req.ID] = req
		db.names[req.Name] = req.ID
	}
	return db
}

func TestPasteCommand(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		clipboardErr error
		wantErr      string
		wantCopied   []string // substrings of the single clipboard write
		wantMissing  []string // substrings the clipboard write must not contain
		wantStdout   []string
		wantNoWrites bool
	}{
		{
			name:       "copies curl command for GET request",
			args:       []string{"test-request"},
			wantCopied: []string{"curl", "https://example.com"},
			wantStdout: []string{"Copied curl command for 'test-request'"},
		},
		{
			name:       "copies method and body",
			args:       []string{"post-request"},
			wantCopied: []string{"-X 'POST'", `{"key":"value"}`, "https://example.com/api"},
		},
		{
			name:        "masks secret headers",
			args:        []string{"secret-request"},
			wantCopied:  []string{"X-Api-Key: " + redact.Mask},
			wantMissing: []string{clipboardTestSecret},
		},
		{
			name:       "reveal copies secret headers",
			args:       []string{"--reveal", "secret-request"},
			wantCopied: []string{"X-Api-Key: " + clipboardTestSecret},
		},
		{
			name:         "prints the command when no clipboard tool exists",
			args:         []string{"test-request"},
			clipboardErr: errNoClipboard,
			wantCopied:   []string{"https://example.com"},
			wantStdout:   []string{"Clipboard tools not available", "curl", "https://example.com"},
		},
		{
			name:         "reports a failing clipboard tool",
			args:         []string{"test-request"},
			clipboardErr: errors.New("exit status 1"),
			wantErr:      "failed to copy to clipboard: exit status 1",
			wantCopied:   []string{"https://example.com"},
		},
		{
			name:         "fails when request name not provided",
			wantErr:      "request name argument is required",
			wantNoWrites: true,
		},
		{
			name:         "fails for non-existent request",
			args:         []string{"nonexistent-request"},
			wantErr:      "request not found",
			wantNoWrites: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clip := fakeClipboard(t, tt.clipboardErr)
			cmd := PasteCommand(clipboardTestDB())

			var err error
			stdout, _ := captureStreams(t, func() {
				err = cmd.Run(context.Background(), append([]string{"paste"}, tt.args...))
			})

			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("want error containing %q, got %v", tt.wantErr, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantNoWrites {
				if len(clip.writes) != 0 {
					t.Fatalf("expected no clipboard writes, got %q", clip.writes)
				}
				return
			}
			if len(clip.writes) != 1 {
				t.Fatalf("expected 1 clipboard write, got %q", clip.writes)
			}
			for _, want := range tt.wantCopied {
				if !strings.Contains(clip.writes[0], want) {
					t.Errorf("clipboard %q does not contain %q", clip.writes[0], want)
				}
			}
			for _, bad := range tt.wantMissing {
				if strings.Contains(clip.writes[0]+stdout, bad) {
					t.Errorf("clipboard or stdout contains %q:\n%s\n%s", bad, clip.writes[0], stdout)
				}
			}
			for _, want := range tt.wantStdout {
				if !strings.Contains(stdout, want) {
					t.Errorf("stdout %q does not contain %q", stdout, want)
				}
			}
		})
	}
}

func TestCodegenClipboard(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		clipboardErr error
		wantErr      string
		wantCopied   []string
		wantMissing  []string
	}{
		{
			name:       "copies generated code instead of printing it",
			args:       []string{"--lang", "curl", "--clipboard", "post-request"},
			wantCopied: []string{"curl", "-X 'POST'", "https://example.com/api"},
		},
		{
			name:        "masks secret headers",
			args:        []string{"--lang", "python", "-c", "secret-request"},
			wantCopied:  []string{"https://example.com/me", redact.Mask},
			wantMissing: []string{clipboardTestSecret},
		},
		{
			name:         "reports missing clipboard tools",
			args:         []string{"--lang", "curl", "--clipboard", "test-request"},
			clipboardErr: errNoClipboard,
			wantErr:      "clipboard tools not available",
			wantCopied:   []string{"https://example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearAgentEnv(t)
			clip := fakeClipboard(t, tt.clipboardErr)
			cmd := CodegenCommand(clipboardTestDB())

			var err error
			stdout, _ := captureStreams(t, func() {
				err = cmd.Run(context.Background(), append([]string{"codegen"}, tt.args...))
			})

			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("want error containing %q, got %v", tt.wantErr, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(clip.writes) != 1 {
				t.Fatalf("expected 1 clipboard write, got %q", clip.writes)
			}
			if strings.TrimSpace(stdout) != "" {
				t.Errorf("--clipboard printed to stdout: %q", stdout)
			}
			for _, want := range tt.wantCopied {
				if !strings.Contains(clip.writes[0], want) {
					t.Errorf("clipboard %q does not contain %q", clip.writes[0], want)
				}
			}
			for _, bad := range tt.wantMissing {
				if strings.Contains(clip.writes[0], bad) {
					t.Errorf("clipboard contains %q:\n%s", bad, clip.writes[0])
				}
			}
		})
	}
}

func TestIsCommandAvailable(t *testing.T) {
	if isCommandAvailable("nonexistent-command-xyz") {
		t.Error("expected nonexistent command to return false")
	}

	if !isCommandAvailable("true") {
		t.Error("expected 'true' command to be available")
	}
}
