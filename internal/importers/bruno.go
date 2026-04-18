package importers

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sreeram/gurl/pkg/types"
)

// BrunoImporter handles Bruno .bru files
type BrunoImporter struct{}

// Name returns the importer name
func (b *BrunoImporter) Name() string {
	return "bruno"
}

// Extensions returns supported file extensions
func (b *BrunoImporter) Extensions() []string {
	return []string{".bru"}
}

// BrunoRequest represents the parsed structure of a .bru file
type BrunoRequest struct {
	Name        string
	Method      string
	URL         string
	Headers     []types.Header
	Body        string
	Auth        *BrunoAuth
	Script      string
	Vars        []BrunoVar
}

// BrunoAuth represents Bruno authentication
type BrunoAuth struct {
	Type        string
	Bearer      string
	Basic       *BrunoBasicAuth
	Digest      *BrunoDigestAuth
}

// BrunoBasicAuth represents basic auth in Bruno
type BrunoBasicAuth struct {
	Username string
	Password string
}

// BrunoDigestAuth represents digest auth in Bruno
type BrunoDigestAuth struct {
	Username string
	Password string
	Realm    string
	Nonce    string
	Qop      string
	Opaque   string
}

// BrunoVar represents a Bruno variable
type BrunoVar struct {
	Name  string
	Type  string // script, env, file
	Value string
}

// Parse reads and parses Bruno .bru files from a directory
func (b *BrunoImporter) Parse(path string) ([]*types.SavedRequest, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat path: %w", err)
	}

	var bruFiles []string

	if info.IsDir() {
		// Scan directory for .bru files
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil, fmt.Errorf("read directory: %w", err)
		}

		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".bru") {
				bruFiles = append(bruFiles, filepath.Join(path, entry.Name()))
			}
		}
	} else if strings.HasSuffix(path, ".bru") {
		bruFiles = append(bruFiles, path)
	} else {
		return nil, fmt.Errorf("not a valid .bru file or directory")
	}

	var requests []*types.SavedRequest

	for _, bruFile := range bruFiles {
		req, err := b.parseFile(bruFile)
		if err != nil {
			continue // Skip invalid files
		}
		requests = append(requests, req)
	}

	return requests, nil
}

// parseFile parses a single .bru file
func (b *BrunoImporter) parseFile(path string) (*types.SavedRequest, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	req := &BrunoRequest{
		Method: "GET",
	}

	var currentSection string
	var bodyLines []string

	for scanner.Scan() {
		line := scanner.Text()

		// Check for section headers
		switch {
		case strings.HasPrefix(line, "meta {"):
			currentSection = "meta"
		case strings.HasPrefix(line, "headers {"):
			currentSection = "headers"
		case strings.HasPrefix(line, "body {"):
			currentSection = "body"
		case strings.HasPrefix(line, "auth {"):
			currentSection = "auth"
		case strings.HasPrefix(line, "script {"):
			currentSection = "script"
		case strings.HasPrefix(line, "vars {"):
			currentSection = "vars"
		case strings.HasPrefix(line, "}") && currentSection != "":
			currentSection = ""
		case line == "" || strings.TrimSpace(line) == "":
			// Skip empty lines
		default:
			// Parse content based on current section
			switch currentSection {
			case "meta":
				b.parseMetaLine(req, line)
			case "headers":
				b.parseHeaderLine(req, line)
			case "body":
				bodyLines = append(bodyLines, line)
			case "auth":
				b.parseAuthLine(req, line)
			case "vars":
				b.parseVarLine(req, line)
			}
		}
	}

	// Build body from body lines
	if len(bodyLines) > 0 {
		req.Body = strings.Join(bodyLines, "\n")
	}

	return b.toSavedRequest(req, path), nil
}

// parseMetaLine parses a meta property line
func (b *BrunoImporter) parseMetaLine(req *BrunoRequest, line string) {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	switch key {
	case "name":
		req.Name = value
	case "method":
		req.Method = strings.ToUpper(value)
	case "url":
		req.URL = value
	}
}

// parseHeaderLine parses a header line
func (b *BrunoImporter) parseHeaderLine(req *BrunoRequest, line string) {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	req.Headers = append(req.Headers, types.Header{
		Key:   key,
		Value: value,
	})
}

