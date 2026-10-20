package commands

import (
	"errors"
	"os"
	"testing"
)

// errRealClipboard is what clipboardWrite returns in tests that did not
// install a fake with fakeClipboard.
var errRealClipboard = errors.New("test reached the real clipboard; use fakeClipboard(t)")

func TestMain(m *testing.M) {
	// No test may overwrite the developer's clipboard: a test that copies
	// without fakeClipboard gets an error instead of running pbcopy, xclip
	// or wl-copy.
	clipboardWrite = func(string) error { return errRealClipboard }
	os.Exit(m.Run())
}

// clipboardFake records what was copied to the clipboard.
type clipboardFake struct {
	writes []string
	err    error
}

// fakeClipboard replaces clipboardWrite for the duration of the test. When
// err is set, every write fails with it.
func fakeClipboard(t *testing.T, err error) *clipboardFake {
	t.Helper()
	fake := &clipboardFake{err: err}
	prev := clipboardWrite
	clipboardWrite = func(text string) error {
		fake.writes = append(fake.writes, text)
		return fake.err
	}
	t.Cleanup(func() { clipboardWrite = prev })
	return fake
}
