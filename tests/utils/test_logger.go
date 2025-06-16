package utils

import (
	"encoding/json"
	"fmt"
	"strings"
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
	PrintToConsole(fmt.Sprintf("\n🏁 Test Completed: %v\n", duration))
	PrintToConsole(PrettyPrintSeparator())
}

// LogStep logs a test step with numbering
func (tl *TestLogger) LogStep(stepNum int, description string) {
	if VerboseEnabled() {
		PrintToConsole(fmt.Sprintf("\n📝 Step %d: %s\n", stepNum, description))
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
		PrintToConsole(fmt.Sprintf("\n🎭 Mock Setup: %s\n", description))
		if mockData != nil {
			PrintToConsole(strings.Repeat("─", 40) + "\n")
			PrintToConsole(PrettyPrintJSON(mockData))
			PrintToConsole("\n")
		}
	}
}

// LogAssertion logs test assertions
func (tl *TestLogger) LogAssertion(description string, expected, actual interface{}) {
	if VerboseEnabled() {
		PrintToConsole(fmt.Sprintf("\n🔍 Assertion: %s\n", description))
		PrintToConsole("Expected:\n")
		PrintToConsole(PrettyPrintJSON(expected))
		PrintToConsole("\nActual:\n")
		PrintToConsole(PrettyPrintJSON(actual))
		PrintToConsole("\n")
	}
}

// LogJSONComparison logs a detailed JSON comparison
func (tl *TestLogger) LogJSONComparison(description string, expected, actual interface{}) {
	if VerboseEnabled() {
		PrintToConsole(fmt.Sprintf("\n🔍 JSON Comparison: %s\n", description))
		
		expectedJSON, _ := json.MarshalIndent(expected, "", "  ")
		actualJSON, _ := json.MarshalIndent(actual, "", "  ")
		
		if string(expectedJSON) == string(actualJSON) {
			PrintToConsole(PrettyPrintSuccess("JSON structures match"))
		} else {
			PrintToConsole(PrettyPrintWarning("JSON structures differ"))
			PrintToConsole(fmt.Sprintf("Expected:\n%s\n", expectedJSON))
			PrintToConsole(fmt.Sprintf("Actual:\n%s\n", actualJSON))
		}
	}
}

// LogTestSummary logs a summary of test results
func (tl *TestLogger) LogTestSummary(passed bool, details string) {
	if VerboseEnabled() {
		if passed {
			PrintToConsole(fmt.Sprintf("\n🎉 TEST PASSED: %s - %s\n", tl.testName, details))
		} else {
			PrintToConsole(fmt.Sprintf("\n💥 TEST FAILED: %s - %s\n", tl.testName, details))
		}
	}
}

// LogConcurrentTestStart logs the start of concurrent test execution
func (tl *TestLogger) LogConcurrentTestStart(numRequests int) {
	if VerboseEnabled() {
		PrintToConsole(fmt.Sprintf("\n🚀 Concurrent Test: Starting %d parallel requests\n", numRequests))
	}
}

// LogConcurrentTestResult logs concurrent test results
func (tl *TestLogger) LogConcurrentTestResult(successful, failed int, totalDuration time.Duration) {
	if VerboseEnabled() {
		PrintToConsole("\n📊 Concurrent Results:\n")
		PrintToConsole(fmt.Sprintf("  ✅ Successful: %d\n", successful))
		PrintToConsole(fmt.Sprintf("  ❌ Failed: %d\n", failed))
		PrintToConsole(fmt.Sprintf("  ⏱️  Total Duration: %v\n", totalDuration))
		if successful+failed > 0 {
			PrintToConsole(fmt.Sprintf("  📈 Average per request: %v\n", totalDuration/time.Duration(successful+failed)))
		}
	}
}