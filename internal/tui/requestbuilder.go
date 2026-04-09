package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sreeram/gurl/internal/client"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
)

// RequestBuilder is a bubbletea sub-model for editing/sending HTTP requests
type RequestBuilder struct {
	// Core data
	db      storage.DB
	request *types.SavedRequest
	editing *types.SavedRequest // Working copy being edited

	// UI State
	width         int
	height        int
	activeSection Section // Currently focused section

	// Method selector
	methodIndex int
	methods     []string

	// URL input
	urlInput textinput.Model

	// URL autocomplete suggestions
	suggestions     []string
	suggestionIndex int
	showSuggestions bool

	// Headers editor
	headerInputs []headerRow

	// Query params editor
	queryInputs []queryRow

	// Body editor
	bodyInput   textarea.Model
	contentType string

	// Auth
	authType   string // "none", "basic", "bearer", "apikey"
	authInputs map[string]textinput.Model

	// Sending state
	sending        bool
	loadingSpinner spinner.Model

	// Messages to return to parent
	msgs []tea.Msg
}

// Section represents different sections in the request builder
type Section int

const (
	SectionMethod Section = iota
	SectionURL
	SectionHeaders
	SectionQueryParams
	SectionBody
	SectionAuth
	SectionSend
)

// headerRow represents a single header key-value pair
type headerRow struct {
	keyInput   textinput.Model
	valueInput textinput.Model
	focused    bool // true if key is focused, false if value
}

// queryRow represents a single query param key-value pair
type queryRow struct {
	keyInput   textinput.Model
	valueInput textinput.Model
	focused    bool
}

// BuilderRequestSelectedMsg is sent when a request is selected to be loaded into the builder
type BuilderRequestSelectedMsg struct {
	Request *types.SavedRequest
}

// RequestSentMsg is sent when a request completes
type RequestSentMsg struct {
	Response *client.Response
	Error    error
}

// RequestSavedMsg is sent when a request is saved
type RequestSavedMsg struct {
	Request *types.SavedRequest
	Error   error
}

// NewRequestBuilder creates a new RequestBuilder component
func NewRequestBuilder(db storage.DB) *RequestBuilder {
	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}

	urlInput := textinput.New()
	urlInput.Placeholder = "Enter URL..."
	urlInput.Prompt = ""

	bodyInput := textarea.New()
	bodyInput.Placeholder = "Request body..."
	bodyInput.SetWidth(60)
	bodyInput.SetHeight(10)
	bodyInput.ShowLineNumbers = false

	loadingSpinner := spinner.New()
	loadingSpinner.Spinner = spinner.Dot

	// Create auth inputs
	usernameInput := textinput.New()
	usernameInput.Placeholder = "Username"

	passwordInput := textinput.New()
	passwordInput.Placeholder = "Password"
	passwordInput.EchoMode = textinput.EchoPassword

	tokenInput := textinput.New()
	tokenInput.Placeholder = "Bearer token"

	apiKeyInput := textinput.New()
	apiKeyInput.Placeholder = "API Key"

	apiValueInput := textinput.New()
	apiValueInput.Placeholder = "Key value"

	apiHeaderInput := textinput.New()
	apiHeaderInput.Placeholder = "Header name (default: X-API-Key)"

	authInputs := map[string]textinput.Model{
		"username":   usernameInput,
		"password":   passwordInput,
		"token":      tokenInput,
		"api_key":    apiKeyInput,
		"api_value":  apiValueInput,
		"api_header": apiHeaderInput,
	}

	rb := &RequestBuilder{
		db:             db,
		methods:        methods,
		methodIndex:    0,
		urlInput:       urlInput,
		bodyInput:      bodyInput,
		contentType:    "json",
		authType:       "none",
		authInputs:     authInputs,
		headerInputs:   []headerRow{},
		queryInputs:    []queryRow{},
		loadingSpinner: loadingSpinner,
	}

	// Add one empty header row by default
	rb.addHeaderRow()

	return rb
}

