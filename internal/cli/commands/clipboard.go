package commands

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// errNoClipboard means none of the supported clipboard tools is installed.
var errNoClipboard = errors.New("clipboard tools not available. Install xclip (Linux) or wl-copy (Wayland) to enable clipboard copy")

// clipboardWrite copies text to the system clipboard. Every clipboard write
// (paste, codegen --clipboard) goes through it, so tests can replace it with
// a fake and never touch the developer's clipboard.
var clipboardWrite = copyToClipboard

// copyToClipboard pipes text into the first available clipboard tool:
// pbcopy (macOS), xclip (X11) or wl-copy (Wayland). It returns
// errNoClipboard when none is installed.
func copyToClipboard(text string) error {
	var cmd *exec.Cmd
	switch {
	case isCommandAvailable("pbcopy"):
		cmd = exec.Command("pbcopy")
	case isCommandAvailable("xclip"):
		cmd = exec.Command("xclip", "-selection", "clipboard")
	case isCommandAvailable("wl-copy"):
		cmd = exec.Command("wl-copy")
	default:
		return errNoClipboard
	}
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running %s: %w", cmd.Path, err)
	}
	return nil
}

// isCommandAvailable checks if a command is available in PATH
func isCommandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
