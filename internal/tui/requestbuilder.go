package tui

import (
	"net/url"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
	width           int
	height          int
	activeSection   Section // Currently focused section
	activeHeaderRow int
	activeQueryRow  int
	activeAuthField int

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
		db:              db,
		methods:         methods,
		methodIndex:     0,
		activeSection:   SectionURL,
		activeHeaderRow: 0,
		activeQueryRow:  0,
		activeAuthField: 0,
		urlInput:        urlInput,
		bodyInput:       bodyInput,
		contentType:     "json",
		authType:        "none",
		authInputs:      authInputs,
		headerInputs:    []headerRow{},
		queryInputs:     []queryRow{},
		loadingSpinner:  loadingSpinner,
	}

	// Add one empty header row by default
	rb.addHeaderRow()
	rb.addQueryRow()
	rb.syncFocus()

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
	rb.activeHeaderRow = 0
	for _, h := range req.Headers {
		row := rb.addHeaderRow()
		row.keyInput.SetValue(h.Key)
		row.valueInput.SetValue(h.Value)
	}
	rb.ensureTrailingHeaderRow()

	// Set query params from the request URL
	rb.queryInputs = []queryRow{}
	rb.activeQueryRow = 0
	rb.loadQueryParams(req.URL)

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
	rb.activeSection = SectionURL
	rb.activeAuthField = 0
	rb.syncFocus()
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
	rb.queryInputs = []queryRow{}
	rb.addQueryRow()
	rb.authType = "none"
	rb.contentType = "json"
	rb.activeSection = SectionURL
	rb.activeHeaderRow = 0
	rb.activeQueryRow = 0
	rb.activeAuthField = 0
	rb.syncFocus()
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
	rb.syncFocus()
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
	rb.syncFocus()
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
	if rb.activeHeaderRow >= len(rb.headerInputs) {
		rb.activeHeaderRow = len(rb.headerInputs) - 1
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
	if rb.activeQueryRow >= len(rb.queryInputs) {
		rb.activeQueryRow = len(rb.queryInputs) - 1
	}
}

// Update implements tea.Model.Update
func (rb *RequestBuilder) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		rb.SetSize(msg.Width, msg.Height)
		return rb, nil

	case tea.KeyPressMsg:
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

// SetSize updates the editor viewport dimensions.
func (rb *RequestBuilder) SetSize(width, height int) {
	rb.width = width
	rb.height = height

	bodyWidth := max(24, width-8)
	bodyHeight := max(8, height/3)

	rb.bodyInput.SetWidth(bodyWidth)
	rb.bodyInput.SetHeight(bodyHeight)
}

