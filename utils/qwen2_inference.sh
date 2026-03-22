#!/bin/bash
# Qwen2 Inference Script - Collects streaming responses from Qwen2 model via Ollama

# Configuration
HOST="http://localhost:11434"
MODEL="qwen2:1.5b"

# Function to stream and collect response from Qwen2
stream_qwen2() {
    local prompt="$1"
    local model="${2:-$MODEL}"
    
    echo "🧠 Qwen2 generating response for: '$prompt'"
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
    echo "✅ Complete Qwen2 response:"
    cat "$temp_file"
    echo ""
    
    rm -f "$temp_file"
}

# Function for Qwen2 chat completion
chat_qwen2() {
    local message="$1"
    local model="${2:-$MODEL}"
    
    echo "💬 Chat with Qwen2 ($model):"
    echo ""
    echo -n "🧠 Qwen2: "
    
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
    echo "✅ Complete Qwen2 response:"
    cat "$temp_file"
    echo ""
    
    rm -f "$temp_file"
}

# Check if Ollama is running and Qwen2 model is available
check_qwen2() {
    if ! curl -s "$HOST/api/version" >/dev/null 2>&1; then
        echo "❌ Cannot connect to Ollama at $HOST"
        echo "Make sure:"
        echo "   1. Qwen2 cluster is running: kubectl get pods"
        echo "   2. Port forwarding is active: kubectl port-forward service/s-kdss-82mqg-0 11434:11434"
        exit 1
    fi
    
    local version=$(curl -s "$HOST/api/version" | grep -o '"version":"[^"]*"' | cut -d'"' -f4)
    echo "✅ Connected to Ollama version: ${version:-unknown}"
    
    # Check for Qwen2 models
    local models=$(curl -s "$HOST/api/tags" 2>/dev/null)
    if echo "$models" | grep -q "qwen2"; then
        local qwen2_models=$(echo "$models" | grep -o '"name":"qwen2[^"]*"' | cut -d'"' -f4 | tr '\n' ', ' | sed 's/,$//')
        echo "🧠 Available Qwen2 models: $qwen2_models"
    else
        echo "⚠️  No Qwen2 models found. Run: kubectl exec -it <pod> -- ollama pull qwen2:1.5b"
    fi
    echo ""
}

# Main script
main() {
    if [ $# -eq 0 ]; then
        echo "🧠 Qwen2 Inference Helper"
        echo "========================="
        echo "Usage:"
        echo "  $0 'Your prompt here'"
        echo "  $0 chat 'Your message here'"
        echo "  $0 chat 'Your message' qwen2:7b"
        echo ""
        echo "Examples:"
        echo "  $0 'Explain quantum computing in simple terms'"
        echo "  $0 chat 'What is machine learning?'"
        echo "  $0 chat 'Write a Python function to sort a list' qwen2:7b"
        echo ""
        echo "💡 Qwen2 excels at reasoning, mathematics, and coding tasks!"
        exit 1
    fi
    
    check_qwen2
    
    if [ "$1" = "chat" ]; then
        if [ $# -lt 2 ]; then
            echo "Error: Chat mode requires a message"
            echo "Usage: $0 chat 'Your message here'"
            exit 1
        fi
        chat_qwen2 "$2" "$3"
    else
        # Generate mode - join all arguments as prompt
        local prompt="$*"
        stream_qwen2 "$prompt" "$MODEL"
    fi
}

main "$@"