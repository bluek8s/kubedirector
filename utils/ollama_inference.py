#!/usr/bin/env python3
"""
Ollama Inference Script - Collects streaming responses and displays complete text
"""
import json
import requests
import sys
import time

def stream_ollama_response(prompt, model="tinyllama", host="http://localhost:11434"):
    """
    Stream response from Ollama API and collect complete text
    """
    url = f"{host}/api/generate"
    
    payload = {
        "model": model,
        "prompt": prompt,
        "stream": True
    }
    
    print(f"🤖 Generating response for: '{prompt}'\n")
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
        
        print(f"\n\n✅ Complete response:\n{complete_response}")
        return complete_response
        
    except requests.exceptions.RequestException as e:
        print(f"\n❌ Error connecting to Ollama: {e}")
        return None
    except KeyboardInterrupt:
        print(f"\n\n⏸️  Interrupted. Partial response:\n{complete_response}")
        return complete_response

def chat_ollama(messages, model="tinyllama", host="http://localhost:11434"):
    """
    Chat completion with Ollama API
    """
    url = f"{host}/api/chat"
    
    payload = {
        "model": model,
        "messages": messages,
        "stream": True
    }
    
    print(f"💬 Chat with {model}:\n")
    print("🤖 Assistant: ", end="", flush=True)
    
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
        
        print(f"\n\n✅ Complete response:\n{complete_response}")
        return complete_response
        
    except requests.exceptions.RequestException as e:
        print(f"\n❌ Error connecting to Ollama: {e}")
        return None
    except KeyboardInterrupt:
        print(f"\n\n⏸️  Interrupted. Partial response:\n{complete_response}")
        return complete_response

def main():
    if len(sys.argv) < 2:
        print("Usage:")
        print("  python ollama_inference.py 'Your prompt here'")
        print("  python ollama_inference.py chat 'Your message here'")
        print("\nExamples:")
        print("  python ollama_inference.py 'The future of AI is'")
        print("  python ollama_inference.py chat 'Why is the sky blue?'")
        return
    
    # Check if Ollama is running
    try:
        response = requests.get("http://localhost:11434/api/version", timeout=5)
        print(f"✅ Connected to Ollama version: {response.json().get('version', 'unknown')}\n")
    except:
        print("❌ Cannot connect to Ollama. Make sure port forwarding is active:")
        print("   kubectl port-forward service/s-kdss-845dg-0 11434:11434")
        return
    
    mode = sys.argv[1]
    
    if mode.lower() == "chat" and len(sys.argv) >= 3:
        # Chat mode
        message = " ".join(sys.argv[2:])
        messages = [{"role": "user", "content": message}]
        chat_ollama(messages)
    else:
        # Generate mode
        prompt = " ".join(sys.argv[1:])
        stream_ollama_response(prompt)

if __name__ == "__main__":
    main()