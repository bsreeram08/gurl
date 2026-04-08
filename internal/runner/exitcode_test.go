package runner

import (
	"errors"
	"testing"
)

func TestDetermineExitCode_Success(t *testing.T) {
	results := []RunResult{
		{Total: 3, Passed: 3, Failed: 0, Skipped: 0},
	}
	code := DetermineExitCode(results, nil, false)
	if code != ExitSuccess {
		t.Errorf("expected %d, got %d", ExitSuccess, code)
	}
}

func TestDetermineExitCode_AssertionFailure(t *testing.T) {
	results := []RunResult{
		{Total: 3, Passed: 2, Failed: 1, Skipped: 0},
	}
	code := DetermineExitCode(results, nil, false)
	if code != ExitAssertionFailure {
		t.Errorf("expected %d, got %d", ExitAssertionFailure, code)
	}
}

func TestDetermineExitCode_RuntimeError(t *testing.T) {
	code := DetermineExitCode(nil, errors.New("network timeout"), false)
	if code != ExitRuntimeError {
		t.Errorf("expected %d, got %d", ExitRuntimeError, code)
	}
}

func TestDetermineExitCode_EmptyCollection(t *testing.T) {
	err := &EmptyCollectionError{Collection: "my-api"}
	code := DetermineExitCode(nil, err, false)
	if code != ExitEmptyCollection {
		t.Errorf("expected %d, got %d", ExitEmptyCollection, code)
	}
}

func TestDetermineExitCode_CIMode_SkipsAreFailures(t *testing.T) {
	results := []RunResult{
		{Total: 3, Passed: 2, Failed: 0, Skipped: 1},
	}
	code := DetermineExitCode(results, nil, true)
	if code != ExitAssertionFailure {
		t.Errorf("expected %d in CI mode, got %d", ExitAssertionFailure, code)
	}
}

func TestDetermineExitCode_NonCIMode_SkipsAreOK(t *testing.T) {
	results := []RunResult{
		{Total: 3, Passed: 2, Failed: 0, Skipped: 1},
	}
	code := DetermineExitCode(results, nil, false)
	if code != ExitSuccess {
		t.Errorf("expected %d in non-CI mode, got %d", ExitSuccess, code)
	}
}

func TestEmptyCollectionError_Unwrap(t *testing.T) {
	err := &EmptyCollectionError{Collection: "test"}
	if !errors.Is(err, ErrEmptyCollection) {
		t.Error("expected errors.Is to match ErrEmptyCollection")
	}
}

func TestEmptyCollectionError_Message(t *testing.T) {
	err := &EmptyCollectionError{Collection: "my-col"}
	expected := "collection 'my-col' is empty or does not exist"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}