// LoadRequest loads a saved request into the builder for editing
func (rb *RequestBuilder) LoadRequest(req *types.SavedRequest) {
	rb.request = req

	// Create working copy
	rb.editing = &types.SavedRequest{
		ID:           req.ID,
		Name:         req.Name,
		CurlCmd:      req.CurlCmd,
		URL:          req.URL,
		Method:       req.Method,
		Headers:      make([]types.Header, len(req.Headers)),
		Body:         req.Body,
		Variables:    req.Variables,
		PathParams:   req.PathParams,
		Collection:   req.Collection,
		Tags:         req.Tags,
		OutputFormat: req.OutputFormat,
		AuthConfig:   req.AuthConfig,
		Timeout:      req.Timeout,
		Assertions:   req.Assertions,
		Folder:       req.Folder,
		SortOrder:    req.SortOrder,
		CreatedAt:    req.CreatedAt,
		UpdatedAt:    req.UpdatedAt,
	}
	copy(rb.editing.Headers, req.Headers)

	// Set URL input
	rb.urlInput.SetValue(req.URL)

	// Set method
	for i, m := range rb.methods {
		if strings.EqualFold(m, req.Method) {
			rb.methodIndex = i
			break
		}
	}

	// Set body
	rb.bodyInput.SetValue(req.Body)

	// Set headers
	rb.headerInputs = []headerRow{}
	for _, h := range req.Headers {
		row := rb.addHeaderRow()
		row.keyInput.SetValue(h.Key)
		row.valueInput.SetValue(h.Value)
	}
	if len(req.Headers) == 0 {
		rb.addHeaderRow()
	}

	// Set auth
	if req.AuthConfig != nil {
		rb.authType = req.AuthConfig.Type
		for k, v := range req.AuthConfig.Params {
			if input, ok := rb.authInputs[k]; ok {
				input.SetValue(v)
			}
		}
	} else {
		rb.authType = "none"
	}

	// Set content type from headers if present
	rb.contentType = detectContentType(req.Headers, req.Body)
}

// NewRequest creates a blank request for a new request
func (rb *RequestBuilder) NewRequest() {
	rb.request = nil
	rb.editing = types.NewSavedRequest("New Request", "", "GET")

	rb.urlInput.SetValue("")
	rb.methodIndex = 0
	rb.bodyInput.SetValue("")
	rb.headerInputs = []headerRow{}
	rb.addHeaderRow()
	rb.authType = "none"
	rb.contentType = "json"
}

// Section navigation
func (rb *RequestBuilder) nextSection() {
	switch rb.activeSection {
	case SectionMethod:
		rb.activeSection = SectionURL
	case SectionURL:
		rb.activeSection = SectionHeaders
	case SectionHeaders:
		rb.activeSection = SectionQueryParams
	case SectionQueryParams:
		rb.activeSection = SectionBody
	case SectionBody:
		rb.activeSection = SectionAuth
	case SectionAuth:
		rb.activeSection = SectionSend
	case SectionSend:
		rb.activeSection = SectionMethod
	}
}

func (rb *RequestBuilder) prevSection() {
	switch rb.activeSection {
	case SectionMethod:
		rb.activeSection = SectionSend
	case SectionURL:
		rb.activeSection = SectionMethod
	case SectionHeaders:
		rb.activeSection = SectionURL
	case SectionQueryParams:
		rb.activeSection = SectionHeaders
	case SectionBody:
		rb.activeSection = SectionQueryParams
	case SectionAuth:
		rb.activeSection = SectionBody
	case SectionSend:
		rb.activeSection = SectionAuth
	}
}

// Header row management
func (rb *RequestBuilder) addHeaderRow() *headerRow {
	keyInput := textinput.New()
	keyInput.Placeholder = "Header key..."
	keyInput.Prompt = ""

	valueInput := textinput.New()
	valueInput.Placeholder = "Value..."
	valueInput.Prompt = ""

	row := headerRow{
		keyInput:   keyInput,
		valueInput: valueInput,
		focused:    true,
	}
	rb.headerInputs = append(rb.headerInputs, row)
	return &rb.headerInputs[len(rb.headerInputs)-1]
}

