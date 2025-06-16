package utils

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

// TestLogger provides enhanced logging for integration tests
type TestLogger struct {
	t         *testing.T
	testName  string
	startTime time.Time
}

// NewTestLogger creates a new test logger instance
func NewTestLogger(t *testing.T, testName string) *TestLogger {
	logger := &TestLogger{
		t:         t,
		testName:  testName,
		startTime: time.Now(),
	}
	
	// Print test header
	PrintToConsole(PrettyPrintTestHeader(testName))
	
	return logger
}

// LogRequest logs an HTTP request with pretty formatting
func (tl *TestLogger) LogRequest(method, url string, body interface{}) {
	PrintToConsole(PrettyPrintRequest(method, url, body))
}

// LogResponse logs an HTTP response with timing information
func (tl *TestLogger) LogResponse(statusCode int, body interface{}, duration time.Duration) {
	PrintToConsole(PrettyPrintResponse(statusCode, body, duration))
}

// LogSchema logs a JSON schema
func (tl *TestLogger) LogSchema(schema interface{}) {
	PrintToConsole(PrettyPrintSchema(schema))
}

// LogValidation logs validation results
func (tl *TestLogger) LogValidation(passed bool, message string) {
	PrintToConsole(PrettyPrintValidation(passed, message))
}

// LogError logs an error message
func (tl *TestLogger) LogError(err error) {
	PrintToConsole(PrettyPrintError(err))
	if err != nil {
		tl.t.Logf("Error: %v", err)
	}
}

// LogSuccess logs a success message
func (tl *TestLogger) LogSuccess(message string) {
	PrintToConsole(PrettyPrintSuccess(message))
}

// LogWarning logs a warning message
func (tl *TestLogger) LogWarning(message string) {
	PrintToConsole(PrettyPrintWarning(message))
}

// LogSeparator adds a visual separator
func (tl *TestLogger) LogSeparator() {
	PrintToConsole(PrettyPrintSeparator())
}

// Finish logs test completion with total duration
func (tl *TestLogger) Finish() {
	duration := time.Since(tl.startTime)
	PrintToConsole(fmt.Sprintf("\n%s🏁 Test Completed:%s %v\n", 
		Bold+Green, Reset, duration))
	PrintToConsole(PrettyPrintSeparator())
}

// LogStep logs a test step with numbering
func (tl *TestLogger) LogStep(stepNum int, description string) {
	if VerboseEnabled() {
		PrintToConsole(fmt.Sprintf("\n%s📝 Step %d:%s %s\n", 
			Bold+Cyan, stepNum, Reset, description))
	}
}

// LogHTTPCall logs an HTTP call with request and response
func (tl *TestLogger) LogHTTPCall(method, url string, requestBody interface{}, statusCode int, responseBody interface{}, duration time.Duration) {
	tl.LogRequest(method, url, requestBody)
	tl.LogResponse(statusCode, responseBody, duration)
}

// LogMockSetup logs mock setup information
func (tl *TestLogger) LogMockSetup(description string, mockData interface{}) {
	if VerboseEnabled() {
		PrintToConsole(fmt.Sprintf("\n%s🎭 Mock Setup:%s %s\n", 
			Bold+Purple, Reset, description))
		if mockData != nil {
			PrintToConsole(PrettyPrintJSON(mockData))
			PrintToConsole("\n")
		}
	}
}

// LogAssertion logs test assertions
func (tl *TestLogger) LogAssertion(description string, expected, actual interface{}) {
	if VerboseEnabled() {
		PrintToConsole(fmt.Sprintf("\n%s🔍 Assertion:%s %s\n", 
			Bold+Blue, Reset, description))
		PrintToConsole(fmt.Sprintf("%sExpected:%s\n", Gray, Reset))
		PrintToConsole(PrettyPrintJSON(expected))
		PrintToConsole(fmt.Sprintf("\n%sActual:%s\n", Gray, Reset))
		PrintToConsole(PrettyPrintJSON(actual))
		PrintToConsole("\n")
	}
}

// LogJSONComparison logs a detailed JSON comparison
func (tl *TestLogger) LogJSONComparison(description string, expected, actual interface{}) {
	if VerboseEnabled() {
		PrintToConsole(fmt.Sprintf("\n%s🔍 JSON Comparison:%s %s\n", 
			Bold+Blue, Reset, description))
		
		expectedJSON, _ := json.MarshalIndent(expected, "", "  ")
		actualJSON, _ := json.MarshalIndent(actual, "", "  ")
		
		if string(expectedJSON) == string(actualJSON) {
			PrintToConsole(PrettyPrintSuccess("JSON structures match"))
		} else {
			PrintToConsole(PrettyPrintWarning("JSON structures differ"))
			PrintToConsole(fmt.Sprintf("%sExpected:%s\n%s\n", Gray, Reset, expectedJSON))
			PrintToConsole(fmt.Sprintf("%sActual:%s\n%s\n", Gray, Reset, actualJSON))
		}
	}
}

// LogTestSummary logs a summary of test results
func (tl *TestLogger) LogTestSummary(passed bool, details string) {
	if VerboseEnabled() {
		if passed {
			PrintToConsole(fmt.Sprintf("\n%s🎉 TEST PASSED:%s %s - %s\n", 
				Bold+Green, Reset, tl.testName, details))
		} else {
			PrintToConsole(fmt.Sprintf("\n%s💥 TEST FAILED:%s %s - %s\n", 
				Bold+Red, Reset, tl.testName, details))
		}
	}
}

// LogConcurrentTestStart logs the start of concurrent test execution
func (tl *TestLogger) LogConcurrentTestStart(numRequests int) {
	if VerboseEnabled() {
		PrintToConsole(fmt.Sprintf("\n%s🚀 Concurrent Test:%s Starting %d parallel requests\n", 
			Bold+Yellow, Reset, numRequests))
	}
}

// LogConcurrentTestResult logs concurrent test results
func (tl *TestLogger) LogConcurrentTestResult(successful, failed int, totalDuration time.Duration) {
	if VerboseEnabled() {
		PrintToConsole(fmt.Sprintf("\n%s📊 Concurrent Results:%s\n", Bold+Cyan, Reset))
		PrintToConsole(fmt.Sprintf("  %s✅ Successful:%s %d\n", Green, Reset, successful))
		PrintToConsole(fmt.Sprintf("  %s❌ Failed:%s %d\n", Red, Reset, failed))
		PrintToConsole(fmt.Sprintf("  %s⏱️  Total Duration:%s %v\n", Purple, Reset, totalDuration))
		PrintToConsole(fmt.Sprintf("  %s📈 Avg per request:%s %v\n", Blue, Reset, totalDuration/time.Duration(successful+failed)))
	}
}