// handleKeyPress handles keyboard input
func (rb *RequestBuilder) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	keyStr := msg.String()

	// Send from anywhere in the editor.
	if keyStr == "ctrl+j" || keyStr == "ctrl+enter" {
		return rb, rb.sendRequest()
	}

	// Ctrl+C cancels (but doesn't quit)
	if keyStr == "ctrl+c" {
		return rb, nil
	}

	// Ctrl+S saves request
	if keyStr == "ctrl+s" {
		return rb, rb.saveRequest()
	}

	// Tab navigates sections
	if keyStr == "tab" {
		rb.nextSection()
		return rb, nil
	}

	if keyStr == "shift+tab" {
		rb.prevSection()
		return rb, nil
	}

	// Section-specific key handling
	switch rb.activeSection {
	case SectionMethod:
		return rb.handleMethodKey(msg)

	case SectionURL:
		if rb.showSuggestions {
			switch msg.Code {
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
		if msg.Code == tea.KeyEnter {
			return rb, rb.sendRequest()
		}
		return rb, nil
	}

	return rb, nil
}

func (rb *RequestBuilder) handleMethodKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// 'g' then 'm' cycles methods (g then m shortcut)
	if len(msg.Text) > 0 {
		runes := msg.Text
		if runes == "g" || runes == "G" {
			// Start of g+m shortcut, do nothing yet
			return rb, nil
		}
		if runes == "m" || runes == "M" {
			// Cycle to next method
			rb.methodIndex = (rb.methodIndex + 1) % len(rb.methods)
			return rb, nil
		}
	}
	switch msg.Code {
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

func (rb *RequestBuilder) handleHeadersKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.Code == tea.KeyEscape {
		rb.nextSection()
		return rb, nil
	}

	switch msg.Code {
	case tea.KeyUp:
		if rb.activeHeaderRow > 0 {
			rb.focusHeaderCell(rb.activeHeaderRow-1, rb.headerInputs[rb.activeHeaderRow].focused)
		}
		return rb, nil
	case tea.KeyDown:
		if rb.activeHeaderRow < len(rb.headerInputs)-1 {
			rb.focusHeaderCell(rb.activeHeaderRow+1, rb.headerInputs[rb.activeHeaderRow].focused)
		}
		return rb, nil
	case tea.KeyLeft:
		rb.focusHeaderCell(rb.activeHeaderRow, true)
		return rb, nil
	case tea.KeyRight:
		rb.focusHeaderCell(rb.activeHeaderRow, false)
		return rb, nil
	case tea.KeyEnter:
		if rb.headerInputs[rb.activeHeaderRow].focused {
			rb.focusHeaderCell(rb.activeHeaderRow, false)
			return rb, nil
		}
		if rb.activeHeaderRow == len(rb.headerInputs)-1 {
			rb.addHeaderRow()
		}
		rb.focusHeaderCell(min(rb.activeHeaderRow+1, len(rb.headerInputs)-1), true)
		return rb, nil
	case tea.KeyBackspace:
		if rb.isHeaderRowEmpty(rb.activeHeaderRow) && len(rb.headerInputs) > 1 {
			current := rb.activeHeaderRow
			rb.removeHeaderRow(current)
			rb.focusHeaderCell(max(0, min(current, len(rb.headerInputs)-1)), true)
			return rb, nil
		}
	}

	var cmd tea.Cmd
	if rb.headerInputs[rb.activeHeaderRow].focused {
		rb.headerInputs[rb.activeHeaderRow].keyInput, cmd = rb.headerInputs[rb.activeHeaderRow].keyInput.Update(msg)
	} else {
		rb.headerInputs[rb.activeHeaderRow].valueInput, cmd = rb.headerInputs[rb.activeHeaderRow].valueInput.Update(msg)
	}
	if cmd != nil {
		return rb, cmd
	}

	rb.ensureTrailingHeaderRow()
	return rb, nil
}

func (rb *RequestBuilder) handleQueryParamsKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.Code == tea.KeyEscape {
		rb.nextSection()
		return rb, nil
	}

	switch msg.Code {
	case tea.KeyUp:
		if rb.activeQueryRow > 0 {
			rb.focusQueryCell(rb.activeQueryRow-1, rb.queryInputs[rb.activeQueryRow].focused)
		}
		return rb, nil
	case tea.KeyDown:
		if rb.activeQueryRow < len(rb.queryInputs)-1 {
			rb.focusQueryCell(rb.activeQueryRow+1, rb.queryInputs[rb.activeQueryRow].focused)
		}
		return rb, nil
	case tea.KeyLeft:
		rb.focusQueryCell(rb.activeQueryRow, true)
		return rb, nil
	case tea.KeyRight:
		rb.focusQueryCell(rb.activeQueryRow, false)
		return rb, nil
	case tea.KeyEnter:
		if rb.queryInputs[rb.activeQueryRow].focused {
			rb.focusQueryCell(rb.activeQueryRow, false)
			return rb, nil
		}
		if rb.activeQueryRow == len(rb.queryInputs)-1 {
			rb.addQueryRow()
		}
		rb.focusQueryCell(min(rb.activeQueryRow+1, len(rb.queryInputs)-1), true)
		return rb, nil
	case tea.KeyBackspace:
		if rb.isQueryRowEmpty(rb.activeQueryRow) && len(rb.queryInputs) > 1 {
			current := rb.activeQueryRow
			rb.removeQueryRow(current)
			rb.focusQueryCell(max(0, min(current, len(rb.queryInputs)-1)), true)
			return rb, nil
		}
	}

	var cmd tea.Cmd
	if rb.queryInputs[rb.activeQueryRow].focused {
		rb.queryInputs[rb.activeQueryRow].keyInput, cmd = rb.queryInputs[rb.activeQueryRow].keyInput.Update(msg)
	} else {
		rb.queryInputs[rb.activeQueryRow].valueInput, cmd = rb.queryInputs[rb.activeQueryRow].valueInput.Update(msg)
	}
	if cmd != nil {
		return rb, cmd
	}

	rb.ensureTrailingQueryRow()
	return rb, nil
}

