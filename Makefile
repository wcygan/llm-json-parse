.PHONY: build test test-unit test-integration test-all clean run dev docker-build docker-run

# Build targets
build:
	go build -o bin/server ./cmd/server

clean:
	rm -rf bin/

# Test targets
test-unit:
	go test -v ./internal/... ./pkg/...

test-integration:
	go test -v ./tests/integration/...

test-integration-verbose:
	VERBOSE_TESTS=true go test -v ./tests/integration/...

test-all: test-unit test-integration

test-all-verbose: test-unit
	VERBOSE_TESTS=true go test -v ./tests/integration/...

test:
	go test -v ./...

test-verbose:
	VERBOSE_TESTS=true go test -v ./...

# Coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Development
run: build
	./bin/server

dev:
	go run ./cmd/server

# Linting and formatting
fmt:
	go fmt ./...

vet:
	go vet ./...

lint: fmt vet

# Docker targets
docker-build:
	docker build -t llm-json-parse .

docker-run: docker-build
	docker run -p 8081:8081 llm-json-parse

# CI targets
ci: lint test-all

# Demo targets
start-demo:
	@echo "🚀 Starting demo environment..."
	@echo "📋 Prerequisites: LLM server must be running on port 8080"
	@echo "   Example: llama-server -hf unsloth/gemma-3-4b-it-GGUF:Q4_K_XL"
	@echo ""
	./scripts/start-demo.sh

stop-demo:
	@echo "🛑 Stopping demo environment..."
	./scripts/stop-demo.sh

# Example requests (requires demo to be running)
example-person:
	@if [ ! -f /tmp/demo_gateway_port.txt ]; then echo "❌ Demo not running. Run 'make start-demo' first."; exit 1; fi
	@GATEWAY_PORT=$$(cat /tmp/demo_gateway_port.txt); \
	echo "👤 Testing person schema validation..."; \
	echo "Request:"; \
	echo '{"schema":{"type":"object","properties":{"name":{"type":"string"},"age":{"type":"number"}},"required":["name"]},"messages":[{"role":"user","content":"Tell me about a person named John who is 25"}]}' | jq; \
	echo ""; \
	echo "Response:"; \
	curl -s -X POST http://localhost:$$GATEWAY_PORT/v1/validated-query \
		-H "Content-Type: application/json" \
		-d '{"schema":{"type":"object","properties":{"name":{"type":"string"},"age":{"type":"number"}},"required":["name"]},"messages":[{"role":"user","content":"Tell me about a person named John who is 25"}]}' | jq

example-recipe:
	@if [ ! -f /tmp/demo_gateway_port.txt ]; then echo "❌ Demo not running. Run 'make start-demo' first."; exit 1; fi
	@GATEWAY_PORT=$$(cat /tmp/demo_gateway_port.txt); \
	echo "🍪 Testing recipe schema validation..."; \
	echo "Request:"; \
	echo '{"schema":{"type":"object","properties":{"name":{"type":"string"},"ingredients":{"type":"array","items":{"type":"string"}},"prep_time":{"type":"number"}},"required":["name","ingredients"]},"messages":[{"role":"user","content":"Give me a chocolate chip cookie recipe"}]}' | jq; \
	echo ""; \
	echo "Response:"; \
	curl -s -X POST http://localhost:$$GATEWAY_PORT/v1/validated-query \
		-H "Content-Type: application/json" \
		-d '{"schema":{"type":"object","properties":{"name":{"type":"string"},"ingredients":{"type":"array","items":{"type":"string"}},"prep_time":{"type":"number"}},"required":["name","ingredients"]},"messages":[{"role":"user","content":"Give me a chocolate chip cookie recipe"}]}' | jq

example-invalid:
	@if [ ! -f /tmp/demo_gateway_port.txt ]; then echo "❌ Demo not running. Run 'make start-demo' first."; exit 1; fi
	@GATEWAY_PORT=$$(cat /tmp/demo_gateway_port.txt); \
	echo "❌ Testing invalid schema (should return 400 error)..."; \
	echo "Request:"; \
	echo '{"schema":{"type":"invalid_type"},"messages":[{"role":"user","content":"Test invalid schema"}]}' | jq; \
	echo ""; \
	echo "Response:"; \
	curl -s -X POST http://localhost:$$GATEWAY_PORT/v1/validated-query \
		-H "Content-Type: application/json" \
		-d '{"schema":{"type":"invalid_type"},"messages":[{"role":"user","content":"Test invalid schema"}]}' | jq

health-check:
	@if [ ! -f /tmp/demo_gateway_port.txt ]; then echo "❌ Demo not running. Run 'make start-demo' first."; exit 1; fi
	@GATEWAY_PORT=$$(cat /tmp/demo_gateway_port.txt); \
	echo "🏥 Checking server health..."; \
	curl -f http://localhost:$$GATEWAY_PORT/health && echo " ✅ Server is healthy" || echo " ❌ Server not responding"

# Interactive demo with custom input
demo-interactive:
	@echo "🎯 Interactive JSON Schema Demo"
	@echo "=============================="
	@echo ""
	@echo "This demo lets you test JSON schema validation with custom input."
	@echo "The LLM will be asked to generate a response matching your schema."
	@echo ""
	@./scripts/interactive-demo.sh

# Legacy example (kept for compatibility)
example-request: example-person