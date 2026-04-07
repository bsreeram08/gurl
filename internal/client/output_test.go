package client

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mockResponse creates a Response with given body and headers
func mockResponse(body []byte, headers map[string][]string) Response {
	return Response{
		StatusCode: 200,
		Headers:    headers,
		Body:       body,
	}
}

// TestSaveResponse_JSON saves JSON body to file, file content matches
func TestSaveResponse_JSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "response.json")

	body := []byte(`{"key": "value", "number": 42}`)
	resp := mockResponse(body, map[string][]string{
		"Content-Type": {"application/json"},
	})

	err := SaveToFile(&resp, path, false)
	if err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if string(got) != string(body) {
		t.Errorf("file content mismatch:\ngot:  %s\nwant: %s", got, body)
	}
}

// TestSaveResponse_Binary saves binary (image/pdf) without corruption
func TestSaveResponse_Binary(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "image.png")

	// PNG header bytes (binary data with null bytes)
	body := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52}
	resp := mockResponse(body, map[string][]string{
		"Content-Type": {"image/png"},
	})

	err := SaveToFile(&resp, path, false)
	if err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if len(got) != len(body) {
		t.Errorf("binary length mismatch: got %d, want %d", len(got), len(body))
	}
	if string(got) != string(body) {
		t.Errorf("binary content mismatch")
	}
}

// TestSaveResponse_AutoFilename derives filename from Content-Disposition header
func TestSaveResponse_AutoFilename(t *testing.T) {
	resp := mockResponse([]byte(`test content`), map[string][]string{
		"Content-Disposition": {`attachment; filename="report.pdf"`},
	})

	filename := DeriveFilename(&resp, "")
	if filename != "report.pdf" {
		t.Errorf("DeriveFilename = %q, want %q", filename, "report.pdf")
	}
}

// TestSaveResponse_AutoFilenameFromURL derives filename from URL when no Content-Disposition
func TestSaveResponse_AutoFilenameFromURL(t *testing.T) {
	resp := mockResponse([]byte(`test content`), map[string][]string{
		"Content-Type": {"text/html"},
	})

	filename := DeriveFilename(&resp, "https://example.com/api/data/download")
	if filename != "download" {
		t.Errorf("DeriveFilename = %q, want %q", filename, "download")
	}
}

// TestSaveResponse_CustomPath saves to user-specified path
func TestSaveResponse_CustomPath(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "custom", "subdir", "output.txt")

	body := []byte("custom path content")
	resp := mockResponse(body, nil)

	err := SaveToFile(&resp, path, false)
	if err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if string(got) != string(body) {
		t.Errorf("file content mismatch: got %q, want %q", string(got), string(body))
	}
}

// TestSaveResponse_CreateDirs creates parent directories if missing
func TestSaveResponse_CreateDirs(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "a", "b", "c", "d", "file.txt")

	body := []byte("nested content")
	resp := mockResponse(body, nil)

	err := SaveToFile(&resp, path, false)
	if err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if string(got) != string(body) {
		t.Errorf("file content mismatch")
	}
}

// TestSaveResponse_ExistingFile returns error (no silent overwrite) unless --force
func TestSaveResponse_ExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "existing.txt")

	// Create existing file
	if err := os.WriteFile(path, []byte("original"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	body := []byte("new content")
	resp := mockResponse(body, nil)

	// Without force, should error
	err := SaveToFile(&resp, path, false)
	if err == nil {
		t.Error("SaveToFile expected error on existing file without force, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") && !strings.Contains(err.Error(), "--force") {
		t.Errorf("error message should mention 'already exists' and '--force', got: %v", err)
	}

	// With force, should succeed
	err = SaveToFile(&resp, path, true)
	if err != nil {
		t.Errorf("SaveToFile with force failed: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(got) != string(body) {
		t.Errorf("file content after force overwrite mismatch")
	}
}

// TestSaveResponse_Stdout writes to stdout when path is "-"
func TestSaveResponse_Stdout(t *testing.T) {
	body := []byte("stdout content")
	resp := mockResponse(body, nil)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := SaveToFile(&resp, "-", false)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("SaveToFile to stdout failed: %v", err)
	}

	// Read captured output
	out, _ := io.ReadAll(r)
	if string(out) != string(body) {
		t.Errorf("stdout content = %q, want %q", string(out), string(body))
	}
}

// TestDeriveFilename_FilenameWithExtension extracts filename with extension from Content-Disposition
func TestDeriveFilename_FilenameWithExtension(t *testing.T) {
	resp := mockResponse([]byte(`test`), map[string][]string{
		"Content-Disposition": {`attachment; filename="data.tar.gz"`},
	})

	filename := DeriveFilename(&resp, "")
	if filename != "data.tar.gz" {
		t.Errorf("DeriveFilename = %q, want %q", filename, "data.tar.gz")
	}
}

// TestDeriveFilename_QuotedFilename extracts quoted filename
func TestDeriveFilename_QuotedFilename(t *testing.T) {
	resp := mockResponse([]byte(`test`), map[string][]string{
		"Content-Disposition": {`attachment; filename="my document.pdf"`},
	})

	filename := DeriveFilename(&resp, "")
	if filename != "my document.pdf" {
		t.Errorf("DeriveFilename = %q, want %q", filename, "my document.pdf")
	}
}

// TestDeriveFilename_NoHeaders falls back to URL last segment
func TestDeriveFilename_NoHeaders(t *testing.T) {
	resp := mockResponse([]byte(`test`), nil)

	filename := DeriveFilename(&resp, "https://api.example.com/v1/users/avatar.jpg?size=large")
	if filename != "avatar.jpg" {
		t.Errorf("DeriveFilename = %q, want %q", filename, "avatar.jpg")
	}
}

// TestDeriveFilename_EmptyFallback returns "response" when nothing available
func TestDeriveFilename_EmptyFallback(t *testing.T) {
	resp := mockResponse([]byte(`test`), nil)

	filename := DeriveFilename(&resp, "")
	if filename != "response" {
		t.Errorf("DeriveFilename = %q, want %q", filename, "response")
	}
}

// TestSaveToFile_EmptyBody saves empty body correctly
func TestSaveToFile_EmptyBody(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "empty.txt")

	resp := mockResponse([]byte{}, nil)

	err := SaveToFile(&resp, path, false)
	if err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("empty file should be 0 bytes, got %d", len(got))
	}
}

// TestSaveToFile_PermissionError tests handling of permission errors
func TestSaveToFile_PermissionError(t *testing.T) {
	// This test would require root or specific setup, so we just verify
	// the error is returned properly if the file is not writable
	// (Not easy to test in normal circumstances)
	t.Skip("Permission error testing requires specific environment setup")
}

// TestSaveToFile_NonExistentParentWithFileAsLastComponent
func TestSaveToFile_NonExistentParentWithFileAsLastComponent(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "nonexistent", "file.txt")

	resp := mockResponse([]byte("content"), nil)

	err := SaveToFile(&resp, path, false)
	if err != nil {
		t.Errorf("SaveToFile should create parent dirs, got error: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(got) != "content" {
		t.Errorf("content mismatch")
	}
}
