#!/usr/bin/env python3
"""
Qwen2 Inference Script - Collects streaming responses from Qwen2 model via Ollama
"""
import json
import requests
import sys
import time

def stream_qwen2_response(prompt, model="qwen2:1.5b", host="http://localhost:11434"):
    """
    Stream response from Qwen2 model via Ollama API and collect complete text
    """
    url = f"{host}/api/generate"
    
    payload = {
        "model": model,
        "prompt": prompt,
        "stream": True
    }
    
    print(f"🧠 Qwen2 generating response for: '{prompt}'\n")
    print("📝 Response: ", end="", flush=True)
    
    complete_response = ""
    
    try:
        with requests.post(url, json=payload, stream=True) as response:
            response.raise_for_status()
            
            for line in response.iter_lines():
                if line:
                    try:
                        data = json.loads(line.decode('utf-8'))
                        
                        if 'response' in data:
                            token = data['response']
                            complete_response += token
                            
                            # Print each token as it comes
                            print(token, end="", flush=True)
                            
                            # Small delay for visual effect (optional)
                            time.sleep(0.01)
                        
                        # Check if generation is complete
                        if data.get('done', False):
                            break
                            
                    except json.JSONDecodeError:
                        continue
        
        print(f"\n\n✅ Complete Qwen2 response:\n{complete_response}")
        return complete_response
        
    except requests.exceptions.RequestException as e:
        print(f"\n❌ Error connecting to Qwen2 via Ollama: {e}")
        print("💡 Make sure port forwarding is active:")
        print("   kubectl port-forward service/s-kdss-82mqg-0 11434:11434")
        return None
    except KeyboardInterrupt:
        print(f"\n\n⏸️  Interrupted. Partial Qwen2 response:\n{complete_response}")
        return complete_response

def chat_qwen2(messages, model="qwen2:1.5b", host="http://localhost:11434"):
    """
    Chat completion with Qwen2 model via Ollama API
    """
    url = f"{host}/api/chat"
    
    payload = {
        "model": model,
        "messages": messages,
        "stream": True
    }
    
    print(f"💬 Chat with Qwen2 ({model}):\n")
    print("🧠 Qwen2: ", end="", flush=True)
    
    complete_response = ""
    
    try:
        with requests.post(url, json=payload, stream=True) as response:
            response.raise_for_status()
            
            for line in response.iter_lines():
                if line:
                    try:
                        data = json.loads(line.decode('utf-8'))
                        
                        if 'message' in data and 'content' in data['message']:
                            token = data['message']['content']
                            complete_response += token
                            
                            # Print each token as it comes
                            print(token, end="", flush=True)
                            
                            # Small delay for visual effect
                            time.sleep(0.01)
                        
                        if data.get('done', False):
                            break
                            
                    except json.JSONDecodeError:
                        continue
        
        print(f"\n\n✅ Complete Qwen2 response:\n{complete_response}")
        return complete_response
        
    except requests.exceptions.RequestException as e:
        print(f"\n❌ Error connecting to Qwen2 via Ollama: {e}")
        print("💡 Make sure port forwarding is active:")
        print("   kubectl port-forward service/s-kdss-82mqg-0 11434:11434")
        return None
    except KeyboardInterrupt:
        print(f"\n\n⏸️  Interrupted. Partial Qwen2 response:\n{complete_response}")
        return complete_response

def check_qwen2_connection(host="http://localhost:11434"):
    """
    Check if Ollama is running and Qwen2 model is available
    """
    try:
        # Check Ollama version
        response = requests.get(f"{host}/api/version", timeout=5)
        version = response.json().get('version', 'unknown')
        print(f"✅ Connected to Ollama version: {version}")
        
        # Check available models
        models_response = requests.get(f"{host}/api/tags", timeout=5)
        models = models_response.json().get('models', [])
        qwen2_models = [m['name'] for m in models if 'qwen2' in m['name'].lower()]
        
        if qwen2_models:
            print(f"🧠 Available Qwen2 models: {', '.join(qwen2_models)}\n")
        else:
            print("⚠️  No Qwen2 models found. Run: kubectl exec -it <pod> -- ollama pull qwen2:1.5b\n")
        
        return True
        
    except requests.exceptions.RequestException:
        print("❌ Cannot connect to Ollama. Make sure:")
        print("   1. Qwen2 cluster is running: kubectl get pods")
        print("   2. Port forwarding is active: kubectl port-forward service/s-kdss-82mqg-0 11434:11434")
        return False

def main():
    if len(sys.argv) < 2:
        print("🧠 Qwen2 Inference Helper")
        print("=" * 25)
        print("Usage:")
        print("  python qwen2_inference.py 'Your prompt here'")
        print("  python qwen2_inference.py chat 'Your message here'")
        print("  python qwen2_inference.py chat 'Your message' qwen2:7b")
        print("\nExamples:")
        print("  python qwen2_inference.py 'Explain quantum computing in simple terms'")
        print("  python qwen2_inference.py chat 'What is machine learning?'")
        print("  python qwen2_inference.py chat 'Write a Python function to sort a list' qwen2:7b")
        print("\n💡 Qwen2 excels at reasoning, mathematics, and coding tasks!")
        return
    
    # Check connection
    if not check_qwen2_connection():
        return
    
    mode = sys.argv[1]
    
    if mode.lower() == "chat" and len(sys.argv) >= 3:
        # Chat mode
        message = sys.argv[2]
        model = sys.argv[3] if len(sys.argv) > 3 else "qwen2:1.5b"
        messages = [{"role": "user", "content": message}]
        chat_qwen2(messages, model)
    else:
        # Generate mode
        prompt = " ".join(sys.argv[1:])
        model = "qwen2:1.5b"  # Default model for generation
        stream_qwen2_response(prompt, model)

if __name__ == "__main__":
    main()