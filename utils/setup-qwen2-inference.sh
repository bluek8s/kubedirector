#!/bin/bash

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

CLUSTER_NAME="qwen2-cluster"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_blue() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

check_docker() {
    log_info "Checking Docker installation and status..."
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed. Please install Docker Desktop from https://docker.com/products/docker-desktop"
        exit 1
    fi
    
    if ! docker info &> /dev/null; then
        log_error "Docker is not running. Please start Docker Desktop and try again."
        exit 1
    fi
    
    log_info "Docker is installed and running ✓"
}

install_k3d() {
    log_info "Checking k3d installation..."
    
    if command -v k3d &> /dev/null; then
        log_info "k3d is already installed ✓"
        return 0
    fi
    
    log_info "Installing k3d..."
    if command -v brew &> /dev/null; then
        brew install k3d
    else
        curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash
    fi
    
    log_info "k3d installed successfully ✓"
}

create_k3d_cluster() {
    log_info "Creating k3d cluster '$CLUSTER_NAME'..."
    
    if k3d cluster list | grep -q "$CLUSTER_NAME"; then
        log_warn "Cluster '$CLUSTER_NAME' already exists. Deleting and recreating..."
        k3d cluster delete "$CLUSTER_NAME"
    fi
    
    k3d cluster create "$CLUSTER_NAME" \
        --port "8080:80@loadbalancer" \
        --port "8443:443@loadbalancer" \
        --k3s-arg "--disable=traefik@server:0" \
        --wait
    
    log_info "k3d cluster '$CLUSTER_NAME' created successfully ✓"
    
    # Verify cluster is ready
    kubectl cluster-info
}

install_kubedirector() {
    log_info "Installing KubeDirector..."
    
    # Apply KubeDirector CRDs and operator
    kubectl apply -f "$PROJECT_ROOT/deploy/kubedirector"
    
    # Wait for kubedirector to be ready
    log_info "Waiting for KubeDirector operator to be ready..."
    kubectl wait --for=condition=available --timeout=300s deployment/kubedirector -n default
    
    log_info "KubeDirector installed successfully ✓"
}

deploy_qwen2_app() {
    log_info "Deploying Qwen2 application definition..."
    
    kubectl apply -f "$PROJECT_ROOT/deploy/example_catalog/cr-app-qwen2.yaml"
    
    # Wait for app to be registered
    sleep 5
    kubectl get kdapp qwen2-server
    
    log_info "Qwen2 app deployed successfully ✓"
}

create_qwen2_cluster() {
    log_info "Creating Qwen2 inference cluster..."
    
    kubectl apply -f "$PROJECT_ROOT/deploy/example_clusters/cr-cluster-qwen2-test.yaml"
    
    log_info "Waiting for Qwen2 cluster to be ready..."
    
    # Wait for cluster to be ready (this may take several minutes)
    timeout=900  # 15 minutes (Qwen2 is larger than TinyLlama)
    elapsed=0
    interval=15
    
    while [ $elapsed -lt $timeout ]; do
        status=$(kubectl get kdcluster qwen2-test-cluster -o jsonpath='{.status.state}' 2>/dev/null || echo "NotFound")
        
        if [ "$status" = "configured" ]; then
            log_info "Qwen2 cluster is ready ✓"
            break
        elif [ "$status" = "NotFound" ]; then
            log_info "Waiting for cluster to be created..."
        else
            log_info "Cluster status: $status (waiting...)"
        fi
        
        sleep $interval
        elapsed=$((elapsed + interval))
    done
    
    if [ $elapsed -ge $timeout ]; then
        log_error "Timeout waiting for cluster to be ready"
        exit 1
    fi
}

setup_qwen2_model() {
    log_info "Setting up Qwen2 model..."
    
    # Get the pod name
    POD_NAME=$(kubectl get pods -l kubedirector.hpe.com/kdcluster=qwen2-test-cluster -o jsonpath='{.items[0].metadata.name}')
    
    if [ -z "$POD_NAME" ]; then
        log_error "Could not find Qwen2 pod"
        exit 1
    fi
    
    log_info "Found pod: $POD_NAME"
    
    # Wait for pod to be ready
    kubectl wait --for=condition=ready pod/$POD_NAME --timeout=300s
    
    # Pull Qwen2 model (this will take longer than TinyLlama)
    log_warn "Pulling Qwen2 model (this will take 10-15 minutes due to model size ~7GB)..."
    kubectl exec -it $POD_NAME -- ollama pull qwen2:7b
    
    log_info "Qwen2 model ready ✓"
}

