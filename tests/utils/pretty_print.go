package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// ANSI color codes
const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	Gray   = "\033[37m"
	Bold   = "\033[1m"
)

// VerboseEnabled checks if verbose test output is enabled
func VerboseEnabled() bool {
	return os.Getenv("VERBOSE_TESTS") == "true" || os.Getenv("VERBOSE_TESTS") == "1"
}

// PrettyPrintJSON formats JSON with indentation and optional syntax highlighting
func PrettyPrintJSON(data interface{}) string {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error formatting JSON: %v", err)
	}
	
	// Always return clean JSON without escape codes for readability
	return string(jsonBytes)
}

// PrettyPrintRequest formats an HTTP request for display
func PrettyPrintRequest(method, url string, body interface{}) string {
	if !VerboseEnabled() {
		return ""
	}
	
	var output strings.Builder
	
	output.WriteString(fmt.Sprintf("📤 REQUEST: %s %s\n", method, url))
	output.WriteString(strings.Repeat("─", 80) + "\n")
	
	if body != nil {
		output.WriteString(PrettyPrintJSON(body))
		output.WriteString("\n")
	}
	
	return output.String()
}

// PrettyPrintResponse formats an HTTP response for display
func PrettyPrintResponse(statusCode int, body interface{}, duration time.Duration) string {
	if !VerboseEnabled() {
		return ""
	}
	
	var output strings.Builder
	
	// Add status message based on code
	statusMsg := getStatusMessage(statusCode)
	
	output.WriteString(fmt.Sprintf("\n📥 RESPONSE: %d %s\n", statusCode, statusMsg))
	output.WriteString(strings.Repeat("─", 80) + "\n")
	
	if body != nil {
		output.WriteString(PrettyPrintJSON(body))
		output.WriteString("\n")
	}
	
	output.WriteString(fmt.Sprintf("\n⏱️  Duration: %v\n", duration))
	
	return output.String()
}

// getStatusMessage returns a human-readable status message
func getStatusMessage(statusCode int) string {
	switch statusCode {
	case 200:
		return "OK"
	case 201:
		return "Created"
	case 400:
		return "Bad Request"
	case 401:
		return "Unauthorized"
	case 403:
		return "Forbidden"
	case 404:
		return "Not Found"
	case 422:
		return "Unprocessable Entity"
	case 500:
		return "Internal Server Error"
	default:
		if statusCode >= 200 && statusCode < 300 {
			return "Success"
		} else if statusCode >= 400 && statusCode < 500 {
			return "Client Error"
		} else if statusCode >= 500 {
			return "Server Error"
		}
		return "Unknown"
	}
}

// PrettyPrintTestHeader creates a decorative test header
func PrettyPrintTestHeader(testName string) string {
	if !VerboseEnabled() {
		return ""
	}
	
	var output strings.Builder
	
	output.WriteString(fmt.Sprintf("\n🧪 Test: %s\n", testName))
	output.WriteString(strings.Repeat("═", 80) + "\n")
	
	return output.String()
}

// PrettyPrintValidation shows validation results
func PrettyPrintValidation(passed bool, message string) string {
	if !VerboseEnabled() {
		return ""
	}
	
	var output strings.Builder
	
	if passed {
		output.WriteString(fmt.Sprintf("✅ PASSED: %s\n", message))
	} else {
		output.WriteString(fmt.Sprintf("❌ FAILED: %s\n", message))
	}
	
	return output.String()
}

// PrettyPrintSchema displays a JSON schema with highlighting
func PrettyPrintSchema(schema interface{}) string {
	if !VerboseEnabled() {
		return ""
	}
	
	var output strings.Builder
	
	output.WriteString("📋 JSON Schema:\n")
	output.WriteString(strings.Repeat("─", 50) + "\n")
	output.WriteString(PrettyPrintJSON(schema))
	output.WriteString("\n")
	
	return output.String()
}

// PrettyPrintError formats error messages
func PrettyPrintError(err error) string {
	if !VerboseEnabled() || err == nil {
		return ""
	}
	
	return fmt.Sprintf("❌ Error: %v\n", err)
}

// PrettyPrintSuccess formats success messages
func PrettyPrintSuccess(message string) string {
	if !VerboseEnabled() {
		return ""
	}
	
	return fmt.Sprintf("✅ Success: %s\n", message)
}

// PrettyPrintWarning formats warning messages
func PrettyPrintWarning(message string) string {
	if !VerboseEnabled() {
		return ""
	}
	
	return fmt.Sprintf("⚠️  Warning: %s\n", message)
}

// PrettyPrintSeparator creates a visual separator
func PrettyPrintSeparator() string {
	if !VerboseEnabled() {
		return ""
	}
	
	return strings.Repeat("═", 80) + "\n"
}

// PrintToConsole outputs formatted text to console (only if verbose mode)
func PrintToConsole(text string) {
	if VerboseEnabled() && text != "" {
		fmt.Print(text)
	}
}

// FormatHTTPRequest creates a formatted string representation of an HTTP request
func FormatHTTPRequest(req *http.Request) string {
	if !VerboseEnabled() {
		return ""
	}
	
	var buf bytes.Buffer
	
	// Read body if present
	if req.Body != nil {
		bodyBytes := make([]byte, 1024)
		n, _ := req.Body.Read(bodyBytes)
		if n > 0 {
			var jsonBody interface{}
			if err := json.Unmarshal(bodyBytes[:n], &jsonBody); err == nil {
				buf.WriteString(PrettyPrintJSON(jsonBody))
			} else {
				buf.WriteString(string(bodyBytes[:n]))
			}
		}
	}
	
	return buf.String()
}