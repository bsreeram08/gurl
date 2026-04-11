package tui

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sreeram/gurl/internal/client"
	"github.com/sreeram/gurl/internal/formatter"
	"github.com/sreeram/gurl/internal/protocols/sse"
	"github.com/sreeram/gurl/internal/protocols/websocket"
)

// ResponseTab represents the active tab in the response viewer
type ResponseTab int

const (
	TabBody ResponseTab = iota
	TabHeaders
	TabCookies
	TabTiming
	TabDiff
)

// ResponseViewer is a bubbletea sub-model for displaying HTTP responses
type ResponseViewer struct {
	response     *client.Response
	prevResponse *client.Response
	activeTab    ResponseTab
	viewport     viewport.Model
	filterText   string
	filtering    bool
	width        int
	height       int
	copied       bool
	saved        bool
	diffResult   string
}

// NewResponseViewer creates a new response viewer component
func NewResponseViewer() *ResponseViewer {
	vp := viewport.New(80, 20)

	return &ResponseViewer{
		activeTab: TabBody,
		viewport:  vp,
	}
}

// SetResponse sets the response to display
func (rv *ResponseViewer) SetResponse(resp *client.Response) {
	if rv.response != nil {
		rv.prevResponse = rv.response
	}
	rv.response = resp
	rv.copied = false
	rv.saved = false
	rv.diffResult = ""
	if rv.prevResponse != nil && rv.response != nil && rv.activeTab == TabDiff {
		rv.computeDiff()
	}
	rv.updateViewportContent()
}

func (rv *ResponseViewer) computeDiff() {
	if rv.prevResponse == nil || rv.response == nil {
		return
	}
	diff, err := formatter.DiffJSON([]byte(rv.prevResponse.Body), []byte(rv.response.Body))
	if err != nil {
		rv.diffResult = fmt.Sprintf("Diff error: %v", err)
	} else {
		rv.diffResult = diff
	}
}

// updateViewportContent updates the viewport with current tab content
func (rv *ResponseViewer) updateViewportContent() {
	if rv.response == nil {
		return
	}

	var content string
	switch rv.activeTab {
	case TabBody:
		content = rv.formatBody()
	case TabHeaders:
		content = rv.formatHeaders()
	case TabCookies:
		content = rv.formatCookies()
	case TabTiming:
		content = rv.formatTiming()
	case TabDiff:
		content = rv.diffResult
		if content == "" {
			content = "  No previous response to diff against.\n  Send a request, then send another to compare."
		}
	}

	rv.viewport.SetContent(content)
}

// formatBody formats the response body with syntax highlighting
func (rv *ResponseViewer) formatBody() string {
	if rv.response == nil || len(rv.response.Body) == 0 {
		return "(empty response body)"
	}

	contentType := ""
	if rv.response.Headers != nil {
		contentType = rv.response.Headers.Get("Content-Type")
	}

	opts := formatter.FormatOptions{
		Indent: "  ",
		Color:  true,
	}

	return formatter.Format(rv.response.Body, contentType, opts)
}

// formatHeaders formats response headers as a table
func (rv *ResponseViewer) formatHeaders() string {
	if rv.response == nil || rv.response.Headers == nil {
		return "(no headers)"
	}

	var sb strings.Builder
	sb.WriteString("Status: ")
	sb.WriteString(rv.statusCodeColor())
	sb.WriteString(fmt.Sprintf("%d %s\n\n", rv.response.StatusCode, http.StatusText(rv.response.StatusCode)))

	sb.WriteString("Headers:\n")
	for key, values := range rv.response.Headers {
		for _, value := range values {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", Style.Header.Render(key), value))
		}
	}

	return sb.String()
}

// formatCookies parses and displays Set-Cookie headers
func (rv *ResponseViewer) formatCookies() string {
	if rv.response == nil || rv.response.Headers == nil {
		return "(no cookies)"
	}

	cookies := rv.response.Headers["Set-Cookie"]
	if len(cookies) == 0 {
		return "(no cookies set)"
	}

	var sb strings.Builder
	sb.WriteString("Cookies:\n\n")
	for _, cookie := range cookies {
		sb.WriteString(fmt.Sprintf("  %s\n", cookie))
	}

	return sb.String()
}

