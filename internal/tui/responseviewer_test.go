package tui

import (
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/sreeram/gurl/internal/client"
)

func TestViewer_DisplayResponse(t *testing.T) {
	rv := NewResponseViewer()

	resp := &client.Response{
		StatusCode: 200,
		Headers:    http.Header{"Content-Type": []string{"application/json"}},
		Body:       []byte(`{"key":"value"}`),
		Duration:   100 * time.Millisecond,
		Size:       15,
	}

	rv.SetResponse(resp)

	if rv.GetResponse() == nil {
		t.Error("expected response to be set")
	}

	view := rv.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
}

func TestViewer_StatusColor(t *testing.T) {
	tests := []struct {
		statusCode int
		wantColor  string
	}{
		{200, "82"}, // 2xx green
		{201, "82"},
		{299, "82"},
		{301, "228"}, // 3xx yellow
		{304, "228"},
		{400, "214"}, // 4xx orange
		{404, "214"},
		{500, "196"}, // 5xx red
		{503, "196"},
	}

	for _, tc := range tests {
		t.Run("", func(t *testing.T) {
			rv := NewResponseViewer()
			rv.response = &client.Response{StatusCode: tc.statusCode}

			got := rv.statusCodeColor()
			if got != tc.wantColor {
				t.Errorf("statusCode %d: got %s, want %s", tc.statusCode, got, tc.wantColor)
			}
		})
	}
}

func TestViewer_PrettyPrintJSON(t *testing.T) {
	rv := NewResponseViewer()

	compactJSON := []byte(`{"name":"test","value":123}`)
	resp := &client.Response{
		StatusCode: 200,
		Headers:    http.Header{"Content-Type": []string{"application/json"}},
		Body:       compactJSON,
		Duration:   time.Second,
		Size:       int64(len(compactJSON)),
	}

	rv.SetResponse(resp)

	content := rv.formatBody()
	if content == "" {
		t.Error("expected formatted body content")
	}

	if content == string(compactJSON) {
		t.Error("expected body to be formatted (pretty printed)")
	}
}

func TestViewer_ScrollBody(t *testing.T) {
	rv := NewResponseViewer()

	largeBody := make([]byte, 10000)
	for i := range largeBody {
		largeBody[i] = 'x'
	}

	resp := &client.Response{
		StatusCode: 200,
		Headers:    http.Header{"Content-Type": []string{"text/plain"}},
		Body:       largeBody,
		Duration:   time.Second,
		Size:       int64(len(largeBody)),
	}

	rv.SetResponse(resp)

	msg := tea.WindowSizeMsg{Width: 80, Height: 40}
	rv.Update(msg)

	upKey := tea.KeyMsg{Type: tea.KeyUp}
	rv.Update(upKey)

	downKey := tea.KeyMsg{Type: tea.KeyDown}
	rv.Update(downKey)
}

func TestViewer_TabSections(t *testing.T) {
	rv := NewResponseViewer()

	resp := &client.Response{
		StatusCode: 200,
		Headers:    http.Header{"Content-Type": []string{"application/json"}, "Set-Cookie": []string{"session=abc"}},
		Body:       []byte(`{"key":"value"}`),
		Duration:   100 * time.Millisecond,
		Size:       15,
	}

	rv.SetResponse(resp)

	if rv.ActiveTab() != TabBody {
		t.Error("expected default tab to be TabBody")
	}

	rv.SetTab(TabHeaders)
	if rv.ActiveTab() != TabHeaders {
		t.Error("expected tab to be TabHeaders")
	}

	rv.SetTab(TabCookies)
	if rv.ActiveTab() != TabCookies {
		t.Error("expected tab to be TabCookies")
	}

	rv.SetTab(TabTiming)
	if rv.ActiveTab() != TabTiming {
		t.Error("expected tab to be TabTiming")
	}

	rv.SetTab(TabBody)
	if rv.ActiveTab() != TabBody {
		t.Error("expected tab to be TabBody")
	}
}

func TestViewer_CopyToClipboard(t *testing.T) {
	rv := NewResponseViewer()

	resp := &client.Response{
		StatusCode: 200,
		Body:       []byte(`copy this`),
		Duration:   time.Second,
		Size:       10,
	}

	rv.SetResponse(resp)

	err := rv.CopyToClipboard()
	if err != nil {
		t.Errorf("CopyToClipboard failed: %v", err)
	}
}

func TestViewer_SaveToFile(t *testing.T) {
	rv := NewResponseViewer()

	resp := &client.Response{
		StatusCode: 200,
		Body:       []byte(`save this content`),
		Duration:   time.Second,
		Size:       19,
	}

	rv.SetResponse(resp)

	tmpFile := "/tmp/gurl_test_response.txt"
	err := rv.SaveToFile(tmpFile)
	if err != nil {
		t.Errorf("SaveToFile failed: %v", err)
	}

	resp2, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Errorf("failed to read saved file: %v", err)
	}

	if string(resp2) != "save this content" {
		t.Errorf("saved content mismatch: got %s", string(resp2))
	}

	os.Remove(tmpFile)
}

func TestViewer_ResponseMeta(t *testing.T) {
	rv := NewResponseViewer()

	resp := &client.Response{
		StatusCode: 200,
		Headers:    http.Header{"Content-Type": []string{"application/json; charset=utf-8"}},
		Body:       []byte(`{"key":"value"}`),
		Duration:   150 * time.Millisecond,
		Size:       15,
	}

	rv.SetResponse(resp)

	meta := rv.MetaInfo()
	if meta == "" {
		t.Error("expected non-empty meta info")
	}

	statusBadge := rv.StatusBadge()
	if statusBadge == "" {
		t.Error("expected non-empty status badge")
	}
}
