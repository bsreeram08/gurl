package runner

import (
	"errors"
	"fmt"
)

// ExitCode represents a process exit code for CI integration.
type ExitCode int

const (
	ExitSuccess          ExitCode = 0
	ExitAssertionFailure ExitCode = 1
	ExitRuntimeError     ExitCode = 2
	ExitEmptyCollection  ExitCode = 3
	ExitScriptError      ExitCode = 4
)

// ErrEmptyCollection is the sentinel error for an empty or missing collection.
var ErrEmptyCollection = errors.New("empty collection")

// EmptyCollectionError wraps ErrEmptyCollection with the collection name.
type EmptyCollectionError struct {
	Collection string
}

func (e *EmptyCollectionError) Error() string {
	return fmt.Sprintf("collection '%s' is empty or does not exist", e.Collection)
}

func (e *EmptyCollectionError) Unwrap() error {
	return ErrEmptyCollection
}

// DetermineExitCode maps runner results and errors to an exit code.
// When ciMode is true, skipped requests are treated as failures (exit 1).
func DetermineExitCode(results []RunResult, err error, ciMode bool) ExitCode {
	if err != nil {
		if errors.Is(err, ErrEmptyCollection) {
			return ExitEmptyCollection
		}
		return ExitRuntimeError
	}

	for _, result := range results {
		if result.Failed > 0 {
			return ExitAssertionFailure
		}
		if ciMode && result.Skipped > 0 {
			return ExitAssertionFailure
		}
	}

	return ExitSuccess
}
