package client

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	ErrFileExists = errors.New("file already exists, use --force to overwrite")
)

// SaveToFile saves response body to a file.
// If path is "-", writes to stdout.
// If force is false and file exists, returns ErrFileExists.
func SaveToFile(resp *Response, path string, force bool) error {
	if path == "-" {
		_, err := os.Stdout.Write(resp.Body)
		return err
	}

	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%w (use --force to overwrite)", ErrFileExists)
		}
	}

	return os.WriteFile(path, resp.Body, 0644)
}

// DeriveFilename extracts filename from Content-Disposition header or URL.
// Falls back to "response" if neither is available.
func DeriveFilename(resp *Response, fallbackURL string) string {
	// Try Content-Disposition header first
	if cd := resp.Headers.Get("Content-Disposition"); cd != "" {
		if filename := extractFilename(cd); filename != "" {
			return filename
		}
	}

	// Try URL if provided via Response.URL
	if resp.URL != "" {
		if filename := filenameFromURL(resp.URL); filename != "" {
			return filename
		}
	}

	// Try fallback URL
	if fallbackURL != "" {
		if filename := filenameFromURL(fallbackURL); filename != "" {
			return filename
		}
	}

	return "response"
}

var (
	quotedFilenameRegex = regexp.MustCompile(`filename\*?=['"]?(?:UTF-8'')?([^"'\r\n]+)['"]?`)
	filenameRegex       = regexp.MustCompile(`filename=['"]?([^"'\s;]+)['"]?`)
)

func extractFilename(contentDisposition string) string {
	// Try RFC 5987 filename* (UTF-8'' encoded)
	matches := quotedFilenameRegex.FindStringSubmatch(contentDisposition)
	if len(matches) >= 2 {
		filename := matches[1]
		// Decode RFC 5987 encoding (UTF-8''...)
		if idx := strings.Index(filename, "''"); idx != -1 {
			// Assume UTF-8, just take after ''
			filename = filename[idx+2:]
		}
		// Unescape percent-encoded
		if decoded, err := url.PathUnescape(filename); err == nil {
			filename = decoded
		}
		return filepath.Base(filename)
	}

	// Try simple filename=
	matches = filenameRegex.FindStringSubmatch(contentDisposition)
	if len(matches) >= 2 {
		return filepath.Base(matches[1])
	}

	return ""
}

func filenameFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	// Get last path segment
	path := strings.TrimSuffix(u.Path, "/")
	if path == "" {
		return ""
	}
	filename := filepath.Base(path)
	// Remove query params if they sneak in
	if qIdx := strings.Index(filename, "?"); qIdx != -1 {
		filename = filename[:qIdx]
	}
	return filename
}

// SaveToFileWithAutoName saves response body to a file with auto-derived filename.
// Convenience wrapper that derives filename if path is empty.
func SaveToFileWithAutoName(resp *Response, path string, force bool) (string, error) {
	if path == "" {
		path = DeriveFilename(resp, "")
	}
	if path == "-" {
		return "", SaveToFile(resp, "-", force)
	}
	if filepath.Dir(path) == "" {
		// Just filename, use current dir
	}
	err := SaveToFile(resp, path, force)
	return path, err
}

// CopyBodyToWriter writes response body to an io.Writer.
// Returns bytes written and any error.
func CopyBodyToWriter(resp *Response, w io.Writer) (int64, error) {
	n, err := w.Write(resp.Body)
	if err != nil {
		return int64(n), err
	}
	return int64(n), nil
}
