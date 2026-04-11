package reporters

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"os"
	"strings"
	"time"

	"github.com/sreeram/gurl/internal/formatter"
)

type Reporter interface {
	Name() string
	Report(results []RunResult) ([]byte, error)
	WriteToFile(results []RunResult, path string) error
}

type RunResult struct {
	CollectionName string
	Total          int
	Passed         int
	Failed         int
	Skipped        int
	Duration       time.Duration
	RequestResults []*RequestResult
	Iteration      int
}

type RequestResult struct {
	RequestName      string
	StatusCode       int
	Passed           bool
	Skipped          bool
	Error            string
	Duration         time.Duration
	AssertionResults []AssertionResult
}

type AssertionResult struct {
	Field   string
	Op      string
	Value   string
	Passed  bool
	Message string
}

type JUnitXMLReporter struct{}

func NewJUnitXMLReporter() *JUnitXMLReporter {
	return &JUnitXMLReporter{}
}

func (r *JUnitXMLReporter) Name() string { return "junit" }

type junitTestSuites struct {
	XMLName xml.Name         `xml:"testsuites"`
	Suites  []junitTestSuite `xml:"testsuite"`
}

type junitTestSuite struct {
	XMLName   xml.Name        `xml:"testsuite"`
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	Skipped   int             `xml:"skipped,attr"`
	Errors    int             `xml:"errors,attr"`
	Time      string          `xml:"time,attr"`
	TestCases []junitTestCase `xml:"testcase"`
}

type junitTestCase struct {
	XMLName   xml.Name      `xml:"testcase"`
	Name      string        `xml:"name,attr"`
	ClassName string        `xml:"classname,attr"`
	Time      string        `xml:"time,attr"`
	Failure   *junitFailure `xml:"failure,omitempty"`
	Skipped   *junitSkipped `xml:"skipped,omitempty"`
	Error     *junitError   `xml:"error,omitempty"`
}

type junitFailure struct {
	XMLName xml.Name `xml:"failure"`
	Message string   `xml:"message,attr"`
	Type    string   `xml:"type,attr"`
	Content string   `xml:",chardata"`
}

type junitSkipped struct {
	XMLName xml.Name `xml:"skipped"`
}

type junitError struct {
	XMLName xml.Name `xml:"error"`
	Message string   `xml:"message,attr"`
	Type    string   `xml:"type,attr"`
	Content string   `xml:",chardata"`
}

func (r *JUnitXMLReporter) Report(results []RunResult) ([]byte, error) {
	suites := make([]junitTestSuite, 0, len(results))

	for _, result := range results {
		cases := make([]junitTestCase, 0, len(result.RequestResults))

		for _, reqResult := range result.RequestResults {
			var failure *junitFailure
			var skipped *junitSkipped
			var err *junitError

			if reqResult.Skipped {
				skipped = &junitSkipped{}
			} else if reqResult.Error != "" {
				err = &junitError{
					Message: reqResult.Error,
					Type:    "request_error",
					Content: reqResult.Error,
				}
			} else if !reqResult.Passed {
				failure = &junitFailure{
					Message: fmt.Sprintf("Status code: %d", reqResult.StatusCode),
					Type:    "assertion_failure",
					Content: fmt.Sprintf("Request '%s' failed with status code %d", reqResult.RequestName, reqResult.StatusCode),
				}
			}

			timeStr := fmt.Sprintf("%.3f", reqResult.Duration.Seconds())

			cases = append(cases, junitTestCase{
				Name:      reqResult.RequestName,
				ClassName: result.CollectionName,
				Time:      timeStr,
				Failure:   failure,
				Skipped:   skipped,
				Error:     err,
			})
		}

		timeStr := fmt.Sprintf("%.3f", result.Duration.Seconds())

		suites = append(suites, junitTestSuite{
			Name:      result.CollectionName,
			Tests:     result.Total,
			Failures:  result.Failed,
			Skipped:   result.Skipped,
			Errors:    0,
			Time:      timeStr,
			TestCases: cases,
		})
	}

	ts := junitTestSuites{Suites: suites}
	return xml.MarshalIndent(ts, "", "  ")
}

func (r *JUnitXMLReporter) WriteToFile(results []RunResult, path string) error {
	content, err := r.Report(results)
	if err != nil {
		return err
	}
	return os.WriteFile(path, content, 0644)
}

type JSONReporter struct{}

func NewJSONReporter() *JSONReporter {
	return &JSONReporter{}
}

func (r *JSONReporter) Name() string { return "json" }

