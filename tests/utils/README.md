# Test Utilities

This package provides enhanced testing utilities for the LLM JSON Parse integration tests.

## Features

### Pretty-Printed Output
- **JSON Syntax Highlighting**: Color-coded JSON with proper indentation
- **HTTP Request/Response Logging**: Formatted display of API interactions
- **Schema Visualization**: Clear presentation of JSON schemas
- **Test Progress Tracking**: Step-by-step test execution with timing

### Verbosity Control
All enhanced output is controlled by the `VERBOSE_TESTS` environment variable:

```bash
# Enable verbose output
export VERBOSE_TESTS=true
go test ./tests/integration/...

# Or use Makefile targets
make test-integration-verbose
```

### Usage Example

```go
func TestMyFeature(t *testing.T) {
    logger := utils.NewTestLogger(t, "my_test_feature")
    defer logger.Finish()

    logger.LogStep(1, "Setting up test data")
    
    // Log schema
    logger.LogSchema(mySchema)
    
    // Log HTTP request/response
    logger.LogRequest("POST", "/api/endpoint", requestData)
    response := makeHTTPCall()
    logger.LogResponse(200, response, duration)
    
    // Log validation results
    logger.LogValidation(passed, "Response matches expected format")
    
    logger.LogTestSummary(true, "All checks passed")
}
```

## Output Format

When `VERBOSE_TESTS=true`, tests produce colorized, structured output:

```
🧪 Test: successful_e2e_flow
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📝 Step 1: Setting up test data and schema
📋 Schema:
────────────────────────────────────────
{
  "type": "object",
  "properties": {
    "name": {"type": "string"}
  }
}

📤 REQUEST POST /v1/validated-query:
{
  "schema": {...},
  "messages": [...]
}

📥 RESPONSE (200 OK):
{
  "name": "Test Response"
}

⏱️  Duration: 1.5ms

✅ Validation: PASSED - Response matches schema
🎉 TEST PASSED: All validations successful
```

## Available Functions

### TestLogger Methods
- `LogStep(num, description)` - Log numbered test steps
- `LogRequest(method, url, body)` - Log HTTP requests
- `LogResponse(status, body, duration)` - Log HTTP responses
- `LogSchema(schema)` - Display JSON schemas
- `LogValidation(passed, message)` - Show validation results
- `LogMockSetup(description, data)` - Log mock configurations
- `LogAssertion(description, expected, actual)` - Log test assertions
- `LogConcurrentTestStart(numRequests)` - Log concurrent test start
- `LogConcurrentTestResult(successful, failed, duration)` - Log concurrent results
- `LogTestSummary(passed, details)` - Final test result summary

### Pretty Print Functions
- `PrettyPrintJSON(data)` - Format JSON with syntax highlighting
- `PrettyPrintRequest(method, url, body)` - Format HTTP requests
- `PrettyPrintResponse(status, body, duration)` - Format HTTP responses
- `PrettyPrintSchema(schema)` - Format JSON schemas
- `PrettyPrintValidation(passed, message)` - Format validation results

## Color Coding

- 🟢 **Green**: Success states, passed validations
- 🔴 **Red**: Failures, errors
- 🟡 **Yellow**: Warnings, non-critical issues
- 🔵 **Blue**: Information, requests/responses
- 🟣 **Purple**: Schemas, timing information
- 🟦 **Cyan**: Test headers, steps

## Integration with CI/CD

In CI environments, verbose output is automatically disabled unless explicitly enabled, ensuring clean build logs while maintaining rich local development experience.