func (rb *RequestBuilder) removeHeaderRow(index int) {
	if index < 0 || index >= len(rb.headerInputs) {
		return
	}
	rb.headerInputs = append(rb.headerInputs[:index], rb.headerInputs[index+1:]...)
	// Ensure at least one row remains
	if len(rb.headerInputs) == 0 {
		rb.addHeaderRow()
	}
}

// Query param row management
func (rb *RequestBuilder) addQueryRow() *queryRow {
	keyInput := textinput.New()
	keyInput.Placeholder = "Param key..."
	keyInput.Prompt = ""

	valueInput := textinput.New()
	valueInput.Placeholder = "Value..."
	valueInput.Prompt = ""

	row := queryRow{
		keyInput:   keyInput,
		valueInput: valueInput,
		focused:    true,
	}
	rb.queryInputs = append(rb.queryInputs, row)
	return &rb.queryInputs[len(rb.queryInputs)-1]
}

func (rb *RequestBuilder) removeQueryRow(index int) {
	if index < 0 || index >= len(rb.queryInputs) {
		return
	}
	rb.queryInputs = append(rb.queryInputs[:index], rb.queryInputs[index+1:]...)
	if len(rb.queryInputs) == 0 {
		rb.addQueryRow()
	}
}

// Update implements tea.Model.Update
func (rb *RequestBuilder) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		rb.width = msg.Width
		rb.height = msg.Height
		return rb, nil

	case tea.KeyMsg:
		return rb.handleKeyPress(msg)

	case BuilderRequestSelectedMsg:
		rb.LoadRequest(msg.Request)
		return rb, nil
	}

	// Update sub-components
	var cmd tea.Cmd
	rb.urlInput, cmd = rb.urlInput.Update(msg)
	if cmd != nil {
		return rb, cmd
	}

	rb.bodyInput, cmd = rb.bodyInput.Update(msg)
	if cmd != nil {
		return rb, cmd
	}

	rb.loadingSpinner, cmd = rb.loadingSpinner.Update(msg)
	if cmd != nil {
		return rb, cmd
	}

	// Update auth inputs
	for _, input := range rb.authInputs {
		input, cmd = input.Update(msg)
	}

	return rb, nil
}

// handleKeyPress handles keyboard input
func (rb *RequestBuilder) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle sending
	if msg.Type == tea.KeyCtrlJ && rb.activeSection == SectionSend {
		return rb, rb.sendRequest()
	}

	// Ctrl+Enter sends request
	if msg.Type == tea.KeyCtrlC {
		return rb, nil
	}

	// Ctrl+S saves request
	if msg.Type == tea.KeyCtrlS {
		return rb, rb.saveRequest()
	}

	// Tab navigates sections
	if msg.String() == "tab" {
		rb.nextSection()
		return rb, nil
	}

	if msg.String() == "shift+tab" {
		rb.prevSection()
		return rb, nil
	}

	// Section-specific key handling
	switch rb.activeSection {
	case SectionMethod:
		return rb.handleMethodKey(msg)

	case SectionURL:
		if rb.showSuggestions {
			switch msg.Type {
			case tea.KeyUp:
				if len(rb.suggestions) > 0 {
					rb.suggestionIndex--
					if rb.suggestionIndex < 0 {
						rb.suggestionIndex = len(rb.suggestions) - 1
					}
				}
				return rb, nil
			case tea.KeyDown:
				if len(rb.suggestions) > 0 {
					rb.suggestionIndex = (rb.suggestionIndex + 1) % len(rb.suggestions)
				}
				return rb, nil
			case tea.KeyEnter:
				rb.selectSuggestion()
				return rb, nil
			case tea.KeyEscape:
				rb.showSuggestions = false
				rb.suggestions = nil
				return rb, nil
			}
		}

		rb.urlInput, _ = rb.urlInput.Update(msg)
		rb.updateSuggestions()
		return rb, nil

	case SectionHeaders:
		return rb.handleHeadersKey(msg)

	case SectionQueryParams:
		return rb.handleQueryParamsKey(msg)

	case SectionBody:
		rb.bodyInput, _ = rb.bodyInput.Update(msg)
		return rb, nil

	case SectionAuth:
		return rb.handleAuthKey(msg)

	case SectionSend:
		if msg.Type == tea.KeyEnter {
			return rb, rb.sendRequest()
		}
		return rb, nil
	}

	return rb, nil
}

