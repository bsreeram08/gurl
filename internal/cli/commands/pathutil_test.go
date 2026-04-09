package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateSafePath_Basic(t *testing.T) {
	tmpDir := t.TempDir()

	filePath := filepath.Join(tmpDir, "file.txt")
	allowedDir := tmpDir

	absPath, _ := filepath.Abs(filePath)
	absAllowed, _ := filepath.Abs(allowedDir)

	resolvedPath, _ := filepath.EvalSymlinks(absPath)
	resolvedAllowed, _ := filepath.EvalSymlinks(absAllowed)

	t.Logf("absPath=%s", absPath)
	t.Logf("absAllowed=%s", absAllowed)
	t.Logf("resolvedPath=%s", resolvedPath)
	t.Logf("resolvedAllowed=%s", resolvedAllowed)
	t.Logf("HasPrefix: %v", strings.HasPrefix(resolvedPath, resolvedAllowed+"/"))

	f, _ := os.Create(filePath)
	f.Close()

	err := ValidateSafePath(filePath, allowedDir)
	if err != nil {
		t.Errorf("expected valid path, got error: %v", err)
	}
}

func TestValidateSafePath_DoubleDot(t *testing.T) {
	tmpDir := t.TempDir()

	err := ValidateSafePath(filepath.Join(tmpDir, "..", "file.txt"), tmpDir)
	if err == nil {
		t.Error("expected error for path with .., got nil")
	}
}

func TestValidateSafePath_Symlink(t *testing.T) {
	tmpDir := t.TempDir()

	parentDir := filepath.Dir(tmpDir)
	evilLink := filepath.Join(tmpDir, "evil")
	os.Symlink(parentDir, evilLink)

	fileInsideLink := filepath.Join(evilLink, "file.txt")
	err := ValidateSafePath(fileInsideLink, tmpDir)
	if err == nil {
		t.Error("expected error for symlink escape, got nil")
	}
}
