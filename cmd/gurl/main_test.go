package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunSupportsShellCompletion(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GURL_DB_PATH", filepath.Join(t.TempDir(), "gurl.db"))
	t.Setenv("GURL_PLUGIN_DIR", t.TempDir())

	for _, shell := range []string{"bash", "zsh", "fish"} {
		t.Run(shell, func(t *testing.T) {
			stdout, stderr, err := runWithArgs(t, []string{"gurl", "completion", shell})
			if err != nil {
				t.Fatalf("expected completion to run for %s: %v\nstderr: %s", shell, err, stderr)
			}
			if strings.TrimSpace(stdout) == "" {
				t.Fatalf("expected %s completion output", shell)
			}
		})
	}
}

func runWithArgs(t *testing.T, args []string) (string, string, error) {
	t.Helper()

	oldArgs := os.Args
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	defer func() {
		os.Args = oldArgs
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	defer stdoutR.Close()

	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stderr pipe: %v", err)
	}
	defer stderrR.Close()

	os.Args = args
	os.Stdout = stdoutW
	os.Stderr = stderrW

	runErr := run()

	if err := stdoutW.Close(); err != nil {
		t.Fatalf("close stdout pipe: %v", err)
	}
	if err := stderrW.Close(); err != nil {
		t.Fatalf("close stderr pipe: %v", err)
	}

	stdout, err := io.ReadAll(stdoutR)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	stderr, err := io.ReadAll(stderrR)
	if err != nil {
		t.Fatalf("read stderr: %v", err)
	}

	return string(stdout), string(stderr), runErr
}