func (rb *RequestBuilder) handleMethodKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// 'g' then 'm' cycles methods (g then m shortcut)
	switch msg.Type {
	case tea.KeyRunes:
		runes := string(msg.Runes)
		if runes == "g" || runes == "G" {
			// Start of g+m shortcut, do nothing yet
			return rb, nil
		}
		if runes == "m" || runes == "M" {
			// Cycle to next method
			rb.methodIndex = (rb.methodIndex + 1) % len(rb.methods)
			return rb, nil
		}
	case tea.KeyUp, tea.KeyLeft:
		rb.methodIndex--
		if rb.methodIndex < 0 {
			rb.methodIndex = len(rb.methods) - 1
		}
		return rb, nil
	case tea.KeyDown, tea.KeyRight:
		rb.methodIndex = (rb.methodIndex + 1) % len(rb.methods)
		return rb, nil
	case tea.KeyEnter:
		rb.nextSection()
		return rb, nil
	}
	return rb, nil
}

func (rb *RequestBuilder) handleHeadersKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// 'a' adds a new header
	if msg.Type == tea.KeyRunes && string(msg.Runes) == "a" {
		rb.addHeaderRow()
		return rb, nil
	}

	// 'd' deletes focused header
	if msg.Type == tea.KeyRunes && string(msg.Runes) == "d" {
		// Find focused row and remove it
		for i := range rb.headerInputs {
			if rb.headerInputs[i].keyInput.Focused() || rb.headerInputs[i].valueInput.Focused() {
				rb.removeHeaderRow(i)
				break
			}
		}
		return rb, nil
	}

	// Escape exits headers section
	if msg.Type == tea.KeyEscape {
		rb.nextSection()
		return rb, nil
	}

	// Update header inputs
	for i := range rb.headerInputs {
		var cmd tea.Cmd
		rb.headerInputs[i].keyInput, cmd = rb.headerInputs[i].keyInput.Update(msg)
		if cmd != nil {
			return rb, cmd
		}
		rb.headerInputs[i].valueInput, cmd = rb.headerInputs[i].valueInput.Update(msg)
		if cmd != nil {
			return rb, cmd
		}
	}

	return rb, nil
}

func (rb *RequestBuilder) handleQueryParamsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// 'a' adds a new param
	if msg.Type == tea.KeyRunes && string(msg.Runes) == "a" {
		rb.addQueryRow()
		return rb, nil
	}

	// 'd' deletes focused param
	if msg.Type == tea.KeyRunes && string(msg.Runes) == "d" {
		for i := range rb.queryInputs {
			if rb.queryInputs[i].keyInput.Focused() || rb.queryInputs[i].valueInput.Focused() {
				rb.removeQueryRow(i)
				break
			}
		}
		return rb, nil
	}

	// Escape exits section
	if msg.Type == tea.KeyEscape {
		rb.nextSection()
		return rb, nil
	}

	// Update query inputs
	for i := range rb.queryInputs {
		var cmd tea.Cmd
		rb.queryInputs[i].keyInput, cmd = rb.queryInputs[i].keyInput.Update(msg)
		if cmd != nil {
			return rb, cmd
		}
		rb.queryInputs[i].valueInput, cmd = rb.queryInputs[i].valueInput.Update(msg)
		if cmd != nil {
			return rb, cmd
		}
	}

	return rb, nil
}

