package reporters

import (
	"encoding/json"
	"encoding/xml"
	"os"
	"strings"
	"testing"
	"time"
)

func createTestResults() []RunResult {
	return []RunResult{
		{
			CollectionName: "test-collection",
			Total:          3,
			Passed:         2,
			Failed:         1,
			Skipped:        0,
			Duration:       150 * time.Millisecond,
			Iteration:      1,
			RequestResults: []*RequestResult{
				{
					RequestName: "get-users",
					StatusCode:  200,
					Passed:      true,
					Skipped:     false,
					Error:       "",
					Duration:    50 * time.Millisecond,
					AssertionResults: []AssertionResult{
						{
							Field:   "status",
							Op:      "=",
							Value:   "200",
							Passed:  true,
							Message: "PASS: status = 200 (actual: 200)",
						},
					},
				},
				{
					RequestName: "create-user",
					StatusCode:  201,
					Passed:      true,
					Skipped:     false,
					Error:       "",
					Duration:    60 * time.Millisecond,
					AssertionResults: []AssertionResult{
						{
							Field:   "status",
							Op:      "=",
							Value:   "201",
							Passed:  true,
							Message: "PASS: status = 201 (actual: 201)",
						},
					},
				},
				{
					RequestName: "delete-user",
					StatusCode:  404,
					Passed:      false,
					Skipped:     false,
					Error:       "",
					Duration:    40 * time.Millisecond,
					AssertionResults: []AssertionResult{
						{
							Field:   "status",
							Op:      "=",
							Value:   "204",
							Passed:  false,
							Message: "FAIL: status = 204 (actual: 404)",
						},
					},
				},
			},
		},
	}
}

func TestReporter_JUnit(t *testing.T) {
	reporter := NewJUnitXMLReporter()
	if reporter.Name() != "junit" {
		t.Errorf("expected name 'junit', got '%s'", reporter.Name())
	}

	results := createTestResults()
	content, err := reporter.Report(results)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var suites junitTestSuites
	if err := xml.Unmarshal(content, &suites); err != nil {
		t.Fatalf("failed to parse XML: %v", err)
	}

	if len(suites.Suites) != 1 {
		t.Errorf("expected 1 test suite, got %d", len(suites.Suites))
	}

	suite := suites.Suites[0]
	if suite.Name != "test-collection" {
		t.Errorf("expected suite name 'test-collection', got '%s'", suite.Name)
	}
	if suite.Tests != 3 {
		t.Errorf("expected 3 tests, got %d", suite.Tests)
	}
	if suite.Failures != 1 {
		t.Errorf("expected 1 failure, got %d", suite.Failures)
	}
	if len(suite.TestCases) != 3 {
		t.Errorf("expected 3 test cases, got %d", len(suite.TestCases))
	}

	for _, tc := range suite.TestCases {
		if tc.ClassName != "test-collection" {
			t.Errorf("expected classname 'test-collection', got '%s'", tc.ClassName)
		}
	}
}

func TestReporter_JSON(t *testing.T) {
	reporter := NewJSONReporter()
	if reporter.Name() != "json" {
		t.Errorf("expected name 'json', got '%s'", reporter.Name())
	}

	results := createTestResults()
	content, err := reporter.Report(results)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var report JSONReport
	if err := json.Unmarshal(content, &report); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if report.TotalRuns != 1 {
		t.Errorf("expected 1 total run, got %d", report.TotalRuns)
	}
	if report.TotalTests != 3 {
		t.Errorf("expected 3 total tests, got %d", report.TotalTests)
	}
	if report.Passed != 2 {
		t.Errorf("expected 2 passed, got %d", report.Passed)
	}
	if report.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", report.Failed)
	}
	if len(report.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(report.Results))
	}
}

func TestReporter_HTML(t *testing.T) {
	reporter := NewHTMLReporter()
	if reporter.Name() != "html" {
		t.Errorf("expected name 'html', got '%s'", reporter.Name())
	}

	results := createTestResults()
	content, err := reporter.Report(results)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := string(content)

	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("HTML output missing DOCTYPE")
	}
	if !strings.Contains(html, "<html") {
		t.Error("HTML output missing html tag")
	}
	if !strings.Contains(html, "<style>") {
		t.Error("HTML output missing style tag")
	}
	if !strings.Contains(html, "background: #f5f5f5") {
		t.Error("HTML output missing embedded CSS")
	}
	if !strings.Contains(html, "Test Report") {
		t.Error("HTML output missing title")
	}
	if strings.Contains(html, "href=") || strings.Contains(html, "src=") {
		t.Error("HTML should not contain external resource links")
	}
}