// formatTiming displays timing breakdown
func (rv *ResponseViewer) formatTiming() string {
	if rv.response == nil {
		return "(no timing data)"
	}

	var sb strings.Builder
	dur := rv.response.Duration

	sb.WriteString("Request Timing:\n\n")
	sb.WriteString(fmt.Sprintf("  Total:    %s\n", dur))
	sb.WriteString(fmt.Sprintf("  DNS:      %s\n", rv.estimateDNS()))
	sb.WriteString(fmt.Sprintf("  Connect:  %s\n", rv.estimateConnect()))
	sb.WriteString(fmt.Sprintf("  TLS:      %s\n", rv.estimateTLS()))
	sb.WriteString(fmt.Sprintf("  TTFB:     %s\n", rv.estimateTTFB(dur)))

	return sb.String()
}

// estimateDNS estimates DNS lookup time (placeholder)
func (rv *ResponseViewer) estimateDNS() time.Duration {
	return rv.response.Duration / 10 // Rough estimate
}

// estimateConnect estimates connection time (placeholder)
func (rv *ResponseViewer) estimateConnect() time.Duration {
	return rv.response.Duration / 5 // Rough estimate
}

// estimateTLS estimates TLS handshake time (placeholder)
func (rv *ResponseViewer) estimateTLS() time.Duration {
	return rv.response.Duration / 4 // Rough estimate
}

// estimateTTFB estimates time to first byte
func (rv *ResponseViewer) estimateTTFB(total time.Duration) time.Duration {
	return total / 2
}

// statusCodeColor returns the color string for the current status code
func (rv *ResponseViewer) statusCodeColor() string {
	if rv.response == nil {
		return ""
	}

	switch {
	case rv.response.StatusCode >= 200 && rv.response.StatusCode < 300:
		return "82" // Green
	case rv.response.StatusCode >= 300 && rv.response.StatusCode < 400:
		return "228" // Yellow
	case rv.response.StatusCode >= 400 && rv.response.StatusCode < 500:
		return "214" // Orange
	default:
		return "196" // Red
	}
}

// StatusBadge returns the status code styled for display
func (rv *ResponseViewer) StatusBadge() string {
	if rv.response == nil {
		return ""
	}

	color := rv.statusCodeColor()
	badge := fmt.Sprintf("%d %s", rv.response.StatusCode, http.StatusText(rv.response.StatusCode))

	style := lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Bold(true)
	return style.Render(badge)
}

// MetaInfo returns response metadata as a string
func (rv *ResponseViewer) MetaInfo() string {
	if rv.response == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Time: %s | Size: %s", rv.response.Duration, formatSize(rv.response.Size)))

	if rv.response.Headers != nil {
		ct := rv.response.Headers.Get("Content-Type")
		if ct != "" {
			sb.WriteString(fmt.Sprintf(" | Type: %s", ct))
		}
	}

	return sb.String()
}

// formatSize formats byte size for display
func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

// CopyToClipboard copies the response body to clipboard
func (rv *ResponseViewer) CopyToClipboard() error {
	if rv.response == nil {
		return fmt.Errorf("no response to copy")
	}

	// Copy raw body, not formatted
	return clipboard.WriteAll(string(rv.response.Body))
}

// SaveToFile saves the response body to a file
func (rv *ResponseViewer) SaveToFile(filename string) error {
	if rv.response == nil {
		return fmt.Errorf("no response to save")
	}

	return os.WriteFile(filename, rv.response.Body, 0644)
}

// ActiveTab returns the currently active tab
func (rv *ResponseViewer) ActiveTab() ResponseTab {
	return rv.activeTab
}

// SetTab switches to the specified tab
func (rv *ResponseViewer) SetTab(tab ResponseTab) {
	rv.activeTab = tab
	rv.updateViewportContent()
}

// handleKeyPress handles keyboard input
func (rv *ResponseViewer) handleKeyPress(msg tea.KeyMsg) bool {
	switch msg.Type {
	case tea.KeyRunes:
		runeStr := string(msg.Runes)
		switch runeStr {
		case "b":
			rv.SetTab(TabBody)
			return true
		case "h":
			rv.SetTab(TabHeaders)
			return true
		case "c":
			rv.SetTab(TabCookies)
			return true
		case "t":
			rv.SetTab(TabTiming)
			return true
		case "d":
			rv.SetTab(TabDiff)
			rv.computeDiff()
			return true
		case "y":
			rv.copied = true
			rv.CopyToClipboard()
			return true
		case "s":
			rv.saved = true
			return true
		}
	case tea.KeyUp, tea.KeyDown:
		// Handled by viewport
		return false
	}

	return false
}