func (rb *RequestBuilder) handleAuthKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.Code == tea.KeyEscape {
		rb.nextSection()
		return rb, nil
	}

	switch msg.String() {
	case "[":
		rb.cycleAuthType(-1)
		return rb, nil
	case "]":
		rb.cycleAuthType(1)
		return rb, nil
	}

	keys := rb.currentAuthKeys()
	if len(keys) == 0 {
		return rb, nil
	}

	switch msg.Code {
	case tea.KeyUp:
		if rb.activeAuthField > 0 {
			rb.activeAuthField--
			rb.syncFocus()
		}
		return rb, nil
	case tea.KeyDown:
		if rb.activeAuthField < len(keys)-1 {
			rb.activeAuthField++
			rb.syncFocus()
		}
		return rb, nil
	}

	key := keys[rb.activeAuthField]
	input := rb.authInputs[key]
	var cmd tea.Cmd
	input, cmd = input.Update(msg)
	rb.authInputs[key] = input
	if cmd != nil {
		return rb, cmd
	}

	return rb, nil
}

func (rb *RequestBuilder) syncFocus() {
	rb.urlInput.Blur()
	rb.bodyInput.Blur()

	for i := range rb.headerInputs {
		rb.headerInputs[i].keyInput.Blur()
		rb.headerInputs[i].valueInput.Blur()
	}
	for i := range rb.queryInputs {
		rb.queryInputs[i].keyInput.Blur()
		rb.queryInputs[i].valueInput.Blur()
	}
	for key, input := range rb.authInputs {
		input.Blur()
		rb.authInputs[key] = input
	}

	switch rb.activeSection {
	case SectionURL:
		rb.urlInput.Focus()
	case SectionHeaders:
		rb.ensureTrailingHeaderRow()
		rb.focusHeaderCell(rb.activeHeaderRow, rb.headerInputs[rb.activeHeaderRow].focused)
	case SectionQueryParams:
		rb.ensureTrailingQueryRow()
		rb.focusQueryCell(rb.activeQueryRow, rb.queryInputs[rb.activeQueryRow].focused)
	case SectionBody:
		rb.bodyInput.Focus()
	case SectionAuth:
		rb.activeAuthField = max(0, min(rb.activeAuthField, len(rb.currentAuthKeys())-1))
		rb.focusAuthField(rb.activeAuthField)
	}
}

func (rb *RequestBuilder) focusHeaderCell(row int, key bool) {
	if len(rb.headerInputs) == 0 {
		rb.addHeaderRow()
	}
	row = max(0, min(row, len(rb.headerInputs)-1))
	rb.activeHeaderRow = row
	for i := range rb.headerInputs {
		rb.headerInputs[i].keyInput.Blur()
		rb.headerInputs[i].valueInput.Blur()
		rb.headerInputs[i].focused = key
	}
	rb.headerInputs[row].focused = key
	if key {
		rb.headerInputs[row].keyInput.Focus()
	} else {
		rb.headerInputs[row].valueInput.Focus()
	}
}

func (rb *RequestBuilder) focusQueryCell(row int, key bool) {
	if len(rb.queryInputs) == 0 {
		rb.addQueryRow()
	}
	row = max(0, min(row, len(rb.queryInputs)-1))
	rb.activeQueryRow = row
	for i := range rb.queryInputs {
		rb.queryInputs[i].keyInput.Blur()
		rb.queryInputs[i].valueInput.Blur()
		rb.queryInputs[i].focused = key
	}
	rb.queryInputs[row].focused = key
	if key {
		rb.queryInputs[row].keyInput.Focus()
	} else {
		rb.queryInputs[row].valueInput.Focus()
	}
}