// parseAuthLine parses an auth property line
func (b *BrunoImporter) parseAuthLine(req *BrunoRequest, line string) {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	switch key {
	case "type":
		if req.Auth == nil {
			req.Auth = &BrunoAuth{}
		}
		req.Auth.Type = value
	case "token", "bearer":
		if req.Auth == nil {
			req.Auth = &BrunoAuth{}
		}
		req.Auth.Bearer = value
	case "username":
		if req.Auth == nil {
			req.Auth = &BrunoAuth{}
		}
		if req.Auth.Type == "digest" {
			if req.Auth.Digest == nil {
				req.Auth.Digest = &BrunoDigestAuth{}
			}
			req.Auth.Digest.Username = value
		} else {
			if req.Auth.Basic == nil {
				req.Auth.Basic = &BrunoBasicAuth{}
			}
			req.Auth.Basic.Username = value
		}
	case "password":
		if req.Auth == nil {
			req.Auth = &BrunoAuth{}
		}
		if req.Auth.Type == "digest" {
			if req.Auth.Digest == nil {
				req.Auth.Digest = &BrunoDigestAuth{}
			}
			req.Auth.Digest.Password = value
		} else {
			if req.Auth.Basic == nil {
				req.Auth.Basic = &BrunoBasicAuth{}
			}
			req.Auth.Basic.Password = value
		}
	case "realm":
		if req.Auth == nil {
			req.Auth = &BrunoAuth{}
		}
		if req.Auth.Digest == nil {
			req.Auth.Digest = &BrunoDigestAuth{}
		}
		req.Auth.Digest.Realm = value
	case "nonce":
		if req.Auth == nil {
			req.Auth = &BrunoAuth{}
		}
		if req.Auth.Digest == nil {
			req.Auth.Digest = &BrunoDigestAuth{}
		}
		req.Auth.Digest.Nonce = value
	case "qop":
		if req.Auth == nil {
			req.Auth = &BrunoAuth{}
		}
		if req.Auth.Digest == nil {
			req.Auth.Digest = &BrunoDigestAuth{}
		}
		req.Auth.Digest.Qop = value
	case "opaque":
		if req.Auth == nil {
			req.Auth = &BrunoAuth{}
		}
		if req.Auth.Digest == nil {
			req.Auth.Digest = &BrunoDigestAuth{}
		}
		req.Auth.Digest.Opaque = value
	}
}

// parseVarLine parses a vars property line
func (b *BrunoImporter) parseVarLine(req *BrunoRequest, line string) {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	req.Vars = append(req.Vars, BrunoVar{
		Name:  key,
		Type:  "env",
		Value: value,
	})
}

// toSavedRequest converts a BrunoRequest to a SavedRequest
func (b *BrunoImporter) toSavedRequest(req *BrunoRequest, path string) *types.SavedRequest {
	saved := &types.SavedRequest{
		Name:    req.Name,
		URL:     req.URL,
		Method:  req.Method,
		Headers: req.Headers,
		Body:    req.Body,
	}

	// Set name from filename if not set in meta
	if saved.Name == "" {
		filename := filepath.Base(path)
		saved.Name = strings.TrimSuffix(filename, ".bru")
	}

	// Apply vars to URL, headers, and body
	if len(req.Vars) > 0 {
		saved.URL = b.applyVars(saved.URL, req.Vars)
		saved.Body = b.applyVars(saved.Body, req.Vars)
		for i := range saved.Headers {
			saved.Headers[i].Value = b.applyVars(saved.Headers[i].Value, req.Vars)
		}
	}

	// Add auth as headers
	if req.Auth != nil {
		switch req.Auth.Type {
		case "bearer":
			saved.Headers = append(saved.Headers, types.Header{
				Key:   "Authorization",
				Value: "Bearer " + req.Auth.Bearer,
			})
		case "basic":
			if req.Auth.Basic != nil {
				saved.Headers = append(saved.Headers, types.Header{
					Key:   "Authorization",
					Value: "Basic " + basicAuth(req.Auth.Basic.Username, req.Auth.Basic.Password),
				})
			}
		}
	}

	// Extract collection from parent directory
	dir := filepath.Base(filepath.Dir(path))
	if dir != "." && dir != "/" {
		saved.Collection = dir
	}

	return saved
}

// applyVars replaces {{var}} patterns with variable values
func (b *BrunoImporter) applyVars(s string, vars []BrunoVar) string {
	for _, v := range vars {
		s = strings.ReplaceAll(s, "{{"+v.Name+"}}", v.Value)
	}
	return s
}

// basicAuth creates a basic auth header value
func basicAuth(username, password string) string {
	return username + ":" + password
}