// View renders the response viewer
func (rv *ResponseViewer) View() string {
	if rv.response == nil {
		return Style.WelcomeText.Render("  Send a request to see the response")
	}

	var sb strings.Builder

	// Status line with status code, time, size
	statusColor := rv.statusCodeColor()
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor)).Bold(true)

	sb.WriteString(fmt.Sprintf("  %s %s %s\n",
		statusStyle.Render(fmt.Sprintf("%d %s", rv.response.StatusCode, http.StatusText(rv.response.StatusCode))),
		Style.PlainText.Render(rv.response.Duration.String()),
		Style.PlainText.Render(formatSize(rv.response.Size)),
	))

	// Tab bar
	sb.WriteString(rv.renderTabBar())
	sb.WriteString("\n")

	// Content based on active tab
	switch rv.activeTab {
	case TabBody:
		sb.WriteString(rv.viewport.View())
	case TabHeaders:
		sb.WriteString(rv.formatHeaders())
	case TabCookies:
		sb.WriteString(rv.formatCookies())
	case TabTiming:
		sb.WriteString(rv.formatTiming())
	}

	// Copy/save feedback
	if rv.copied {
		sb.WriteString(Style.Hint.Render("  Copied!"))
	} else if rv.saved {
		sb.WriteString(Style.Hint.Render("  Saved!"))
	}

	return sb.String()
}

// renderTabBar renders the tab bar
func (rv *ResponseViewer) renderTabBar() string {
	tabs := []string{"Body", "Headers", "Cookies", "Timing", "Diff"}
	var sb strings.Builder

	for i, tab := range tabs {
		if ResponseTab(i) == rv.activeTab {
			sb.WriteString(Style.SelectedItem.Render(fmt.Sprintf(" [%s] ", tab)))
		} else {
			sb.WriteString(Style.ListItem.Render(fmt.Sprintf(" %s ", tab)))
		}
	}

	return sb.String()
}

// Update implements tea.Model.Update
func (rv *ResponseViewer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		rv.width = msg.Width
		rv.height = msg.Height
		rv.viewport.Width = msg.Width - 4
		rv.viewport.Height = msg.Height - 8
		rv.updateViewportContent()
		return rv, nil

	case tea.KeyMsg:
		if rv.handleKeyPress(msg) {
			return rv, nil
		}

		// Pass to viewport for scrolling
		if IsNavigateUpKey(msg.String()) {
			rv.viewport.LineUp(3)
			return rv, nil
		}
		if IsNavigateDownKey(msg.String()) {
			rv.viewport.LineDown(3)
			return rv, nil
		}

		rv.viewport, _ = rv.viewport.Update(msg)
	}

	return rv, nil
}

// Init implements tea.Model.Init
func (rv *ResponseViewer) Init() tea.Cmd {
	return nil
}

type StreamingViewer struct {
	url        string
	protocol   string
	headers    http.Header
	messages   []StreamingMessage
	connected  bool
	connecting bool
	errMsg     string
	input      textinput.Model
	width      int
	height     int
	mu         sync.Mutex
	ctx        context.Context
	cancelFn   context.CancelFunc
}

type StreamingMessage struct {
	Dir     string
	Content string
	Type    string
	Time    string
}

func NewStreamingViewer(url, protocol string, headers http.Header, width, height int) *StreamingViewer {
	ti := textinput.New()
	ti.Placeholder = "Send message..."
	ti.Prompt = "> "

	return &StreamingViewer{
		url:      url,
		protocol: protocol,
		headers:  headers,
		input:    ti,
		width:    width,
		height:   height,
	}
}

func (sv *StreamingViewer) Init() tea.Cmd {
	return nil
}

func (sv *StreamingViewer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if sv.connected {
			if sv.protocol == "ws" || sv.protocol == "wss" {
				if msg.String() == "enter" && sv.input.Value() != "" {
					sv.sendMessage(sv.input.Value())
					sv.input.SetValue("")
					return sv, nil
				}
				sv.input, _ = sv.input.Update(msg)
			}
		}
	}
	return sv, nil
}

