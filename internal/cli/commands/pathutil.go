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

	// First: check the unresolved path for ".." escapes.
	if !strings.HasPrefix(absPath, resolvedAllowed) {
		return fmt.Errorf("path escapes allowed directory: %s", path)
	}

	// Second: attempt to resolve symlinks and verify the resolved path stays
	// within the allowed directory. EvalSymlinks returns an error if the final
	// target doesn't exist; in that case we additionally resolve the parent
	// directory to catch symlink escapes where only the leaf file is missing.
	resolvedPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		// File doesn't exist — resolve the parent directory via symlinks to
		// check whether the symlink chain escapes, even though the leaf is absent.
		resolvedParent, parentErr := filepath.EvalSymlinks(filepath.Dir(absPath))
		if parentErr != nil {
			// Parent can't be resolved — likely a broken symlink; reject.
			return fmt.Errorf("path escapes allowed directory: %s", path)
		}
		resolvedPath = filepath.Join(resolvedParent, filepath.Base(absPath))
	}

	if !strings.HasPrefix(resolvedPath, resolvedAllowed) {
		return fmt.Errorf("path escapes allowed directory: %s", path)
	}

	return nil
}
