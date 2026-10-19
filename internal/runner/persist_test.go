package runner

import (
	"strings"
	"testing"
)

func TestPersistRefusesPlaceholderValues(t *testing.T) {
	_, err := PersistDirtyVariables(nil, nil, "beta", "", map[string]string{"orderId": "{{API_KEY}}"}, nil)
	if err == nil || !strings.Contains(err.Error(), "orderId") {
		t.Fatalf("expected refusal, got %v", err)
	}
}
