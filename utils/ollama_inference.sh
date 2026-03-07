#!/bin/bash
# Simple bash script to collect Ollama streaming responses

# Configuration
HOST="http://localhost:11434"
MODEL="tinyllama"

# Function to stream and collect response
stream_ollama() {
    local prompt="$1"
    local model="${2:-$MODEL}"
    
    echo "🤖 Generating response for: '$prompt'"
    echo ""
    echo -n "📝 Response: "
    
    # Create temp file for collecting response
    local temp_file=$(mktemp)
    local complete_response=""
    
    # Stream the response and collect tokens
    curl -s -X POST "$HOST/api/generate" \
        -H "Content-Type: application/json" \
        -d "{\"model\":\"$model\",\"prompt\":\"$prompt\",\"stream\":true}" | \
    while IFS= read -r line; do
        if [ -n "$line" ]; then
            # Extract response field using jq (or sed if jq not available)
            if command -v jq >/dev/null 2>&1; then
                token=$(echo "$line" | jq -r '.response // empty' 2>/dev/null)
                done_status=$(echo "$line" | jq -r '.done // false' 2>/dev/null)
            else
                # Fallback using sed (less reliable but works without jq)
                token=$(echo "$line" | sed -n 's/.*"response":"\([^"]*\)".*/\1/p')
                done_status=$(echo "$line" | grep -o '"done":true' >/dev/null && echo "true" || echo "false")
            fi
            
            if [ -n "$token" ] && [ "$token" != "null" ]; then
                echo -n "$token"
                echo -n "$token" >> "$temp_file"
            fi
            
            # Check if generation is complete
            if [ "$done_status" = "true" ]; then
                break
            fi
        fi
    done
    
    echo ""
    echo ""
    echo "✅ Complete response:"
    cat "$temp_file"
    echo ""
    
    rm -f "$temp_file"
}

# Function for chat completion
chat_ollama() {
    local message="$1"
    local model="${2:-$MODEL}"
    
    echo "💬 Chat with $model:"
    echo ""
    echo -n "🤖 Assistant: "
    
    local temp_file=$(mktemp)
    
    # Create messages JSON
    local messages_json="{\"model\":\"$model\",\"messages\":[{\"role\":\"user\",\"content\":\"$message\"}],\"stream\":true}"
    
    curl -s -X POST "$HOST/api/chat" \
        -H "Content-Type: application/json" \
        -d "$messages_json" | \
    while IFS= read -r line; do
        if [ -n "$line" ]; then
            if command -v jq >/dev/null 2>&1; then
                token=$(echo "$line" | jq -r '.message.content // empty' 2>/dev/null)
                done_status=$(echo "$line" | jq -r '.done // false' 2>/dev/null)
            else
                # Fallback for systems without jq
                token=$(echo "$line" | sed -n 's/.*"content":"\([^"]*\)".*/\1/p')
                done_status=$(echo "$line" | grep -o '"done":true' >/dev/null && echo "true" || echo "false")
            fi
            
            if [ -n "$token" ] && [ "$token" != "null" ]; then
                echo -n "$token"
                echo -n "$token" >> "$temp_file"
            fi
            
            if [ "$done_status" = "true" ]; then
                break
            fi
        fi
    done
    
    echo ""
    echo ""
    echo "✅ Complete response:"
    cat "$temp_file"
    echo ""
    
    rm -f "$temp_file"
}

# Check if Ollama is running
check_ollama() {
    if ! curl -s "$HOST/api/version" >/dev/null 2>&1; then
        echo "❌ Cannot connect to Ollama at $HOST"
        echo "Make sure port forwarding is active:"
        echo "   kubectl port-forward service/s-kdss-845dg-0 11434:11434"
        exit 1
    fi
    
    local version=$(curl -s "$HOST/api/version" | grep -o '"version":"[^"]*"' | cut -d'"' -f4)
    echo "✅ Connected to Ollama version: ${version:-unknown}"
    echo ""
}

# Main script
main() {
    if [ $# -eq 0 ]; then
        echo "Usage:"
        echo "  $0 'Your prompt here'"
        echo "  $0 chat 'Your message here'"
        echo "  $0 chat 'Your message' model_name"
        echo ""
        echo "Examples:"
        echo "  $0 'The future of AI is'"
        echo "  $0 chat 'Why is the sky blue?'"
        echo "  $0 chat 'Hello!' gemma2:2b"
        exit 1
    fi
    
    check_ollama
    
    if [ "$1" = "chat" ]; then
        if [ $# -lt 2 ]; then
            echo "Error: Chat mode requires a message"
            echo "Usage: $0 chat 'Your message here'"
            exit 1
        fi
        chat_ollama "$2" "$3"
    else
        # Generate mode - join all arguments as prompt
        local prompt="$*"
        stream_ollama "$prompt"
    fi
}

main "$@"