#!/bin/bash
set -e

echo "🚀 Starting LLM JSON Parse Demo"
echo "==============================="

# Check if real LLM server is running on port 8080
if ! lsof -Pi :8080 -sTCP:LISTEN -t >/dev/null ; then
    echo "❌ No LLM server found on port 8080"
    echo "    Please start an LLM server first:"
    echo "    llama-server -hf unsloth/gemma-3-4b-it-GGUF:Q4_K_XL"
    echo "    or any compatible OpenAI-style API server"
    exit 1
fi

# Test if it's a working LLM server
echo "🔍 Checking LLM server compatibility..."
if ! curl -s -X POST http://localhost:8080/v1/chat/completions \
    -H "Content-Type: application/json" \
    -d '{"messages":[{"role":"user","content":"test"}]}' | grep -q "choices"; then
    echo "❌ Server on port 8080 is not compatible with OpenAI chat API"
    echo "    Expected response with 'choices' field"
    exit 1
fi

echo "✅ Found compatible LLM server on port 8080"

# Find an available port for the gateway server (try 8081, 8082, 8083)
GATEWAY_PORT=""
for port in 8081 8082 8083; do
    if ! lsof -Pi :$port -sTCP:LISTEN -t >/dev/null ; then
        GATEWAY_PORT=$port
        break
    fi
done

if [ -z "$GATEWAY_PORT" ]; then
    echo "⚠️  Ports 8081-8083 are all in use. Please free up one of these ports."
    exit 1
fi

echo "🌐 Using port $GATEWAY_PORT for gateway server"

# Build the server
echo "🔨 Building server..."
make build

# Start the gateway server in background
echo "🌐 Starting gateway server on port $GATEWAY_PORT..."
PORT=$GATEWAY_PORT LLM_SERVER_URL=http://localhost:8080 nohup ./bin/server > /tmp/gateway.log 2>&1 &
GATEWAY_PID=$!

# Wait for gateway server to start and check multiple times
echo "⏳ Waiting for gateway server to start..."
for i in {1..10}; do
    sleep 1
    if curl -s http://localhost:$GATEWAY_PORT/health >/dev/null 2>&1; then
        echo "✅ Gateway server is responding on port $GATEWAY_PORT"
        break
    fi
    if [ $i -eq 10 ]; then
        echo "❌ Failed to start gateway server after 10 seconds"
        echo "Last few lines of gateway log:"
        tail -5 /tmp/gateway.log 2>/dev/null || echo "No log file found"
        kill $GATEWAY_PID 2>/dev/null || true
        exit 1
    fi
    echo "  Attempt $i/10: Checking http://localhost:$GATEWAY_PORT/health"
done

echo "✅ Gateway server started (PID: $GATEWAY_PID)"
echo ""
echo "🎉 Demo setup complete!"
echo ""
echo "📖 Available endpoints:"
echo "   • Gateway Server: http://localhost:$GATEWAY_PORT"
echo "   • Health Check:   http://localhost:$GATEWAY_PORT/health"
echo "   • Real LLM:       http://localhost:8080"
echo ""
echo "📝 Try the examples:"
echo "   make demo-interactive"
echo "   make example-person"
echo "   make example-recipe"
echo "   make example-invalid"
echo ""
echo "🛑 To stop the demo:"
echo "   make stop-demo"
echo ""

# Create a file to track the PIDs for cleanup
echo "$GATEWAY_PID" > /tmp/demo_gateway.pid
echo "$GATEWAY_PORT" > /tmp/demo_gateway_port.txt

echo "Demo servers are running in the background..."
echo "Press Ctrl+C to stop or run 'make stop-demo'"

# Setup signal handler for graceful shutdown
cleanup() {
    echo ""
    echo "🛑 Stopping demo servers..."
    kill $GATEWAY_PID 2>/dev/null || true
    rm -f /tmp/demo_gateway.pid /tmp/demo_gateway_port.txt /tmp/gateway.log
    echo "✅ Demo stopped"
    exit 0
}

trap cleanup SIGINT SIGTERM

# Keep the script running
wait