type JSONReport struct {
	GeneratedAt string          `json:"generated_at"`
	TotalRuns   int             `json:"total_runs"`
	TotalTests  int             `json:"total_tests"`
	Passed      int             `json:"passed"`
	Failed      int             `json:"failed"`
	Skipped     int             `json:"skipped"`
	Duration    string          `json:"duration"`
	Results     []JSONRunResult `json:"results"`
}

type JSONRunResult struct {
	CollectionName string              `json:"collection_name"`
	Iteration      int                 `json:"iteration"`
	Total          int                 `json:"total"`
	Passed         int                 `json:"passed"`
	Failed         int                 `json:"failed"`
	Skipped        int                 `json:"skipped"`
	Duration       string              `json:"duration"`
	RequestResults []JSONRequestResult `json:"request_results"`
}

type JSONRequestResult struct {
	RequestName string                `json:"request_name"`
	StatusCode  int                   `json:"status_code"`
	Passed      bool                  `json:"passed"`
	Skipped     bool                  `json:"skipped"`
	Error       string                `json:"error,omitempty"`
	Duration    string                `json:"duration"`
	Assertions  []JSONAssertionResult `json:"assertions,omitempty"`
}

type JSONAssertionResult struct {
	Field   string `json:"field"`
	Op      string `json:"op"`
	Value   string `json:"value"`
	Passed  bool   `json:"passed"`
	Message string `json:"message,omitempty"`
}

func (r *JSONReporter) Report(results []RunResult) ([]byte, error) {
	report := JSONReport{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Results:     make([]JSONRunResult, 0, len(results)),
	}

	var totalPassed, totalFailed, totalSkipped, totalTests int
	var totalDuration time.Duration

	for _, result := range results {
		runResult := JSONRunResult{
			CollectionName: result.CollectionName,
			Iteration:      result.Iteration,
			Total:          result.Total,
			Passed:         result.Passed,
			Failed:         result.Failed,
			Skipped:        result.Skipped,
			Duration:       result.Duration.String(),
			RequestResults: make([]JSONRequestResult, 0, len(result.RequestResults)),
		}

		for _, reqResult := range result.RequestResults {
			assertions := make([]JSONAssertionResult, 0, len(reqResult.AssertionResults))
			for _, ar := range reqResult.AssertionResults {
				assertions = append(assertions, JSONAssertionResult{
					Field:   ar.Field,
					Op:      ar.Op,
					Value:   ar.Value,
					Passed:  ar.Passed,
					Message: ar.Message,
				})
			}

			runResult.RequestResults = append(runResult.RequestResults, JSONRequestResult{
				RequestName: reqResult.RequestName,
				StatusCode:  reqResult.StatusCode,
				Passed:      reqResult.Passed,
				Skipped:     reqResult.Skipped,
				Error:       reqResult.Error,
				Duration:    reqResult.Duration.String(),
				Assertions:  assertions,
			})
		}

		totalPassed += result.Passed
		totalFailed += result.Failed
		totalSkipped += result.Skipped
		totalTests += result.Total
		totalDuration += result.Duration

		report.Results = append(report.Results, runResult)
	}

	report.TotalRuns = len(results)
	report.TotalTests = totalTests
	report.Passed = totalPassed
	report.Failed = totalFailed
	report.Skipped = totalSkipped
	report.Duration = totalDuration.String()

	return json.MarshalIndent(report, "", "  ")
}

func (r *JSONReporter) WriteToFile(results []RunResult, path string) error {
	content, err := r.Report(results)
	if err != nil {
		return err
	}
	return os.WriteFile(path, content, 0644)
}

type HTMLReporter struct{}

func NewHTMLReporter() *HTMLReporter {
	return &HTMLReporter{}
}

func (r *HTMLReporter) Name() string { return "html" }