func (rb *RequestBuilder) focusAuthField(index int) {
	keys := rb.currentAuthKeys()
	if len(keys) == 0 {
		return
	}
	index = max(0, min(index, len(keys)-1))
	rb.activeAuthField = index

	for key, input := range rb.authInputs {
		input.Blur()
		rb.authInputs[key] = input
	}

	activeKey := keys[index]
	input := rb.authInputs[activeKey]
	input.Focus()
	rb.authInputs[activeKey] = input
}

func (rb *RequestBuilder) currentAuthKeys() []string {
	switch rb.authType {
	case "basic":
		return []string{"username", "password"}
	case "bearer":
		return []string{"token"}
	case "apikey":
		return []string{"api_key", "api_value", "api_header"}
	default:
		return nil
	}
}

func (rb *RequestBuilder) cycleAuthType(delta int) {
	authTypes := []string{"none", "basic", "bearer", "apikey"}
	current := 0
	for i, authType := range authTypes {
		if authType == rb.authType {
			current = i
			break
		}
	}
	current = (current + delta + len(authTypes)) % len(authTypes)
	rb.authType = authTypes[current]
	rb.activeAuthField = 0
	rb.syncFocus()
}

func (rb *RequestBuilder) ensureTrailingHeaderRow() {
	if len(rb.headerInputs) == 0 {
		rb.addHeaderRow()
	}

	for len(rb.headerInputs) > 1 && rb.isHeaderRowEmpty(len(rb.headerInputs)-1) && rb.isHeaderRowEmpty(len(rb.headerInputs)-2) {
		rb.headerInputs = rb.headerInputs[:len(rb.headerInputs)-1]
	}

	last := rb.headerInputs[len(rb.headerInputs)-1]
	if last.keyInput.Value() != "" || last.valueInput.Value() != "" {
		rb.addHeaderRow()
	}

	if rb.activeHeaderRow >= len(rb.headerInputs) {
		rb.activeHeaderRow = len(rb.headerInputs) - 1
	}
}

func (rb *RequestBuilder) ensureTrailingQueryRow() {
	if len(rb.queryInputs) == 0 {
		rb.addQueryRow()
	}

	for len(rb.queryInputs) > 1 && rb.isQueryRowEmpty(len(rb.queryInputs)-1) && rb.isQueryRowEmpty(len(rb.queryInputs)-2) {
		rb.queryInputs = rb.queryInputs[:len(rb.queryInputs)-1]
	}

	last := rb.queryInputs[len(rb.queryInputs)-1]
	if last.keyInput.Value() != "" || last.valueInput.Value() != "" {
		rb.addQueryRow()
	}

	if rb.activeQueryRow >= len(rb.queryInputs) {
		rb.activeQueryRow = len(rb.queryInputs) - 1
	}
}

func (rb *RequestBuilder) isHeaderRowEmpty(index int) bool {
	if index < 0 || index >= len(rb.headerInputs) {
		return true
	}
	row := rb.headerInputs[index]
	return strings.TrimSpace(row.keyInput.Value()) == "" && strings.TrimSpace(row.valueInput.Value()) == ""
}

func (rb *RequestBuilder) isQueryRowEmpty(index int) bool {
	if index < 0 || index >= len(rb.queryInputs) {
		return true
	}
	row := rb.queryInputs[index]
	return strings.TrimSpace(row.keyInput.Value()) == "" && strings.TrimSpace(row.valueInput.Value()) == ""
}

