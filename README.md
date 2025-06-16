# llm-json-parse

HTTP gateway server that enforces JSON schema validation on LLM responses using structured outputs.

## Quick Start

### Prerequisites

You need a running LLM server that supports the OpenAI chat API format:

```bash
# Option 1: llama-server (recommended)
llama-server -hf unsloth/gemma-3-4b-it-GGUF:Q4_K_XL

# Option 2: Any OpenAI-compatible API server on port 8080
```

### Running the Demo

1. **Start the Demo Environment**:
```bash
# This will detect your running LLM server and start the gateway
make start-demo
```

2. **Try the Interactive Demo**:
```bash
# Interactive demo with custom schema and prompt input
make demo-interactive

# Or test pre-built examples
make example-person   # Person schema validation
make example-recipe   # Recipe schema validation  
make example-invalid  # Invalid schema (error case)

# Stop the demo when done
make stop-demo
```

### Manual Setup

```bash
# Build and run gateway manually
go build -o bin/server ./cmd/server
./bin/server
```

## Interactive Demo

The demo environment includes:
- **Real LLM Server** (port 8080) - Your running llama-server or compatible API
- **JSON Schema Gateway** (port 8081) - Validation service with automatic port detection
- **Interactive Mode** - Custom schema and prompt input with real-time LLM responses
- **Pre-built Examples** - Common use cases with working schemas

### Interactive Demo Usage

**Custom Schema Input:**
```bash
$ make demo-interactive
🎯 Interactive JSON Schema Demo
==============================

Available schema templates:
1. Person (name, age, occupation)
2. Product (name, price, category)  
3. Event (title, date, location)
4. Custom (enter your own schema)

Choose a schema template (1-4) [1]: 2
📋 Using Product schema
Enter your prompt for the LLM [Describe a wireless headphone product that costs $199]: Create a gaming laptop under $1500

📤 Request:
{
  "schema": {...},
  "messages": [{"role": "user", "content": "Create a gaming laptop under $1500"}]
}

📥 Response:
{
  "name": "Gaming Laptop Pro",
  "price": 1299,
  "category": "Electronics",
  "description": "High-performance gaming laptop with RTX graphics"
}
✅ Response validates successfully against the schema!
```

### Example Output

**Person Schema Validation:**
```bash
$ make example-person
👤 Testing person schema validation...
Request:
{
  "schema": {
    "type": "object", 
    "properties": {
      "name": {"type": "string"},
      "age": {"type": "number"}
    },
    "required": ["name"]
  },
  "messages": [{"role": "user", "content": "Tell me about John who is 25"}]
}

Response:
{
  "name": "John",
  "age": 25,
  "occupation": "Software Developer"
}
```

**Invalid Schema (Error Handling):**
```bash
$ make example-invalid
❌ Testing invalid schema (should return 400 error)...
Response:
{
  "error": "Invalid schema",
  "details": "invalid schema: compile schema: '/type' does not validate..."
}
```

## Environment Variables

- `LLM_SERVER_URL` - LLM server URL (default: http://localhost:8080)
- `PORT` - Gateway server port (default: 8081)

## Features

- JSON schema validation of LLM responses
- Support for structured outputs via llama-server
- Detailed validation error reporting
- Health check endpoint
- Comprehensive integration test suite with interactive output

## Testing

Integration tests require a running LLM server:

```bash
# Start your LLM server first
llama-server -hf unsloth/gemma-3-4b-it-GGUF:Q4_K_XL

# Run tests with different verbosity levels
make test-integration         # Standard test output
make test-integration-verbose # Interactive pretty-printed output
make test-all-verbose        # All tests with verbose output

# Tests will skip automatically if no LLM server is detected
```

### Verbose Test Output

When `VERBOSE_TESTS=true`, integration tests display:
- 📊 Clean HTTP request/response logging with timing
- 📋 JSON schema visualization
- ✅ Step-by-step progress tracking with emojis
- 📈 Concurrent test performance metrics
- 🔍 Detailed assertion comparisons
