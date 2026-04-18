package commands

import (
	"fmt"
	"os"
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

	absAllowed, err := filepath.Abs(allowedDir)
	if err != nil {
		return err
	}

	resolvedAllowed, err := filepath.EvalSymlinks(absAllowed)
	if err != nil {
		return fmt.Errorf("failed to resolve allowed directory: %w", err)
	}

	if !strings.HasSuffix(resolvedAllowed, string(os.PathSeparator)) {
		resolvedAllowed += string(os.PathSeparator)
	}

	resolvedPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		resolvedParent, parentErr := filepath.EvalSymlinks(filepath.Dir(absPath))
		if parentErr != nil {
			return fmt.Errorf("path escapes allowed directory: %s", path)
		}
		resolvedPath = filepath.Join(resolvedParent, filepath.Base(absPath))
	}

	if !strings.HasPrefix(resolvedPath, resolvedAllowed) {
		return fmt.Errorf("path escapes allowed directory: %s", path)
	}

	return nil
}
