package runner

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sreeram/gurl/pkg/types"
)

func TestParseRunIf_SupportedExpressions(t *testing.T) {
	tests := []struct {
		name string
		expr string
		want runIfCondition
	}{
		{name: "equals with bare value", expr: "env == beta", want: runIfCondition{Variable: "env", Operator: "==", Value: "beta"}},
		{name: "not equals with single quoted value", expr: "env != 'prod'", want: runIfCondition{Variable: "env", Operator: "!=", Value: "prod"}},
		{name: "equals empty string", expr: "terminalId == ''", want: runIfCondition{Variable: "terminalId", Operator: "==", Value: ""}},
		{name: "not equals double quoted empty string", expr: "terminalId != \"\"", want: runIfCondition{Variable: "terminalId", Operator: "!=", Value: ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRunIf(tt.expr)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %+v, got %+v", tt.want, got)
			}
		})
	}
}

func TestEvaluateRunIf_AbsentVariablesAreEmpty(t *testing.T) {
	tests := []struct {
		name string
		expr string
		vars map[string]string
		want bool
	}{
		{name: "equals empty when absent", expr: "terminalId == ''", vars: map[string]string{}, want: true},
		{name: "not equals empty when absent", expr: "terminalId != ''", vars: map[string]string{}, want: false},
		{name: "equals non-empty matches", expr: "env == beta", vars: map[string]string{"env": "beta"}, want: true},
		{name: "not equals non-empty matches", expr: "env != prod", vars: map[string]string{"env": "beta"}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := evaluateRunIf(tt.expr, tt.vars)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestParseRunIf_UnsupportedExpressions(t *testing.T) {
	tests := []string{
		"terminalId contains term",
		"env == beta && region == us",
		"env in (beta,prod)",
		"env > beta",
	}

	for _, expr := range tests {
		t.Run(expr, func(t *testing.T) {
			if _, err := parseRunIf(expr); err == nil {
				t.Fatalf("expected error for %q", expr)
			}
		})
	}
}

func TestRunner_RunIfFalseSkipsBeforeHTTP(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	requestCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	db.SaveRequest(&types.SavedRequest{
		ID:         "runif-skip",
		Name:       "conditional-step",
		URL:        ts.URL,
		Method:     http.MethodGet,
		Collection: "runif-flow",
		RunIf:      "terminalId == ''",
	})

	runner := NewRunner(db, envStorage)
	results, err := runner.Run(context.Background(), RunConfig{
		CollectionName: "runif-flow",
		Vars:           map[string]string{"terminalId": "term_123"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if requestCount != 0 {
		t.Fatalf("expected no HTTP requests, got %d", requestCount)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 run result, got %d", len(results))
	}
	result := results[0]
	if result.Skipped != 1 || result.Failed != 0 {
		t.Fatalf("expected skipped=1 failed=0, got skipped=%d failed=%d", result.Skipped, result.Failed)
	}
	if len(result.RequestResults) != 1 {
		t.Fatalf("expected 1 request result, got %d", len(result.RequestResults))
	}
	requestResult := result.RequestResults[0]
	if !requestResult.Skipped {
		t.Fatalf("expected request to be skipped, got %+v", requestResult)
	}
	if requestResult.SkipReason != SkipReasonRunIf {
		t.Fatalf("expected skip reason %q, got %q", SkipReasonRunIf, requestResult.SkipReason)
	}
	if requestResult.FailurePhase != "" {
		t.Fatalf("expected no failure phase, got %q", requestResult.FailurePhase)
	}
}

func TestRunner_InvalidRunIfFailsBeforeHTTP(t *testing.T) {
	db := newMockDB()
	envStorage := newMockEnvStorage()

	requestCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	db.SaveRequest(&types.SavedRequest{
		ID:         "runif-invalid",
		Name:       "conditional-step",
		URL:        ts.URL,
		Method:     http.MethodGet,
		Collection: "runif-flow",
		RunIf:      "terminalId contains term",
	})

	runner := NewRunner(db, envStorage)
	results, err := runner.Run(context.Background(), RunConfig{CollectionName: "runif-flow"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if requestCount != 0 {
		t.Fatalf("expected no HTTP requests, got %d", requestCount)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 run result, got %d", len(results))
	}
	requestResult := results[0].RequestResults[0]
	if requestResult.Skipped {
		t.Fatalf("expected request to fail, got skipped result: %+v", requestResult)
	}
	if requestResult.FailurePhase != FailurePhaseRunIf {
		t.Fatalf("expected failure phase %q, got %q", FailurePhaseRunIf, requestResult.FailurePhase)
	}
	if requestResult.Error == "" {
		t.Fatalf("expected an error message for invalid run_if")
	}
	if got := requestResult.Error; got == "" || !containsRunIfSupportText(got) {
		t.Fatalf("expected supported syntax message, got %q", got)
	}
}

func containsRunIfSupportText(s string) bool {
	return strings.Contains(s, "VAR == VALUE") && strings.Contains(s, "VAR != VALUE")
}