func (rb *RequestBuilder) loadQueryParams(rawURL string) {
	if parsed, err := url.Parse(rawURL); err == nil {
		values := parsed.Query()
		for key, list := range values {
			if len(list) == 0 {
				row := rb.addQueryRow()
				row.keyInput.SetValue(key)
				continue
			}
			for _, value := range list {
				row := rb.addQueryRow()
				row.keyInput.SetValue(key)
				row.valueInput.SetValue(value)
			}
		}
	}
	rb.ensureTrailingQueryRow()
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

	// Sync query params back into the URL.
	if parsed, err := url.Parse(rb.urlInput.Value()); err == nil {
		values := url.Values{}
		for _, row := range rb.queryInputs {
			key := strings.TrimSpace(row.keyInput.Value())
			if key == "" {
				continue
			}
			values.Add(key, row.valueInput.Value())
		}
		parsed.RawQuery = values.Encode()
		rb.editing.URL = parsed.String()
	}

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
	if err != nil {
		// Log DB error but don't show anything to user - suggestions are non-critical
		rb.suggestions = nil
		return
	}
	if len(requests) == 0 {
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

// View implements tea.Model.View.
func (rb *RequestBuilder) View() tea.View {
	var content string
	if rb.editing == nil {
		content = rb.welcomeView()
	} else {
		sections := []string{
			rb.renderMethodURL(),
			rb.renderActionRow(),
			rb.renderEditorTabs(),
		}

		switch rb.activeSurface() {
		case "query":
			sections = append(sections, rb.renderQueryParams())
		case "auth":
			sections = append(sections, rb.renderAuth())
		default:
			sections = append(sections, rb.renderHeaders(), rb.renderBody())
		}

		sections = append(sections, rb.renderSend())
		content = strings.Join(sections, "\n\n")
	}
	return tea.NewView(content)
}

func (rb *RequestBuilder) activeSurface() string {
	switch rb.activeSection {
	case SectionQueryParams:
		return "query"
	case SectionAuth:
		return "auth"
	default:
		return "overview"
	}
}

func (rb *RequestBuilder) welcomeView() string {
	lines := []string{
		Style.WelcomeText.Render("Start with a saved request or a blank editor"),
		Style.PlainText.Render("Use Ctrl+1 to browse requests and Ctrl+2 to return here."),
		Style.Hint.Render("Ctrl+T opens a fresh tab. Ctrl+K searches across saved requests."),
	}
	return strings.Join(lines, "\n\n")
}

func (rb *RequestBuilder) renderMethodURL() string {
	method := RenderMethodBadge(rb.methods[rb.methodIndex])
	urlWidth := max(18, rb.width-lipgloss.Width(method)-6)

	urlView := rb.urlInput.View()
	if rb.activeSection != SectionURL {
		if strings.TrimSpace(rb.urlInput.Value()) == "" {
			urlView = Style.Hint.Render("https://api.example.com/v1/resource")
		} else {
			urlView = Style.URL.Render(truncateText(rb.urlInput.Value(), urlWidth))
		}
	}

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Center, method, " ", lipgloss.NewStyle().Width(urlWidth).Render(urlView)),
		Style.Hint.Render("Shift+Tab method  ·  Tab headers  ·  Ctrl+Enter send"),
	)

	return SectionStyle(rb.activeSection == SectionMethod || rb.activeSection == SectionURL).
		Width(max(24, rb.width)).
		Render(content)
}

func (rb *RequestBuilder) renderActionRow() string {
	title := "New Request"
	if rb.editing != nil && rb.editing.Name != "" {
		title = rb.editing.Name
	}

	sendLabel := "Send (Ctrl+Enter)"
	if rb.sending {
		sendLabel = rb.loadingSpinner.View() + " Sending"
	}

	left := Style.PlainText.Copy().Bold(true).Render(truncateText(title, max(12, rb.width-34)))
	right := lipgloss.JoinHorizontal(
		lipgloss.Center,
		RenderActionButton("Save", false, false),
		" ",
		RenderActionButton(sendLabel, true, rb.activeSection == SectionSend || rb.sending),
	)

	gap := rb.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}

	return left + strings.Repeat(" ", gap) + right
}

func (rb *RequestBuilder) renderEditorTabs() string {
	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		RenderMiniTab("Overview", rb.activeSurface() == "overview"),
		"  ",
		RenderMiniTab("Query", rb.activeSurface() == "query"),
		"  ",
		RenderMiniTab("Auth", rb.activeSurface() == "auth"),
	)
}

