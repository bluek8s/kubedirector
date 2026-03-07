#!/bin/bash

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

CLUSTER_NAME="llm-cluster"
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

check_docker() {
    log_info "Checking Docker installation and status..."
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed. Please install Docker Desktop for Mac from https://docker.com/products/docker-desktop"
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
    kubectl wait --for=condition=available --timeout=300s deployment/kubedirector -n kubedirector
    
    log_info "KubeDirector installed successfully ✓"
}

deploy_tinyllama_app() {
    log_info "Deploying TinyLlama application definition..."
    
    kubectl apply -f "$PROJECT_ROOT/deploy/example_catalog/cr-app-tinyllama.yaml"
    
    # Wait for app to be registered
    sleep 5
    kubectl get kdapp vllm-server
    
    log_info "TinyLlama app deployed successfully ✓"
}

create_tinyllama_cluster() {
    log_info "Creating TinyLlama inference cluster..."
    
    kubectl apply -f "$PROJECT_ROOT/deploy/example_clusters/cr-cluster-tinyllama-test.yaml"
    
    log_info "Waiting for TinyLlama cluster to be ready..."
    
    # Wait for cluster to be ready (this may take several minutes)
    timeout=600  # 10 minutes
    elapsed=0
    interval=10
    
    while [ $elapsed -lt $timeout ]; do
        status=$(kubectl get kdcluster ollama-test-cluster -o jsonpath='{.status.state}' 2>/dev/null || echo "NotFound")
        
        if [ "$status" = "configured" ]; then
            log_info "TinyLlama cluster is ready ✓"
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

setup_tinyllama_model() {
    log_info "Setting up TinyLlama model..."
    
    # Get the pod name
    POD_NAME=$(kubectl get pods -l kubedirector.hpe.com/kdcluster=ollama-test-cluster -o jsonpath='{.items[0].metadata.name}')
    
    if [ -z "$POD_NAME" ]; then
        log_error "Could not find TinyLlama pod"
        exit 1
    fi
    
    log_info "Found pod: $POD_NAME"
    
    # Wait for pod to be ready
    kubectl wait --for=condition=ready pod/$POD_NAME --timeout=300s
    
    # Pull TinyLlama model
    log_info "Pulling TinyLlama model (this may take a few minutes)..."
    kubectl exec -it $POD_NAME -- ollama pull tinyllama
    
    log_info "TinyLlama model ready ✓"
}

get_service_info() {
    log_info "Getting service information..."
    
    # Get service details
    SERVICE_INFO=$(kubectl get svc -l kubedirector.hpe.com/kdcluster=ollama-test-cluster)
    echo "$SERVICE_INFO"
    
    # Get the NodePort for the service
    NODE_PORT=$(kubectl get svc -l kubedirector.hpe.com/kdcluster=ollama-test-cluster -o jsonpath='{.items[0].spec.ports[0].nodePort}')
    
    if [ ! -z "$NODE_PORT" ]; then
        log_info "TinyLlama API is available at: http://localhost:$NODE_PORT"
        echo
        echo "You can test the inference with:"
        echo "curl http://localhost:$NODE_PORT/api/generate -d '{\"model\":\"tinyllama\",\"prompt\":\"Hello world\"}'"
    fi
}

main() {
    echo "================================================"
    echo "   KubeDirector TinyLlama Setup Script for Mac  "
    echo "================================================"
    echo
    
    check_docker
    install_k3d
    create_k3d_cluster
    install_kubedirector
    deploy_tinyllama_app
    create_tinyllama_cluster
    setup_tinyllama_model
    get_service_info
    
    echo
    echo "================================================"
    log_info "🎉 Setup completed successfully!"
    log_info "Your TinyLlama inference cluster is ready to accept text generation calls!"
    echo "================================================"
}

main "$@"