func (rb *RequestBuilder) handleAuthKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Escape exits section
	if msg.Type == tea.KeyEscape {
		rb.nextSection()
		return rb, nil
	}

	// Number keys for auth type selection
	if msg.Type == tea.KeyRunes {
		switch string(msg.Runes) {
		case "1":
			rb.authType = "none"
		case "2":
			rb.authType = "basic"
		case "3":
			rb.authType = "bearer"
		case "4":
			rb.authType = "apikey"
		}
	}

	// Update auth inputs if auth type is set
	if rb.authType != "none" {
		for key, input := range rb.authInputs {
			var cmd tea.Cmd
			input, cmd = input.Update(msg)
			if cmd != nil {
				rb.authInputs[key] = input
				return rb, cmd
			}
			rb.authInputs[key] = input
		}
	}

	return rb, nil
}

// sendRequest sends the HTTP request asynchronously via a tea.Cmd.
func (rb *RequestBuilder) sendRequest() tea.Cmd {
	rb.sending = true
	clientReq := rb.buildClientRequest()
	return func() tea.Msg {
		resp, err := client.Execute(clientReq)
		return RequestSentMsg{Response: &resp, Error: err}
	}
}

// saveRequest saves the current request to the database via a tea.Cmd.
func (rb *RequestBuilder) saveRequest() tea.Cmd {
	rb.syncEditingFromForm()
	if rb.editing == nil {
		return nil
	}
	editing := rb.editing
	isUpdate := rb.request != nil && rb.request.ID != ""
	db := rb.db
	return func() tea.Msg {
		var err error
		if isUpdate {
			err = db.UpdateRequest(editing)
		} else {
			editing.ID = ""
			err = db.SaveRequest(editing)
		}
		return RequestSavedMsg{Request: editing, Error: err}
	}
}

// syncEditingFromForm syncs form data back to editing request
func (rb *RequestBuilder) syncEditingFromForm() {
	if rb.editing == nil {
		return
	}

	rb.editing.Method = rb.methods[rb.methodIndex]
	rb.editing.URL = rb.urlInput.Value()

	// Sync headers
	rb.editing.Headers = []types.Header{}
	for _, row := range rb.headerInputs {
		key := row.keyInput.Value()
		value := row.valueInput.Value()
		if key != "" {
			rb.editing.Headers = append(rb.editing.Headers, types.Header{
				Key:   key,
				Value: value,
			})
		}
	}

	// Sync body
	rb.editing.Body = rb.bodyInput.Value()

	// Sync auth
	if rb.authType != "none" {
		params := make(map[string]string)
		switch rb.authType {
		case "basic":
			params["username"] = rb.authInputs["username"].Value()
			params["password"] = rb.authInputs["password"].Value()
		case "bearer":
			params["token"] = rb.authInputs["token"].Value()
		case "apikey":
			params["key"] = rb.authInputs["api_key"].Value()
			params["value"] = rb.authInputs["api_value"].Value()
			params["header"] = rb.authInputs["api_header"].Value()
		}
		rb.editing.AuthConfig = &types.AuthConfig{
			Type:   rb.authType,
			Params: params,
		}
	} else {
		rb.editing.AuthConfig = nil
	}

	rb.editing.UpdatedAt = time.Now().Unix()
}

// buildClientRequest builds a client.Request from the form data
func (rb *RequestBuilder) buildClientRequest() client.Request {
	rb.syncEditingFromForm()

	req := client.Request{
		Method:  rb.editing.Method,
		URL:     rb.editing.URL,
		Headers: []client.Header{},
		Body:    rb.editing.Body,
	}

	// Add headers
	for _, h := range rb.editing.Headers {
		req.Headers = append(req.Headers, client.Header{
			Key:   h.Key,
			Value: h.Value,
		})
	}

	// Add auth headers
	if rb.editing.AuthConfig != nil {
		switch rb.editing.AuthConfig.Type {
		case "basic":
			// Basic auth is handled by client
			req.Headers = append(req.Headers, client.Header{
				Key:   "Authorization",
				Value: "Basic " + basicAuth(rb.editing.AuthConfig.Params["username"], rb.editing.AuthConfig.Params["password"]),
			})
		case "bearer":
			req.Headers = append(req.Headers, client.Header{
				Key:   "Authorization",
				Value: "Bearer " + rb.editing.AuthConfig.Params["token"],
			})
		case "apikey":
			headerName := rb.editing.AuthConfig.Params["header"]
			if headerName == "" {
				headerName = "X-API-Key"
			}
			req.Headers = append(req.Headers, client.Header{
				Key:   headerName,
				Value: rb.editing.AuthConfig.Params["value"],
			})
		}
	}

	return req
}