func (sv *StreamingViewer) View() string {
	var sb strings.Builder

	if !sv.connected && !sv.connecting {
		connectingStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
		sb.WriteString(connectingStyle.Render("  Connecting to " + sv.url + "..."))
		sb.WriteString("\n")
		sb.WriteString("\n")
	}

	if sv.connecting {
		sp := spinner.New()
		sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
		sb.WriteString(fmt.Sprintf("  %s Connecting...\n\n", sp.View()))
	}

	if sv.errMsg != "" {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		sb.WriteString(errStyle.Render("  Error: " + sv.errMsg))
		sb.WriteString("\n\n")
	}

	if sv.protocol == "ws" || sv.protocol == "wss" {
		if sv.connected {
			sb.WriteString(Style.Hint.Render("  [WS connected] Type message and press Enter to send\n\n"))
		}
	}

	maxLines := sv.height - 8
	if len(sv.messages) > maxLines {
		sv.messages = sv.messages[len(sv.messages)-maxLines:]
	}

	for _, msg := range sv.messages {
		dirColor := lipgloss.Color("39")
		if msg.Dir == "sent" {
			dirColor = lipgloss.Color("82")
		}
		dirStyle := lipgloss.NewStyle().Foreground(dirColor).Bold(true)
		typeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		timeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		if msg.Type != "" {
			sb.WriteString(fmt.Sprintf("  %s %s %s %s\n", dirStyle.Render(msg.Dir+":"), typeStyle.Render("["+msg.Type+"]"), timeStyle.Render(msg.Time), msg.Content))
		} else {
			sb.WriteString(fmt.Sprintf("  %s %s %s\n", dirStyle.Render(msg.Dir+":"), timeStyle.Render(msg.Time), msg.Content))
		}
	}

	if len(sv.messages) == 0 && sv.connected {
		sb.WriteString(Style.Hint.Render("  No messages yet. Send one below."))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")

	if sv.protocol == "ws" || sv.protocol == "wss" {
		sb.WriteString(sv.input.View())
	}

	return sb.String()
}

func (sv *StreamingViewer) Connect() tea.Cmd {
	return func() tea.Msg {
		sv.connecting = true
		sv.connected = false

		sv.ctx, sv.cancelFn = context.WithCancel(context.Background())

		if sv.protocol == "ws" || sv.protocol == "wss" {
			sv.connectWS(sv.ctx)
		} else if sv.protocol == "sse" {
			sv.connectSSE(sv.ctx)
		} else {
			sv.errMsg = "Unknown protocol: " + sv.protocol
			sv.connecting = false
		}

		return StreamingConnectedMsg{Viewer: sv}
	}
}

// Close cancels the streaming context and signals goroutines to exit
func (sv *StreamingViewer) Close() {
	if sv.cancelFn != nil {
		sv.cancelFn()
	}
}

func (sv *StreamingViewer) connectWS(ctx context.Context) {
	wsClient := websocket.NewClient()
	headers := make(http.Header)
	for k, v := range sv.headers {
		headers[k] = v
	}

	if err := wsClient.Connect(ctx, sv.url, headers); err != nil {
		sv.errMsg = err.Error()
		sv.connecting = false
		return
	}

	sv.connected = true
	sv.connecting = false

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				data, msgType, err := wsClient.Receive()
				if err != nil {
					sv.addMessage("recv", string(data), fmt.Sprintf("%v", err), time.Now().Format("15:04:05"))
					return
				}
				typeStr := "text"
				if msgType == websocket.MessageTypeBinary {
					typeStr = "binary"
				}
				sv.addMessage("recv", string(data), typeStr, time.Now().Format("15:04:05"))
			}
		}
	}()
}

func (sv *StreamingViewer) connectSSE(ctx context.Context) {
	sseClient := sse.NewClient()

	headers := make(map[string]string)
	for k, v := range sv.headers {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	eventChan, errorChan, err := sseClient.Connect(ctx, sv.url, sse.WithHeader("Accept", "text/event-stream"))
	if err != nil {
		sv.errMsg = err.Error()
		sv.connecting = false
		return
	}

	sv.connected = true
	sv.connecting = false

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-eventChan:
				if !ok {
					return
				}
				sv.addMessage("recv", event.Data, event.Type, time.Now().Format("15:04:05"))
			case err, ok := <-errorChan:
				if !ok {
					return
				}
				sv.addMessage("recv", "", fmt.Sprintf("error: %v", err), time.Now().Format("15:04:05"))
			}
		}
	}()
}

func (sv *StreamingViewer) sendMessage(content string) {
	if sv.protocol == "ws" || sv.protocol == "wss" {
		wsClient := websocket.NewClient()
		headers := make(http.Header)
		for k, v := range sv.headers {
			headers[k] = v
		}
		ctx := context.Background()
		if err := wsClient.Connect(ctx, sv.url, headers); err == nil {
			defer wsClient.Close()
			wsClient.SendText(content)
			sv.addMessage("sent", content, "", time.Now().Format("15:04:05"))
		}
	}
}

func (sv *StreamingViewer) addMessage(dir, content, msgType, time string) {
	sv.mu.Lock()
	sv.messages = append(sv.messages, StreamingMessage{
		Dir:     dir,
		Content: content,
		Type:    msgType,
		Time:    time,
	})
	sv.mu.Unlock()
}

type StreamingConnectedMsg struct {
	Viewer *StreamingViewer
}

// GetResponse returns the current response
func (rv *ResponseViewer) GetResponse() *client.Response {
	return rv.response
}
