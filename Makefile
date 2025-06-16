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

# Example usage
example-request:
	@echo "Starting example request (requires server running on :8081)..."
	curl -X POST http://localhost:8081/v1/validated-query \
		-H "Content-Type: application/json" \
		-d '{ \
			"schema": { \
				"type": "object", \
				"properties": { \
					"name": {"type": "string"}, \
					"age": {"type": "number"} \
				}, \
				"required": ["name"] \
			}, \
			"messages": [ \
				{"role": "user", "content": "Tell me about a person named John who is 25"} \
			] \
		}' | jq

health-check:
	@echo "Checking server health..."
	curl -f http://localhost:8081/health || echo "Server not responding"