func TestReporter_JUnit_FailedTest(t *testing.T) {
	reporter := NewJUnitXMLReporter()

	results := []RunResult{
		{
			CollectionName: "fail-test",
			Total:          1,
			Passed:         0,
			Failed:         1,
			Skipped:        0,
			Duration:       100 * time.Millisecond,
			Iteration:      1,
			RequestResults: []*RequestResult{
				{
					RequestName: "failing-request",
					StatusCode:  500,
					Passed:      false,
					Skipped:     false,
					Error:       "",
					Duration:    100 * time.Millisecond,
				},
			},
		},
	}

	content, err := reporter.Report(results)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var suites junitTestSuites
	if err := xml.Unmarshal(content, &suites); err != nil {
		t.Fatalf("failed to parse XML: %v", err)
	}

	suite := suites.Suites[0]
	if suite.Failures != 1 {
		t.Errorf("expected 1 failure, got %d", suite.Failures)
	}

	tc := suite.TestCases[0]
	if tc.Failure == nil {
		t.Error("expected failure element to be set")
	}
	if tc.Failure.Message != "Status code: 500" {
		t.Errorf("unexpected failure message: %s", tc.Failure.Message)
	}
}

func TestReporter_JUnit_Skipped(t *testing.T) {
	reporter := NewJUnitXMLReporter()

	results := []RunResult{
		{
			CollectionName: "skip-test",
			Total:          1,
			Passed:         0,
			Failed:         0,
			Skipped:        1,
			Duration:       50 * time.Millisecond,
			Iteration:      1,
			RequestResults: []*RequestResult{
				{
					RequestName: "skipped-request",
					StatusCode:  0,
					Passed:      false,
					Skipped:     true,
					Error:       "",
					Duration:    50 * time.Millisecond,
				},
			},
		},
	}

	content, err := reporter.Report(results)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var suites junitTestSuites
	if err := xml.Unmarshal(content, &suites); err != nil {
		t.Fatalf("failed to parse XML: %v", err)
	}

	suite := suites.Suites[0]
	if suite.Skipped != 1 {
		t.Errorf("expected 1 skipped, got %d", suite.Skipped)
	}

	tc := suite.TestCases[0]
	if tc.Skipped == nil {
		t.Error("expected skipped element to be set")
	}
}

func TestReporter_Console(t *testing.T) {
	reporter := NewConsoleReporter()
	if reporter.Name() != "console" {
		t.Errorf("expected name 'console', got '%s'", reporter.Name())
	}

	results := createTestResults()
	content, err := reporter.Report(results)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := string(content)

	if !strings.Contains(output, "TEST REPORT") {
		t.Error("console output missing header")
	}
	if !strings.Contains(output, "\033[") {
		t.Error("console output missing ANSI color codes")
	}
	if !strings.Contains(output, "✓ PASS") && !strings.Contains(output, "PASS") {
		t.Error("console output missing PASS indicator")
	}
	if !strings.Contains(output, "✗ FAIL") && !strings.Contains(output, "FAIL") {
		t.Error("console output missing FAIL indicator")
	}
}

func TestReporter_Multiple(t *testing.T) {
	junit := NewJUnitXMLReporter()
	json := NewJSONReporter()
	html := NewHTMLReporter()
	console := NewConsoleReporter()

	results := createTestResults()

	junitContent, err := junit.Report(results)
	if err != nil {
		t.Fatalf("junit report failed: %v", err)
	}
	if len(junitContent) == 0 {
		t.Error("junit report empty")
	}

	jsonContent, err := json.Report(results)
	if err != nil {
		t.Fatalf("json report failed: %v", err)
	}
	if len(jsonContent) == 0 {
		t.Error("json report empty")
	}

	htmlContent, err := html.Report(results)
	if err != nil {
		t.Fatalf("html report failed: %v", err)
	}
	if len(htmlContent) == 0 {
		t.Error("html report empty")
	}

	consoleContent, err := console.Report(results)
	if err != nil {
		t.Fatalf("console report failed: %v", err)
	}
	if len(consoleContent) == 0 {
		t.Error("console report empty")
	}

	tmpDir := t.TempDir()
	junitPath := tmpDir + "/junit.xml"
	jsonPath := tmpDir + "/report.json"
	htmlPath := tmpDir + "/report.html"
	consolePath := tmpDir + "/console.txt"

	if err := junit.WriteToFile(results, junitPath); err != nil {
		t.Fatalf("junit WriteToFile failed: %v", err)
	}
	if err := json.WriteToFile(results, jsonPath); err != nil {
		t.Fatalf("json WriteToFile failed: %v", err)
	}
	if err := html.WriteToFile(results, htmlPath); err != nil {
		t.Fatalf("html WriteToFile failed: %v", err)
	}
	if err := console.WriteToFile(results, consolePath); err != nil {
		t.Fatalf("console WriteToFile failed: %v", err)
	}

	for _, path := range []string{junitPath, jsonPath, htmlPath, consolePath} {
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("file %s does not exist: %v", path, err)
		}
		if info.Size() == 0 {
			t.Errorf("file %s is empty", path)
		}
	}
}
