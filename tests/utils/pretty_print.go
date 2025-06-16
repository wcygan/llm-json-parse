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
	
	if !VerboseEnabled() {
		return string(jsonBytes)
	}
	
	// Add syntax highlighting
	jsonStr := string(jsonBytes)
	jsonStr = strings.ReplaceAll(jsonStr, "\"", Cyan+"\""+Reset)
	jsonStr = strings.ReplaceAll(jsonStr, ":", ":"+Reset)
	jsonStr = strings.ReplaceAll(jsonStr, "{", Bold+"{"+Reset)
	jsonStr = strings.ReplaceAll(jsonStr, "}", Bold+"}"+Reset)
	jsonStr = strings.ReplaceAll(jsonStr, "[", Bold+"["+Reset)
	jsonStr = strings.ReplaceAll(jsonStr, "]", Bold+"]"+Reset)
	
	return jsonStr
}

// PrettyPrintRequest formats an HTTP request for display
func PrettyPrintRequest(method, url string, body interface{}) string {
	if !VerboseEnabled() {
		return ""
	}
	
	var output strings.Builder
	
	output.WriteString(fmt.Sprintf("%s📤 REQUEST%s %s%s %s%s\n", 
		Bold+Blue, Reset, Bold+method+Reset, Reset, url, Reset))
	output.WriteString(strings.Repeat("━", 70) + "\n")
	
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
	
	// Status color based on HTTP status code
	statusColor := Green
	if statusCode >= 400 && statusCode < 500 {
		statusColor = Yellow
	} else if statusCode >= 500 {
		statusColor = Red
	}
	
	output.WriteString(fmt.Sprintf("\n%s📥 RESPONSE%s %s(%d)%s\n", 
		Bold+Blue, Reset, statusColor, statusCode, Reset))
	output.WriteString(strings.Repeat("━", 70) + "\n")
	
	if body != nil {
		output.WriteString(PrettyPrintJSON(body))
		output.WriteString("\n")
	}
	
	output.WriteString(fmt.Sprintf("\n%s⏱️  Duration:%s %v\n", 
		Purple, Reset, duration))
	
	return output.String()
}

// PrettyPrintTestHeader creates a decorative test header
func PrettyPrintTestHeader(testName string) string {
	if !VerboseEnabled() {
		return ""
	}
	
	var output strings.Builder
	
	output.WriteString(fmt.Sprintf("\n%s🧪 Test:%s %s%s%s\n", 
		Bold+Cyan, Reset, Bold, testName, Reset))
	output.WriteString(strings.Repeat("━", 70) + "\n")
	
	return output.String()
}

// PrettyPrintValidation shows validation results
func PrettyPrintValidation(passed bool, message string) string {
	if !VerboseEnabled() {
		return ""
	}
	
	var output strings.Builder
	
	if passed {
		output.WriteString(fmt.Sprintf("%s✅ Validation: PASSED%s - %s\n", 
			Green, Reset, message))
	} else {
		output.WriteString(fmt.Sprintf("%s❌ Validation: FAILED%s - %s\n", 
			Red, Reset, message))
	}
	
	return output.String()
}

// PrettyPrintSchema displays a JSON schema with highlighting
func PrettyPrintSchema(schema interface{}) string {
	if !VerboseEnabled() {
		return ""
	}
	
	var output strings.Builder
	
	output.WriteString(fmt.Sprintf("%s📋 Schema:%s\n", Bold+Purple, Reset))
	output.WriteString(strings.Repeat("─", 40) + "\n")
	output.WriteString(PrettyPrintJSON(schema))
	output.WriteString("\n")
	
	return output.String()
}

// PrettyPrintError formats error messages
func PrettyPrintError(err error) string {
	if !VerboseEnabled() || err == nil {
		return ""
	}
	
	return fmt.Sprintf("%s❌ Error:%s %v\n", Red, Reset, err)
}

// PrettyPrintSuccess formats success messages
func PrettyPrintSuccess(message string) string {
	if !VerboseEnabled() {
		return ""
	}
	
	return fmt.Sprintf("%s✅ Success:%s %s\n", Green, Reset, message)
}

// PrettyPrintWarning formats warning messages
func PrettyPrintWarning(message string) string {
	if !VerboseEnabled() {
		return ""
	}
	
	return fmt.Sprintf("%s⚠️  Warning:%s %s\n", Yellow, Reset, message)
}

// PrettyPrintSeparator creates a visual separator
func PrettyPrintSeparator() string {
	if !VerboseEnabled() {
		return ""
	}
	
	return strings.Repeat("═", 70) + "\n"
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