func (rb *RequestBuilder) renderHeaders() string {
	lines := []string{Style.Header.Render("HEADERS")}
	for i, row := range rb.headerInputs {
		prefix := "  "
		if rb.activeSection == SectionHeaders && i == rb.activeHeaderRow {
			prefix = "▶ "
		}

		keyView := row.keyInput.View()
		valueView := row.valueInput.View()
		lines = append(lines, prefix+keyView+"  "+Style.Hint.Render("->")+"  "+valueView)
	}

	lines = append(lines, Style.Hint.Render("Enter next cell  ·  ↑/↓ rows  ·  ←/→ key/value"))
	return SectionStyle(rb.activeSection == SectionHeaders).
		Width(max(24, rb.width)).
		Render(strings.Join(lines, "\n"))
}

func (rb *RequestBuilder) renderQueryParams() string {
	lines := []string{Style.Header.Render("QUERY PARAMS")}
	for i, row := range rb.queryInputs {
		prefix := "  "
		if rb.activeSection == SectionQueryParams && i == rb.activeQueryRow {
			prefix = "▶ "
		}

		keyView := row.keyInput.View()
		valueView := row.valueInput.View()
		lines = append(lines, prefix+keyView+"  "+Style.Hint.Render("=")+"  "+valueView)
	}

	lines = append(lines, Style.Hint.Render("Enter next row  ·  ↑/↓ rows  ·  Esc auth"))
	return SectionStyle(rb.activeSection == SectionQueryParams).
		Width(max(24, rb.width)).
		Render(strings.Join(lines, "\n"))
}

func (rb *RequestBuilder) renderBody() string {
	formats := lipgloss.JoinHorizontal(
		lipgloss.Center,
		RenderMiniTab("JSON", rb.contentType == "json"),
		"  ",
		RenderMiniTab("XML", rb.contentType == "xml"),
		"  ",
		RenderMiniTab("FORM", rb.contentType == "form"),
		"  ",
		RenderMiniTab("RAW", rb.contentType == "raw"),
	)

	header := RenderPanelHeader("BODY", formats, max(12, rb.width-4))
	content := header + "\n" + rb.bodyInput.View()

	return SectionStyle(rb.activeSection == SectionBody).
		Width(max(24, rb.width)).
		Render(content)
}

func (rb *RequestBuilder) renderAuth() string {
	authTabs := lipgloss.JoinHorizontal(
		lipgloss.Center,
		RenderMiniTab("None", rb.authType == "none"),
		"  ",
		RenderMiniTab("Basic", rb.authType == "basic"),
		"  ",
		RenderMiniTab("Bearer", rb.authType == "bearer"),
		"  ",
		RenderMiniTab("API Key", rb.authType == "apikey"),
	)

	lines := []string{
		RenderPanelHeader("AUTH", authTabs, max(12, rb.width-4)),
	}

	keys := rb.currentAuthKeys()
	if len(keys) == 0 {
		lines = append(lines, Style.Hint.Render("No auth applied. Use [ or ] to choose a strategy."))
	} else {
		for i, key := range keys {
			label := strings.ReplaceAll(strings.ReplaceAll(key, "_", " "), "api ", "API ")
			prefix := "  "
			if rb.activeAuthField == i {
				prefix = "▶ "
			}
			lines = append(lines, prefix+strings.Title(label)+": "+rb.authInputs[key].View())
		}
	}

	lines = append(lines, Style.Hint.Render("[ / ] auth type  ·  ↑/↓ fields  ·  Esc send"))
	return SectionStyle(rb.activeSection == SectionAuth).
		Width(max(24, rb.width)).
		Render(strings.Join(lines, "\n"))
}

func (rb *RequestBuilder) renderSend() string {
	return Style.Hint.Render("Ctrl+1 requests  ·  Ctrl+2 editor  ·  Ctrl+3 response  ·  Tab sections")
}

// Init implements tea.Model.Init
func (rb *RequestBuilder) Init() tea.Cmd {
	return nil
}

// GetMessages returns nil since the msgs field was removed
func (rb *RequestBuilder) GetMessages() []tea.Msg {
	return nil
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
