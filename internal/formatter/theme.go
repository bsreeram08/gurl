package formatter

// ANSI color codes for syntax highlighting
// These are extracted to a separate file for future TUI reuse

const (
	// Reset resets all colors and attributes
	Reset = "\033[0m"

	// Black text color
	Black = "\033[30m"
	// Red text color
	Red = "\033[31m"
	// Green text color
	Green = "\033[32m"
	// Yellow text color
	Yellow = "\033[33m"
	// Blue text color
	Blue = "\033[34m"
	// Magenta text color
	Magenta = "\033[35m"
	// Cyan text color
	Cyan = "\033[36m"
	// White text color
	White = "\033[37m"

	// Bold modifiers
	Bold         = "\033[1m"
	BoldOff      = "\033[21m"
	Dim          = "\033[2m"
	DimOff       = "\033[22m"
	Italic       = "\033[3m"
	ItalicOff    = "\033[23m"
	Underline    = "\033[4m"
	UnderlineOff = "\033[24m"
)

// ColorScheme defines a color scheme for syntax highlighting
type ColorScheme struct {
	Keyword   string
	String    string
	Number    string
	Boolean   string
	Null      string
	Comment   string
	Tag       string
	Attribute string
	Plaintext string
}

// DefaultColorScheme is the default color scheme for code formatting
var DefaultColorScheme = ColorScheme{
	Keyword:   Cyan,
	String:    Green,
	Number:    Yellow,
	Boolean:   Magenta,
	Null:      Red,
	Comment:   Dim + White,
	Tag:       Cyan,
	Attribute: Yellow,
}