// basicAuth generates a Basic auth header value
func basicAuth(username, password string) string {
	auth := username + ":" + password
	return encodeBase64([]byte(auth))
}

// encodeBase64 encodes bytes to base64
func encodeBase64(data []byte) string {
	const encodeStd = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	const padding = '='

	if len(data) == 0 {
		return ""
	}

	result := make([]byte, (len(data)+2)/3*4)
	for i, j := 0, 0; i < len(data); i, j = i+3, j+4 {
		var val uint32
		switch len(data) - i {
		case 1:
			val = uint32(data[i]) << 16
			result[j] = encodeStd[val>>18&0x3F]
			result[j+1] = encodeStd[val>>12&0x3F]
			result[j+2] = padding
			result[j+3] = padding
		case 2:
			val = uint32(data[i])<<16 | uint32(data[i+1])<<8
			result[j] = encodeStd[val>>18&0x3F]
			result[j+1] = encodeStd[val>>12&0x3F]
			result[j+2] = encodeStd[val>>6&0x3F]
			result[j+3] = padding
		default:
			val = uint32(data[i])<<16 | uint32(data[i+1])<<8 | uint32(data[i+2])
			result[j] = encodeStd[val>>18&0x3F]
			result[j+1] = encodeStd[val>>12&0x3F]
			result[j+2] = encodeStd[val>>6&0x3F]
			result[j+3] = encodeStd[val&0x3F]
		}
	}
	return string(result)
}

func (rb *RequestBuilder) updateSuggestions() {
	if rb.db == nil {
		rb.suggestions = nil
		return
	}

	requests, err := rb.db.ListRequests(&storage.ListOptions{Limit: 100})
	if err != nil || len(requests) == 0 {
		rb.suggestions = nil
		return
	}

	urlMap := make(map[string]bool)
	var uniqueURLs []string
	for _, req := range requests {
		if req.URL == "" {
			continue
		}
		if !urlMap[req.URL] {
			urlMap[req.URL] = true
			uniqueURLs = append(uniqueURLs, req.URL)
		}
	}

	input := rb.urlInput.Value()
	if input == "" {
		rb.suggestions = nil
		return
	}

	var matches []string
	for _, url := range uniqueURLs {
		if fuzzyMatch(input, url) {
			matches = append(matches, url)
		}
	}

	rb.suggestions = matches
	rb.suggestionIndex = 0
	rb.showSuggestions = len(matches) > 0
}

func fuzzyMatch(input, target string) bool {
	input = strings.ToLower(input)
	target = strings.ToLower(target)

	if strings.Contains(target, input) {
		return true
	}

	if strings.HasPrefix(target, "http://") {
		target = target[7:]
	} else if strings.HasPrefix(target, "https://") {
		target = target[8:]
	}
	if idx := strings.Index(target, "/"); idx > 0 {
		target = target[:idx]
	}
	if strings.Contains(target, input) {
		return true
	}

	inputIdx := 0
	for _, ch := range target {
		if inputIdx < len(input) && strings.ToLower(string(ch)) == strings.ToLower(string(rune(input[inputIdx]))) {
			inputIdx++
		}
	}
	return inputIdx == len(input)
}

func (rb *RequestBuilder) selectSuggestion() {
	if !rb.showSuggestions || rb.suggestionIndex < 0 || rb.suggestionIndex >= len(rb.suggestions) {
		return
	}
	rb.urlInput.SetValue(rb.suggestions[rb.suggestionIndex])
	rb.showSuggestions = false
	rb.suggestions = nil
	rb.suggestionIndex = 0
}