func (r *HTMLReporter) Report(results []RunResult) ([]byte, error) {
	var sb strings.Builder

	sb.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Test Report</title>
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; padding: 20px; }
.container { max-width: 1200px; margin: 0 auto; }
h1 { color: #333; margin-bottom: 20px; }
.summary { background: white; border-radius: 8px; padding: 20px; margin-bottom: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
.summary-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(120px, 1fr)); gap: 15px; margin-top: 15px; }
.summary-item { text-align: center; padding: 15px; border-radius: 6px; }
.summary-item.passed { background: #d4edda; color: #155724; }
.summary-item.failed { background: #f8d7da; color: #721c24; }
.summary-item.skipped { background: #fff3cd; color: #856404; }
.summary-item.total { background: #e9ecef; color: #495057; }
.summary-item .value { font-size: 32px; font-weight: bold; }
.summary-item .label { font-size: 12px; text-transform: uppercase; margin-top: 5px; }
.iteration { background: white; border-radius: 8px; margin-bottom: 15px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); overflow: hidden; }
.iteration-header { background: #333; color: white; padding: 15px; display: flex; justify-content: space-between; align-items: center; }
.iteration-title { font-weight: 600; }
.iteration-duration { font-size: 14px; opacity: 0.8; }
.iteration-body { padding: 15px; }
.test-result { padding: 12px; border-radius: 6px; margin-bottom: 8px; display: flex; align-items: center; gap: 10px; }
.test-result.passed { background: #d4edda; border-left: 4px solid #28a745; }
.test-result.failed { background: #f8d7da; border-left: 4px solid #dc3545; }
.test-result.skipped { background: #fff3cd; border-left: 4px solid #ffc107; }
.test-result .status { font-weight: bold; font-size: 12px; width: 60px; }
.test-result.passed .status { color: #155724; }
.test-result.failed .status { color: #721c24; }
.test-result.skipped .status { color: #856404; }
.test-result .name { flex: 1; color: #333; }
.test-result .duration { font-size: 12px; color: #666; }
.test-result .message { font-size: 12px; color: #666; margin-left: auto; }
.generated { text-align: center; color: #666; font-size: 12px; margin-top: 20px; }
</style>
</head>
<body>
<div class="container">
<h1>Test Report</h1>
`)

	var totalPassed, totalFailed, totalSkipped, totalTests int
	var totalDuration time.Duration

	for _, result := range results {
		totalPassed += result.Passed
		totalFailed += result.Failed
		totalSkipped += result.Skipped
		totalTests += result.Total
		totalDuration += result.Duration
	}

	sb.WriteString(`<div class="summary">
<div>Summary</div>
<div class="summary-grid">
<div class="summary-item total"><div class="value">`)
	sb.WriteString(fmt.Sprintf("%d", totalTests))
	sb.WriteString(`</div><div class="label">Total</div></div>
<div class="summary-item passed"><div class="value">`)
	sb.WriteString(fmt.Sprintf("%d", totalPassed))
	sb.WriteString(`</div><div class="label">Passed</div></div>
<div class="summary-item failed"><div class="value">`)
	sb.WriteString(fmt.Sprintf("%d", totalFailed))
	sb.WriteString(`</div><div class="label">Failed</div></div>
<div class="summary-item skipped"><div class="value">`)
	sb.WriteString(fmt.Sprintf("%d", totalSkipped))
	sb.WriteString(`</div><div class="label">Skipped</div></div>
</div>
</div>
`)

	for _, result := range results {
		sb.WriteString(`<div class="iteration">
<div class="iteration-header">
<span class="iteration-title">`)
		sb.WriteString(html.EscapeString(result.CollectionName))
		sb.WriteString(` - Iteration `)
		sb.WriteString(fmt.Sprintf("%d", result.Iteration))
		sb.WriteString(`</span>
<span class="iteration-duration">`)
		sb.WriteString(result.Duration.String())
		sb.WriteString(`</span>
</div>
<div class="iteration-body">
`)

		for _, reqResult := range result.RequestResults {
			status := "PASS"
			class := "passed"
			message := ""
			if reqResult.Skipped {
				status = "SKIP"
				class = "skipped"
			} else if !reqResult.Passed {
				status = "FAIL"
				class = "failed"
				if reqResult.Error != "" {
					message = reqResult.Error
				} else {
					message = fmt.Sprintf("Status: %d", reqResult.StatusCode)
				}
			}

		sb.WriteString(`<div class="test-result `)
		sb.WriteString(class)
		sb.WriteString(`">
<span class="status">`)
		sb.WriteString(status)
		sb.WriteString(`</span>
<span class="name">`)
		sb.WriteString(html.EscapeString(reqResult.RequestName))
		sb.WriteString(`</span>
<span class="duration">`)
		sb.WriteString(reqResult.Duration.String())
		sb.WriteString(`</span>`)
		if message != "" {
			sb.WriteString(`<span class="message">`)
			sb.WriteString(html.EscapeString(message))
			sb.WriteString(`</span>`)
		}
		sb.WriteString(`</div>
`)
		}

		sb.WriteString(`</div></div>
`)
	}

	sb.WriteString(`<div class="generated">Generated at `)
	sb.WriteString(time.Now().UTC().Format(time.RFC3339))
	sb.WriteString(`</div>
</div>
</body>
</html>
`)

	return []byte(sb.String()), nil
}

func (r *HTMLReporter) WriteToFile(results []RunResult, path string) error {
	content, err := r.Report(results)
	if err != nil {
		return err
	}
	return os.WriteFile(path, content, 0644)
}

type ConsoleReporter struct{}

func NewConsoleReporter() *ConsoleReporter {
	return &ConsoleReporter{}
}

func (r *ConsoleReporter) Name() string { return "console" }

func (r *ConsoleReporter) Report(results []RunResult) ([]byte, error) {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(formatter.Cyan + formatter.Bold)
	sb.WriteString("═══════════════════════════════════════════")
	sb.WriteString(formatter.Reset + "\n")
	sb.WriteString(formatter.Cyan + formatter.Bold)
	sb.WriteString("           TEST REPORT")
	sb.WriteString(formatter.Reset + "\n")
	sb.WriteString(formatter.Cyan + formatter.Bold)
	sb.WriteString("═══════════════════════════════════════════")
	sb.WriteString(formatter.Reset + "\n\n")

	var totalPassed, totalFailed, totalSkipped, totalTests int
	var totalDuration time.Duration

	for _, result := range results {
		totalPassed += result.Passed
		totalFailed += result.Failed
		totalSkipped += result.Skipped
		totalTests += result.Total
		totalDuration += result.Duration

		sb.WriteString(formatter.Bold)
		sb.WriteString(fmt.Sprintf("▶ Iteration %d: %s", result.Iteration, result.CollectionName))
		sb.WriteString(formatter.Reset)
		sb.WriteString(fmt.Sprintf(" (%s)\n", result.Duration.String()))

		for _, reqResult := range result.RequestResults {
			if reqResult.Skipped {
				sb.WriteString(formatter.Yellow + formatter.Bold + "  ⊘ SKIP" + formatter.Reset + " ")
				sb.WriteString(reqResult.RequestName)
				sb.WriteString("\n")
			} else if reqResult.Error != "" {
				sb.WriteString(formatter.Red + formatter.Bold + "  ✗ FAIL" + formatter.Reset + " ")
				sb.WriteString(reqResult.RequestName)
				sb.WriteString(": ")
				sb.WriteString(formatter.Red + reqResult.Error + formatter.Reset)
				sb.WriteString("\n")
			} else if reqResult.Passed {
				sb.WriteString(formatter.Green + formatter.Bold + "  ✓ PASS" + formatter.Reset + " ")
				sb.WriteString(reqResult.RequestName)
				sb.WriteString(fmt.Sprintf(" (%dms)", reqResult.Duration.Milliseconds()))
				sb.WriteString("\n")
			} else {
				sb.WriteString(formatter.Red + formatter.Bold + "  ✗ FAIL" + formatter.Reset + " ")
				sb.WriteString(reqResult.RequestName)
				sb.WriteString(fmt.Sprintf(" (status: %d)", reqResult.StatusCode))
				sb.WriteString("\n")
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString(formatter.Bold)
	sb.WriteString("───────────────────────────────────────────")
	sb.WriteString(formatter.Reset + "\n")

	sb.WriteString(fmt.Sprintf("Total:   %s%d%s", formatter.Bold, totalTests, formatter.Reset))
	sb.WriteString(fmt.Sprintf("  |  Passed:  %s%d%s", formatter.Green, totalPassed, formatter.Reset))
	sb.WriteString(fmt.Sprintf("  |  Failed:  %s%d%s", formatter.Red, totalFailed, formatter.Reset))
	sb.WriteString(fmt.Sprintf("  |  Skipped: %s%d%s", formatter.Yellow, totalSkipped, formatter.Reset))
	sb.WriteString(fmt.Sprintf("  |  Duration: %s\n", totalDuration.String()))

	return []byte(sb.String()), nil
}

func (r *ConsoleReporter) WriteToFile(results []RunResult, path string) error {
	content, err := r.Report(results)
	if err != nil {
		return err
	}
	return os.WriteFile(path, content, 0644)
}

var reporters = make(map[string]func() Reporter)

func RegisterReporter(name string, factory func() Reporter) {
	reporters[name] = factory
}

func GetReporter(name string) Reporter {
	if factory, ok := reporters[name]; ok {
		return factory()
	}
	return nil
}

func AvailableReporters() []string {
	names := make([]string, 0, len(reporters))
	for name := range reporters {
		names = append(names, name)
	}
	return names
}

func init() {
	RegisterReporter("junit", func() Reporter { return NewJUnitXMLReporter() })
	RegisterReporter("json", func() Reporter { return NewJSONReporter() })
	RegisterReporter("html", func() Reporter { return NewHTMLReporter() })
	RegisterReporter("console", func() Reporter { return NewConsoleReporter() })
}