get_service_info() {
    log_info "Getting service information..."
    
    # Get service details
    SERVICE_INFO=$(kubectl get svc -l kubedirector.hpe.com/kdcluster=qwen2-test-cluster)
    echo "$SERVICE_INFO"
    
    # Get the NodePort for the service
    NODE_PORT=$(kubectl get svc -l kubedirector.hpe.com/kdcluster=qwen2-test-cluster -o jsonpath='{.items[0].spec.ports[0].nodePort}')
    
    if [ ! -z "$NODE_PORT" ]; then
        log_info "Qwen2 API is available at: http://localhost:$NODE_PORT"
        echo
        echo "You can test the inference with:"
        echo "curl http://localhost:$NODE_PORT/api/generate -d '{\"model\":\"qwen2:7b\",\"prompt\":\"Hello, how are you?\"}'"
        echo
        echo "Or use the chat API:"
        echo "curl http://localhost:$NODE_PORT/api/chat -d '{\"model\":\"qwen2:7b\",\"messages\":[{\"role\":\"user\",\"content\":\"What is machine learning?\"}]}'"
    fi
}

test_qwen2_inference() {
    log_blue "Testing Qwen2 inference..."
    
    # Get the NodePort
    NODE_PORT=$(kubectl get svc -l kubedirector.hpe.com/kdcluster=qwen2-test-cluster -o jsonpath='{.items[0].spec.ports[0].nodePort}')
    
    if [ -z "$NODE_PORT" ]; then
        log_error "Could not get service port"
        return 1
    fi
    
    local HOST="http://localhost:$NODE_PORT"
    
    # Wait a moment for service to be fully ready
    sleep 5
    
    echo
    log_blue "🧪 Running test inference..."
    echo "Prompt: 'Explain quantum computing in simple terms'"
    echo
    echo -n "🤖 Qwen2 Response: "
    
    # Test with a simple prompt
    curl -s -X POST "$HOST/api/generate" \
        -H "Content-Type: application/json" \
        -d '{"model":"qwen2:7b","prompt":"Explain quantum computing in simple terms","stream":false}' | \
    if command -v jq >/dev/null 2>&1; then
        jq -r '.response'
    else
        sed -n 's/.*"response":"\([^"]*\)".*/\1/p'
    fi
    
    echo
    log_info "✅ Qwen2 inference test completed!"
}

main() {
    echo "=================================================="
    echo "     KubeDirector Qwen2 Setup Script"
    echo "=================================================="
    echo
    
    # Parse command line arguments
    SKIP_SETUP=false
    TEST_ONLY=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --test-only)
                TEST_ONLY=true
                shift
                ;;
            --skip-setup)
                SKIP_SETUP=true
                shift
                ;;
            -h|--help)
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  --test-only     Only run inference test (assumes cluster is already running)"
                echo "  --skip-setup    Skip initial setup, only deploy and test Qwen2"
                echo "  -h, --help      Show this help message"
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                echo "Use --help for usage information"
                exit 1
                ;;
        esac
    done
    
    if [ "$TEST_ONLY" = true ]; then
        test_qwen2_inference
        exit 0
    fi
    
    if [ "$SKIP_SETUP" = false ]; then
        check_docker
        install_k3d
        create_k3d_cluster
        install_kubedirector
    fi
    
    deploy_qwen2_app
    create_qwen2_cluster
    setup_qwen2_model
    get_service_info
    test_qwen2_inference
    
    echo
    echo "=================================================="
    log_info "🎉 Qwen2 setup completed successfully!"
    log_info "Your Qwen2 inference cluster is ready for advanced text generation!"
    echo "=================================================="
    echo
    echo "💡 Pro tips:"
    echo "  • Qwen2 excels at reasoning, math, and coding tasks"
    echo "  • Try asking complex questions or coding problems"
    echo "  • Use '$0 --test-only' to run quick tests"
    echo "  • Monitor resources: kubectl top pods"
}

main "$@"