// detectContentType detects content type from headers or body
func detectContentType(headers []types.Header, body string) string {
	for _, h := range headers {
		if strings.EqualFold(h.Key, "Content-Type") {
			if strings.Contains(h.Value, "json") {
				return "json"
			}
			if strings.Contains(h.Value, "xml") {
				return "xml"
			}
			if strings.Contains(h.Value, "form") {
				return "form"
			}
		}
	}

	// Try to detect from body
	body = strings.TrimSpace(body)
	if strings.HasPrefix(body, "{") || strings.HasPrefix(body, "[") {
		return "json"
	}
	if strings.HasPrefix(body, "<") {
		return "xml"
	}
	return "raw"
}

// View implements tea.Model.View
func (rb *RequestBuilder) View() string {
	if rb.editing == nil {
		return rb.welcomeView()
	}

	var sb strings.Builder

	// Method + URL section
	sb.WriteString(rb.renderMethodURL())
	sb.WriteString("\n")

	// Headers section
	sb.WriteString(rb.renderHeaders())
	sb.WriteString("\n")

	// Query params section
	sb.WriteString(rb.renderQueryParams())
	sb.WriteString("\n")

	// Body section
	sb.WriteString(rb.renderBody())
	sb.WriteString("\n")

	// Auth section
	sb.WriteString(rb.renderAuth())
	sb.WriteString("\n")

	// Send section
	sb.WriteString(rb.renderSend())

	return sb.String()
}

func (rb *RequestBuilder) welcomeView() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("82")).
		Bold(true)

	return style.Render("  Welcome to Gurl TUI!") + "\n\n" +
		Style.PlainText.Render("  Select a request from the sidebar to edit") + "\n" +
		Style.Hint.Render("  Press 'n' to create a new request")
}

func (rb *RequestBuilder) renderMethodURL() string {
	method := rb.methods[rb.methodIndex]
	methodColor := getMethodColor(method)

	methodStyle := lipgloss.NewStyle().
		Foreground(methodColor).
		Bold(true)

	selected := rb.activeSection == SectionMethod
	prefix := "  "
	if selected {
		prefix = "▶ "
	}

	url := rb.urlInput.View()
	if rb.activeSection == SectionURL {
	} else {
		if rb.urlInput.Value() == "" {
			url = Style.Hint.Render("Enter URL...")
		}
	}

	var result strings.Builder
	result.WriteString(prefix + methodStyle.Render(method) + " " + url)

	if rb.showSuggestions && len(rb.suggestions) > 0 {
		result.WriteString("\n")
		for i, suggestion := range rb.suggestions {
			rowPrefix := "  "
			if i == rb.suggestionIndex {
				rowPrefix = "▶ "
			}
			result.WriteString(rowPrefix + Style.PlainText.Render(suggestion) + "\n")
		}
	}

	return result.String()
}

func (rb *RequestBuilder) renderHeaders() string {
	selected := rb.activeSection == SectionHeaders
	prefix := "  "
	if selected {
		prefix = "▶ "
	}

	var sb strings.Builder
	sb.WriteString(prefix)
	sb.WriteString(Style.Header.Render("Headers"))
	sb.WriteString(" (a=add, d=delete, tab=next)")
	sb.WriteString("\n")

	for i, row := range rb.headerInputs {
		rowPrefix := "    "
		sb.WriteString(rowPrefix)
		sb.WriteString(fmt.Sprintf("%d. ", i+1))
		sb.WriteString(row.keyInput.View())
		sb.WriteString(" : ")
		sb.WriteString(row.valueInput.View())
		sb.WriteString("\n")
	}

	return sb.String()
}

func (rb *RequestBuilder) renderQueryParams() string {
	selected := rb.activeSection == SectionQueryParams
	prefix := "  "
	if selected {
		prefix = "▶ "
	}

	var sb strings.Builder
	sb.WriteString(prefix)
	sb.WriteString(Style.Header.Render("Query Params"))
	sb.WriteString(" (a=add, d=delete, tab=next)")
	sb.WriteString("\n")

	for i, row := range rb.queryInputs {
		rowPrefix := "    "
		sb.WriteString(rowPrefix)
		sb.WriteString(fmt.Sprintf("%d. ", i+1))
		sb.WriteString(row.keyInput.View())
		sb.WriteString(" : ")
		sb.WriteString(row.valueInput.View())
		sb.WriteString("\n")
	}

	return sb.String()
}

