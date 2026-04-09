package commands

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ValidateSafePath checks that the given path is safe (contained within the
// allowed directory). It resolves symlinks and returns an error if the path
// escapes the allowed directory via ".." or symlinks.
func ValidateSafePath(path string, allowedDir string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	resolvedPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		resolvedPath = absPath
	}

	absAllowed, err := filepath.Abs(allowedDir)
	if err != nil {
		return err
	}
	resolvedAllowed, err := filepath.EvalSymlinks(absAllowed)
	if err != nil {
		resolvedAllowed = absAllowed
	}

	if !strings.HasSuffix(resolvedAllowed, string(filepath.Separator)) {
		resolvedAllowed += string(filepath.Separator)
	}
	if !strings.HasPrefix(resolvedPath, resolvedAllowed) {
		return fmt.Errorf("path escapes allowed directory: %s", path)
	}

	return nil
}
