#!/bin/bash

echo "🛑 Stopping LLM JSON Parse Demo..."

# Stop gateway server using saved PID
if [ -f /tmp/demo_gateway.pid ]; then
    GATEWAY_PID=$(cat /tmp/demo_gateway.pid)
    echo "Stopping gateway server (PID: $GATEWAY_PID)..."
    kill $GATEWAY_PID 2>/dev/null || true
    rm -f /tmp/demo_gateway.pid
fi

# Remove temporary files
rm -f /tmp/gateway.log /tmp/demo_gateway_port.txt

echo "✅ Demo stopped successfully!"
echo "💡 Real LLM server on port 8080 is still running"