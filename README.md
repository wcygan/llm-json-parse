# llm-json-parse

HTTP gateway server that enforces JSON schema validation on LLM responses using structured outputs.

## Quick Start

1. **Start LLM Server** (llama-server or compatible):
```bash
llama-server -hf unsloth/gemma-3-4b-it-GGUF:Q4_K_XL
```

2. **Build and Run Gateway**:
```bash
go build -o bin/server ./cmd/server
./bin/server
```

3. **Send Validated Query**:
```bash
curl -X POST http://localhost:8081/v1/validated-query \
  -H "Content-Type: application/json" \
  -d '{
    "schema": {
      "type": "object",
      "properties": {
        "recipe_name": {"type": "string"},
        "ingredients": {
          "type": "array",
          "items": {"type": "string"}
        }
      },
      "required": ["recipe_name", "ingredients"]
    },
    "messages": [
      {"role": "user", "content": "Give me a simple chocolate chip cookie recipe"}
    ]
  }' | jq
```

## Environment Variables

- `LLM_SERVER_URL` - LLM server URL (default: http://localhost:8080)
- `PORT` - Gateway server port (default: 8081)

## Features

- JSON schema validation of LLM responses
- Support for structured outputs via llama-server
- Detailed validation error reporting
- Health check endpoint
