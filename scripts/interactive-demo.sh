#!/bin/bash
set -e

# Check if demo is running
if [ ! -f /tmp/demo_gateway_port.txt ]; then
    echo "❌ Demo not running. Run 'make start-demo' first."
    exit 1
fi

GATEWAY_PORT=$(cat /tmp/demo_gateway_port.txt)

echo "🎯 Interactive JSON Schema Demo"
echo "=============================="
echo ""
echo "Available schema templates:"
echo "1. Person (name, age, occupation)"
echo "2. Product (name, price, category)"
echo "3. Event (title, date, location)"
echo "4. Custom (enter your own schema)"
echo ""

# Function to get user input with prompt
get_input() {
    local prompt="$1"
    local default="$2"
    if [ -n "$default" ]; then
        echo -n "$prompt [$default]: "
    else
        echo -n "$prompt: "
    fi
    read -r input
    if [ -z "$input" ] && [ -n "$default" ]; then
        echo "$default"
    else
        echo "$input"
    fi
}

# Schema templates
person_schema='{
  "type": "object",
  "properties": {
    "name": {"type": "string"},
    "age": {"type": "number"},
    "occupation": {"type": "string"}
  },
  "required": ["name"]
}'

product_schema='{
  "type": "object",
  "properties": {
    "name": {"type": "string"},
    "price": {"type": "number"},
    "category": {"type": "string"},
    "description": {"type": "string"}
  },
  "required": ["name", "price"]
}'

event_schema='{
  "type": "object",
  "properties": {
    "title": {"type": "string"},
    "date": {"type": "string"},
    "location": {"type": "string"},
    "attendees": {"type": "number"}
  },
  "required": ["title", "date"]
}'

# Get schema choice
echo "Choose a schema template (1-4) [1]: "
read -r choice_input
if [ -z "$choice_input" ]; then
    choice="1"
else
    choice="$choice_input"
fi

case $choice in
    1)
        echo "📋 Using Person schema"
        schema="$person_schema"
        default_prompt="Tell me about a software engineer named Alice who is 28 years old"
        ;;
    2)
        echo "📋 Using Product schema"
        schema="$product_schema"
        default_prompt="Describe a wireless headphone product that costs $199"
        ;;
    3)
        echo "📋 Using Event schema"
        schema="$event_schema"
        default_prompt="Create an event for a tech conference next month in San Francisco"
        ;;
    4)
        echo "📋 Custom schema mode"
        echo ""
        echo "Enter your JSON schema (press Enter twice when done):"
        schema=""
        while IFS= read -r line; do
            if [ -z "$line" ]; then
                break
            fi
            schema="$schema$line"
        done
        default_prompt="Generate data matching the schema"
        ;;
    *)
        echo "❌ Invalid choice. Using Person schema."
        schema="$person_schema"
        default_prompt="Tell me about a software engineer named Alice who is 28 years old"
        ;;
esac

# Get user prompt
echo ""
echo "Enter your prompt for the LLM [$default_prompt]: "
read -r user_input
if [ -z "$user_input" ]; then
    user_prompt="$default_prompt"
else
    user_prompt="$user_input"
fi

echo ""
echo "🔄 Sending request..."
echo ""

# Create the request payload
request_payload=$(cat <<EOF
{
  "schema": $schema,
  "messages": [
    {
      "role": "user",
      "content": "$user_prompt"
    }
  ]
}
EOF
)

# Display the request
echo "📤 Request:"
echo "----------"
echo "$request_payload" | jq
echo ""

# Send the request
echo "📥 Response:"
echo "-----------"
response=$(curl -s -X POST http://localhost:$GATEWAY_PORT/v1/validated-query \
    -H "Content-Type: application/json" \
    -d "$request_payload")

# Check if response is valid JSON and display it
if echo "$response" | jq empty 2>/dev/null; then
    echo "$response" | jq
    
    # Check if it's an error response
    if echo "$response" | jq -e '.error' >/dev/null 2>&1; then
        echo ""
        echo "❌ Validation failed or error occurred"
    else
        echo ""
        echo "✅ Response validates successfully against the schema!"
    fi
else
    echo "❌ Invalid response received:"
    echo "$response"
fi

echo ""
echo "🔄 Run 'make demo-interactive' again to try another example"
echo "🛑 Run 'make stop-demo' to stop the demo environment"