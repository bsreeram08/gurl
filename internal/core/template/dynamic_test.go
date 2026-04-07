package template

import (
	"regexp"
	"testing"
	"time"

	"github.com/google/uuid"
)

// UUID v4 format: 8-4-4-4-12 hex digits
var uuidV4Regex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func TestDynamic_UUID(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "uuid generates valid v4 format",
			input: "{{$uuid}}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ResolveDynamic(tt.input)
			if err != nil {
				t.Errorf("ResolveDynamic() error = %v, want nil", err)
				return
			}
			if !uuidV4Regex.MatchString(result) {
				t.Errorf("ResolveDynamic() = %v, want UUID v4 format", result)
			}
			// Verify it's actually a valid UUID
			if _, err := uuid.Parse(result); err != nil {
				t.Errorf("ResolveDynamic() produced invalid UUID: %v", err)
			}
		})
	}
}

func TestDynamic_Timestamp(t *testing.T) {
	before := time.Now().Unix()
	result, err := ResolveDynamic("{{$timestamp}}")
	after := time.Now().Unix()
	if err != nil {
		t.Errorf("ResolveDynamic() error = %v, want nil", err)
		return
	}
	// Parse the result
	var ts int64
	for _, c := range result {
		if c < '0' || c > '9' {
			t.Errorf("ResolveDynamic() = %v, want integer timestamp", result)
			return
		}
		ts = ts*10 + int64(c-'0')
	}
	if ts < before || ts > after {
		t.Errorf("ResolveDynamic() timestamp = %v, want between %v and %v", ts, before, after)
	}
}

func TestDynamic_ISOTimestamp(t *testing.T) {
	result, err := ResolveDynamic("{{$isoTimestamp}}")
	if err != nil {
		t.Errorf("ResolveDynamic() error = %v, want nil", err)
		return
	}
	// ISO 8601 / RFC3339 format: 2006-01-02T15:04:05Z07:00
	_, err = time.Parse(time.RFC3339, result)
	if err != nil {
		t.Errorf("ResolveDynamic() = %v, not valid RFC3339 format: %v", result, err)
	}
}

func TestDynamic_RandomInt(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantMin int64
		wantMax int64
	}{
		{
			name:    "randomInt in range 1-100",
			input:   "{{$randomInt(1, 100)}}",
			wantMin: 1,
			wantMax: 100,
		},
		{
			name:    "randomInt in range 0-10",
			input:   "{{$randomInt(0, 10)}}",
			wantMin: 0,
			wantMax: 10,
		},
		{
			name:    "randomInt same min max",
			input:   "{{$randomInt(5, 5)}}",
			wantMin: 5,
			wantMax: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ResolveDynamic(tt.input)
			if err != nil {
				t.Errorf("ResolveDynamic() error = %v, want nil", err)
				return
			}
			var val int64
			for _, c := range result {
				if c < '0' || c > '9' {
					t.Errorf("ResolveDynamic() = %v, want integer", result)
					return
				}
				val = val*10 + int64(c-'0')
			}
			if val < tt.wantMin || val > tt.wantMax {
				t.Errorf("ResolveDynamic() = %v, want between %v and %v", val, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestDynamic_RandomString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantLen int
	}{
		{
			name:    "randomString length 16",
			input:   "{{$randomString(16)}}",
			wantLen: 16,
		},
		{
			name:    "randomString length 8",
			input:   "{{$randomString(8)}}",
			wantLen: 8,
		},
		{
			name:    "randomString length 32",
			input:   "{{$randomString(32)}}",
			wantLen: 32,
		},
	}

	// Alphanumeric regex
	alphanumeric := regexp.MustCompile(`^[a-zA-Z0-9]+$`)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ResolveDynamic(tt.input)
			if err != nil {
				t.Errorf("ResolveDynamic() error = %v, want nil", err)
				return
			}
			if len(result) != tt.wantLen {
				t.Errorf("ResolveDynamic() len = %v, want %v", len(result), tt.wantLen)
			}
			if !alphanumeric.MatchString(result) {
				t.Errorf("ResolveDynamic() = %v, want alphanumeric only", result)
			}
		})
	}
}

func TestDynamic_RandomEmail(t *testing.T) {
	result, err := ResolveDynamic("{{$randomEmail}}")
	if err != nil {
		t.Errorf("ResolveDynamic() error = %v, want nil", err)
		return
	}
	// Format: {randomString(8)}@example.com
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9]+@example\.com$`)
	if !emailRegex.MatchString(result) {
		t.Errorf("ResolveDynamic() = %v, want format user@example.com", result)
	}
}

func TestDynamic_RandomUUID_Uniqueness(t *testing.T) {
	// Run 100 iterations and verify all are unique
	results := make(map[string]bool)
	for i := 0; i < 100; i++ {
		result, err := ResolveDynamic("{{$uuid}}")
		if err != nil {
			t.Errorf("ResolveDynamic() iteration %d error = %v, want nil", i, err)
			return
		}
		if results[result] {
			t.Errorf("ResolveDynamic() produced duplicate UUID: %v", result)
			return
		}
		results[result] = true
	}
}

func TestDynamic_UnknownFunction(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "unknown function returns error",
			input:   "{{$unknown}}",
			wantErr: true,
		},
		{
			name:    "unknownWithArgs returns error",
			input:   "{{$unknownFunc(1, 2)}}",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ResolveDynamic(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveDynamic() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && err.Error() == "" {
				t.Errorf("ResolveDynamic() returned empty error message, want descriptive error")
			}
		})
	}
}
