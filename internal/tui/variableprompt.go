package tui

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbletea/v2"
)

// VariablePromptDoneMsg is sent when the variable prompt is dismissed.
type VariablePromptDoneMsg struct {
	Variables map[string]string
	Cancelled bool
}

// VariablePrompt prompts the user for missing template variables before sending.
type VariablePrompt struct {
	rqName   string
	varNames []string
	values   map[string]string
	active   int
	input    textinput.Model
	errMsg   string
	width    int
	height   int
}

// NewVariablePrompt creates a new variable prompt for the given request.
func NewVariablePrompt(requestName string, varNames []string, initialValues map[string]string) *VariablePrompt {
	input := textinput.New()
	input.Placeholder = "value"
	input.Prompt = ""

	values := make(map[string]string, len(varNames))
	for _, name := range varNames {
		if v, ok := initialValues[name]; ok {
			values[name] = v
		}
	}

	if len(varNames) > 0 {
		input.SetValue(values[varNames[0]])
	}

	return &VariablePrompt{
		rqName:   requestName,
		varNames: varNames,
		values:   values,
		active:   0,
		input:    input,
		width:    80,
		height:   24,
	}
}

// Init implements tea.Model.Init.
func (vp *VariablePrompt) Init() tea.Cmd {
	return nil
}

// Update handles key input for variable entry.
func (vp *VariablePrompt) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if vp == nil {
		return nil, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		vp.width = msg.Width
		vp.height = msg.Height
		return vp, nil

	case tea.KeyPressMsg:
		switch msg.Code {
		case tea.KeyUp:
			vp.commitActive()
			if vp.active > 0 {
				vp.active--
			}
			vp.restoreActive()
			return vp, nil
		case tea.KeyDown:
			vp.commitActive()
			if vp.active < len(vp.varNames)-1 {
				vp.active++
			}
			vp.restoreActive()
			return vp, nil
		case tea.KeyEnter:
			vp.commitActive()
			if vp.active < len(vp.varNames)-1 {
				vp.active++
				vp.restoreActive()
				return vp, nil
			}
			return vp, vp.finish(false)
		case tea.KeyEscape:
			return nil, vp.finish(true)
		}

		var cmd tea.Cmd
		vp.input, cmd = vp.input.Update(msg)
		if cmd != nil {
			return vp, cmd
		}
	}

	return vp, nil
}

// View renders the variable prompt.
func (vp *VariablePrompt) View() tea.View {
	return vp.renderView()
}

// GetMessages returns no queue messages.
func (vp *VariablePrompt) GetMessages() []tea.Msg {
	return nil
}

// finish returns a command that emits VariablePromptDoneMsg.
func (vp *VariablePrompt) finish(cancelled bool) tea.Cmd {
	if vp == nil {
		return nil
	}

	if cancelled {
		return func() tea.Msg {
			return VariablePromptDoneMsg{Variables: nil, Cancelled: true}
		}
	}

	missing := make([]string, 0)
	for _, name := range vp.varNames {
		if value := strings.TrimSpace(vp.values[name]); value == "" {
			missing = append(missing, name)
		}
	}

	if len(missing) > 0 {
		vp.errMsg = "Missing values: " + strings.Join(missing, ", ")
		return nil
	}

	result := make(map[string]string, len(vp.values))
	for k, v := range vp.values {
		result[k] = v
	}

	return func() tea.Msg {
		return VariablePromptDoneMsg{Variables: result, Cancelled: false}
	}
}

func (vp *VariablePrompt) commitActive() {
	if vp == nil || len(vp.varNames) == 0 {
		return
	}
	name := vp.varNames[vp.active]
	vp.values[name] = vp.input.Value()
	vp.errMsg = ""
}

func (vp *VariablePrompt) restoreActive() {
	if vp == nil || len(vp.varNames) == 0 {
		return
	}
	vp.errMsg = ""
	name := vp.varNames[vp.active]
	vp.input.SetValue(vp.values[name])
}

func (vp *VariablePrompt) renderView() tea.View {
	var b strings.Builder

	b.WriteString(Style.Header.Render("Request Variables"))
	b.WriteString("\n")
	if vp.rqName != "" {
		b.WriteString(Style.PlainText.Render("Request: " + vp.rqName))
		b.WriteString("\n")
	}

	if len(vp.varNames) == 0 {
		b.WriteString(Style.PlainText.Render("No variables found."))
		b.WriteString("\n")
		b.WriteString(Style.Hint.Render("  Enter: submit  Esc: cancel"))
		return makeView(Style.Modal.Width(max(30, vp.width-20)).Height(max(8, vp.height-8)).Render(b.String()))
	}

	b.WriteString("\n")
	for i, name := range vp.varNames {
		prefix := "  "
		if i == vp.active {
			prefix = "▶ "
		}
		line := name
		if value := vp.values[name]; value != "" {
			line += " = " + value
		}
		b.WriteString(prefix + line)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	if vp.active < len(vp.varNames) {
		activeName := vp.varNames[vp.active]
		b.WriteString("Enter value for {{" + activeName + "}}")
		b.WriteString("\n")
		b.WriteString(vp.input.View())
		b.WriteString("\n")
	}

	if vp.errMsg != "" {
		b.WriteString(Style.Hint.Render("  " + vp.errMsg))
		b.WriteString("\n")
	}

	help := "  ↑/↓ move  Enter: next/submit  Esc: cancel"
	b.WriteString(Style.Hint.Render(help))

	width := max(64, vp.width-20)
	height := max(12, vp.height-8)
	return makeView(Style.Modal.Width(width).Height(height).Render(b.String()))
}