func (rb *RequestBuilder) renderBody() string {
	selected := rb.activeSection == SectionBody
	prefix := "  "
	if selected {
		prefix = "▶ "
	}

	var sb strings.Builder
	sb.WriteString(prefix)
	sb.WriteString(Style.Header.Render("Body"))
	sb.WriteString(" [")
	sb.WriteString(rb.contentType)
	sb.WriteString("]")
	sb.WriteString(" (tab=next)")
	sb.WriteString("\n")

	bodyView := rb.bodyInput.View()
	// Add indent
	lines := strings.Split(bodyView, "\n")
	for i, line := range lines {
		if i == 0 && selected {
			// First line already has cursor
			sb.WriteString("    ")
			sb.WriteString(line)
		} else {
			sb.WriteString("    ")
			sb.WriteString(line)
		}
		if i < len(lines)-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func (rb *RequestBuilder) renderAuth() string {
	selected := rb.activeSection == SectionAuth
	prefix := "  "
	if selected {
		prefix = "▶ "
	}

	var sb strings.Builder
	sb.WriteString(prefix)
	sb.WriteString(Style.Header.Render("Auth"))
	sb.WriteString(" (1=none, 2=basic, 3=bearer, 4=apikey)")
	sb.WriteString("\n")

	authLabel := "None"
	authStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	switch rb.authType {
	case "basic":
		authLabel = "Basic"
		authStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("green"))
	case "bearer":
		authLabel = "Bearer"
		authStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("blue"))
	case "apikey":
		authLabel = "API Key"
		authStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("yellow"))
	}

	sb.WriteString("    Auth: ")
	sb.WriteString(authStyle.Render(authLabel))
	sb.WriteString("\n")

	// Show auth inputs based on type
	switch rb.authType {
	case "basic":
		sb.WriteString("    Username: ")
		sb.WriteString(rb.authInputs["username"].View())
		sb.WriteString("\n")
		sb.WriteString("    Password: ")
		sb.WriteString(rb.authInputs["password"].View())
		sb.WriteString("\n")
	case "bearer":
		sb.WriteString("    Token: ")
		sb.WriteString(rb.authInputs["token"].View())
		sb.WriteString("\n")
	case "apikey":
		sb.WriteString("    Key: ")
		sb.WriteString(rb.authInputs["api_key"].View())
		sb.WriteString(" Value: ")
		sb.WriteString(rb.authInputs["api_value"].View())
		sb.WriteString(" Header: ")
		sb.WriteString(rb.authInputs["api_header"].View())
		sb.WriteString("\n")
	}

	return sb.String()
}

func (rb *RequestBuilder) renderSend() string {
	selected := rb.activeSection == SectionSend
	prefix := "  "
	if selected {
		prefix = "▶ "
	}

	var sb strings.Builder

	if rb.sending {
		sb.WriteString(prefix)
		sb.WriteString(rb.loadingSpinner.View())
		sb.WriteString(" Sending...")
	} else {
		sb.WriteString(prefix)
		sb.WriteString(Style.Hint.Render("[Enter] Send  "))
		sb.WriteString(Style.Hint.Render("[Ctrl+S] Save  "))
		sb.WriteString(Style.Hint.Render("[Tab] Next Section"))
	}

	return sb.String()
}

// Init implements tea.Model.Init
func (rb *RequestBuilder) Init() tea.Cmd {
	return nil
}

// GetMessages returns any messages generated by the builder
func (rb *RequestBuilder) GetMessages() []tea.Msg {
	msgs := rb.msgs
	rb.msgs = []tea.Msg{}
	return msgs
}

// GetEditingRequest returns the currently editing request
func (rb *RequestBuilder) GetEditingRequest() *types.SavedRequest {
	rb.syncEditingFromForm()
	return rb.editing
}

// IsSending returns true if a request is currently being sent
func (rb *RequestBuilder) IsSending() bool {
	return rb.sending
}
