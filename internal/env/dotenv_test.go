package env

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseDotenv(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected map[string]string
		wantErr  bool
	}{
		{
			name:    "simple KEY=value",
			content: "FOO=bar\nBAZ=qux",
			expected: map[string]string{
				"FOO": "bar",
				"BAZ": "qux",
			},
		},
		{
			name:    "double quoted values",
			content: `FOO="hello world"`,
			expected: map[string]string{
				"FOO": "hello world",
			},
		},
		{
			name:    "single quoted values",
			content: "FOO='hello world'",
			expected: map[string]string{
				"FOO": "hello world",
			},
		},
		{
			name:    "comments ignored",
			content: "# This is a comment\nFOO=bar\n# Another comment",
			expected: map[string]string{
				"FOO": "bar",
			},
		},
		{
			name:    "empty lines ignored",
			content: "FOO=bar\n\nBAZ=qux\n",
			expected: map[string]string{
				"FOO": "bar",
				"BAZ": "qux",
			},
		},
		{
			name:    "export prefix ignored",
			content: "export FOO=bar\nexport BAZ=qux",
			expected: map[string]string{
				"FOO": "bar",
				"BAZ": "qux",
			},
		},
		{
			name:    "empty value",
			content: "FOO=\nBAR=baz",
			expected: map[string]string{
				"FOO": "",
				"BAR": "baz",
			},
		},
		{
			name:    "value with equals sign",
			content: "FOO=bar=baz",
			expected: map[string]string{
				"FOO": "bar=baz",
			},
		},
		{
			name:    "double quoted with equals sign",
			content: `FOO="bar=baz"`,
			expected: map[string]string{
				"FOO": "bar=baz",
			},
		},
		{
			name:    "trailing whitespace trimmed",
			content: "FOO=bar   \n",
			expected: map[string]string{
				"FOO": "bar",
			},
		},
		{
			name:    "mixed comments and empty lines",
			content: "# comment\n\nFOO=bar\n\n# another\nBAZ=qux\n",
			expected: map[string]string{
				"FOO": "bar",
				"BAZ": "qux",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDotenv(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDotenv() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				for key, val := range tt.expected {
					if result[key] != val {
						t.Errorf("ParseDotenv()[%s] = %q, want %q", key, result[key], val)
					}
				}
				for key, val := range result {
					if tt.expected[key] != val {
						t.Errorf("ParseDotenv()[%s] = %q, want %q", key, val, tt.expected[key])
					}
				}
			}
		})
	}
}

func TestParseDotenvFile(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected map[string]string
	}{
		{
			name: "standard env file",
			content: `# Database configuration
DB_HOST=localhost
DB_PORT=5432
DB_NAME="myapp"

# API keys
API_KEY='secret-key'
DEBUG=true
`,
			expected: map[string]string{
				"DB_HOST": "localhost",
				"DB_PORT": "5432",
				"DB_NAME": "myapp",
				"API_KEY": "secret-key",
				"DEBUG":   "true",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			envFile := filepath.Join(tmpDir, ".env")
			if err := os.WriteFile(envFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write temp env file: %v", err)
			}

			result, err := ParseDotenvFile(envFile)
			if err != nil {
				t.Errorf("ParseDotenvFile() error = %v", err)
				return
			}

			for key, val := range tt.expected {
				if result[key] != val {
					t.Errorf("ParseDotenvFile()[%s] = %q, want %q", key, result[key], val)
				}
			}
		})
	}
}

func TestParseDotenvFileNotFound(t *testing.T) {
	_, err := ParseDotenvFile("/nonexistent/.env")
	if err == nil {
		t.Error("ParseDotenvFile() expected error for nonexistent file")
	}
}
