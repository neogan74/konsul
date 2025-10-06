#!/bin/bash

# Konsul Load Testing Script
# Generates realistic traffic for testing observability stack

set -e

# Configuration
KONSUL_URL="${KONSUL_URL:-http://localhost:8888}"
DURATION="${DURATION:-60}"  # seconds
SERVICE_COUNT="${SERVICE_COUNT:-10}"
KV_KEY_COUNT="${KV_KEY_COUNT:-50}"
CONCURRENT_WORKERS="${CONCURRENT_WORKERS:-5}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Stats
TOTAL_REQUESTS=0
SUCCESSFUL_REQUESTS=0
FAILED_REQUESTS=0

log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

success() {
    echo -e "${GREEN}✓${NC} $1"
}

error() {
    echo -e "${RED}✗${NC} $1"
}

warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

# Check if Konsul is running
check_konsul() {
    log "Checking Konsul availability at ${KONSUL_URL}..."
    if curl -sf "${KONSUL_URL}/health" > /dev/null 2>&1; then
        success "Konsul is running"
        return 0
    else
        error "Konsul is not accessible at ${KONSUL_URL}"
        exit 1
    fi
}

# Register a service
register_service() {
    local service_id=$1
    local service_name=$2
    local port=$((8000 + RANDOM % 1000))
    local ip="10.0.${RANDOM:0:1}.${RANDOM:0:3}"
    local tags=("production" "v1.0" "backend")

    # Randomly select tags
    local selected_tags=""
    for tag in "${tags[@]}"; do
        if [ $((RANDOM % 2)) -eq 0 ]; then
            selected_tags="${selected_tags}\"${tag}\","
        fi
    done
    selected_tags=$(echo "$selected_tags" | sed 's/,$//')

    local payload=$(cat <<EOF
{
    "id": "${service_id}",
    "name": "${service_name}",
    "address": "${ip}",
    "port": ${port},
    "tags": [${selected_tags}],
    "meta": {
        "version": "1.0.0",
        "region": "us-west-2",
        "environment": "production"
    }
}
EOF
)

    if curl -sf -X POST "${KONSUL_URL}/services" \
        -H "Content-Type: application/json" \
        -d "$payload" > /dev/null 2>&1; then
        ((SUCCESSFUL_REQUESTS++))
        return 0
    else
        ((FAILED_REQUESTS++))
        return 1
    fi
    ((TOTAL_REQUESTS++))
}

# Deregister a service
deregister_service() {
    local service_id=$1

    if curl -sf -X DELETE "${KONSUL_URL}/services/${service_id}" > /dev/null 2>&1; then
        ((SUCCESSFUL_REQUESTS++))
        return 0
    else
        ((FAILED_REQUESTS++))
        return 1
    fi
    ((TOTAL_REQUESTS++))
}

# Get services
get_services() {
    if curl -sf "${KONSUL_URL}/services" > /dev/null 2>&1; then
        ((SUCCESSFUL_REQUESTS++))
        return 0
    else
        ((FAILED_REQUESTS++))
        return 1
    fi
    ((TOTAL_REQUESTS++))
}

# Send heartbeat
send_heartbeat() {
    local service_id=$1

    if curl -sf -X POST "${KONSUL_URL}/services/${service_id}/heartbeat" > /dev/null 2>&1; then
        ((SUCCESSFUL_REQUESTS++))
        return 0
    else
        ((FAILED_REQUESTS++))
        return 1
    fi
    ((TOTAL_REQUESTS++))
}

# Set KV pair
set_kv() {
    local key=$1
    local value=$2

    if curl -sf -X PUT "${KONSUL_URL}/kv/${key}" \
        -H "Content-Type: application/json" \
        -d "{\"value\": \"${value}\"}" > /dev/null 2>&1; then
        ((SUCCESSFUL_REQUESTS++))
        return 0
    else
        ((FAILED_REQUESTS++))
        return 1
    fi
    ((TOTAL_REQUESTS++))
}

# Get KV pair
get_kv() {
    local key=$1

    if curl -sf "${KONSUL_URL}/kv/${key}" > /dev/null 2>&1; then
        ((SUCCESSFUL_REQUESTS++))
        return 0
    else
        ((FAILED_REQUESTS++))
        return 1
    fi
    ((TOTAL_REQUESTS++))
}

# Delete KV pair
delete_kv() {
    local key=$1

    if curl -sf -X DELETE "${KONSUL_URL}/kv/${key}" > /dev/null 2>&1; then
        ((SUCCESSFUL_REQUESTS++))
        return 0
    else
        ((FAILED_REQUESTS++))
        return 1
    fi
    ((TOTAL_REQUESTS++))
}

# Health check
health_check() {
    if curl -sf "${KONSUL_URL}/health" > /dev/null 2>&1; then
        ((SUCCESSFUL_REQUESTS++))
        return 0
    else
        ((FAILED_REQUESTS++))
        return 1
    fi
    ((TOTAL_REQUESTS++))
}

# Worker function - simulates realistic application behavior
worker() {
    local worker_id=$1
    local service_names=("web" "api" "cache" "database" "queue" "auth")
    local registered_services=()
    local created_keys=()

    log "Worker ${worker_id} started"

    while true; do
        # Random operation selection (weighted)
        local op=$((RANDOM % 100))

        if [ $op -lt 20 ]; then
            # 20% - Register a new service
            local service_name="${service_names[$((RANDOM % ${#service_names[@]}))]}"
            local service_id="${service_name}-${worker_id}-${RANDOM}"

            if register_service "$service_id" "$service_name"; then
                registered_services+=("$service_id")
            fi

        elif [ $op -lt 25 ] && [ ${#registered_services[@]} -gt 0 ]; then
            # 5% - Deregister a service
            local idx=$((RANDOM % ${#registered_services[@]}))
            local service_id="${registered_services[$idx]}"

            deregister_service "$service_id"
            unset 'registered_services[$idx]'
            registered_services=("${registered_services[@]}")

        elif [ $op -lt 45 ] && [ ${#registered_services[@]} -gt 0 ]; then
            # 20% - Send heartbeat
            local idx=$((RANDOM % ${#registered_services[@]}))
            local service_id="${registered_services[$idx]}"

            send_heartbeat "$service_id"

        elif [ $op -lt 60 ]; then
            # 15% - Set KV
            local key="config/worker${worker_id}/key-${RANDOM}"
            local value="value-$(date +%s)-${RANDOM}"

            if set_kv "$key" "$value"; then
                created_keys+=("$key")
            fi

        elif [ $op -lt 70 ] && [ ${#created_keys[@]} -gt 0 ]; then
            # 10% - Get KV
            local idx=$((RANDOM % ${#created_keys[@]}))
            local key="${created_keys[$idx]}"

            get_kv "$key"

        elif [ $op -lt 75 ] && [ ${#created_keys[@]} -gt 0 ]; then
            # 5% - Delete KV
            local idx=$((RANDOM % ${#created_keys[@]}))
            local key="${created_keys[$idx]}"

            delete_kv "$key"
            unset 'created_keys[$idx]'
            created_keys=("${created_keys[@]}")

        elif [ $op -lt 95 ]; then
            # 20% - List services
            get_services

        else
            # 5% - Health check
            health_check
        fi

        # Random delay between operations (50-500ms)
        sleep 0.$(printf "%03d" $((50 + RANDOM % 450)))
    done
}

# Cleanup function
cleanup() {
    log "Cleaning up..."

    # Kill all background workers
    jobs -p | xargs -r kill 2>/dev/null

    echo ""
    log "Load test completed!"
    log "Statistics:"
    echo "  Total Requests:      ${TOTAL_REQUESTS}"
    echo "  Successful:          ${SUCCESSFUL_REQUESTS}"
    echo "  Failed:              ${FAILED_REQUESTS}"

    if [ ${TOTAL_REQUESTS} -gt 0 ]; then
        local success_rate=$((SUCCESSFUL_REQUESTS * 100 / TOTAL_REQUESTS))
        echo "  Success Rate:        ${success_rate}%"
    fi

    exit 0
}

# Set up signal handlers
trap cleanup SIGINT SIGTERM

# Main
main() {
    echo "╔════════════════════════════════════════════════════════════╗"
    echo "║           Konsul Load Testing Script                      ║"
    echo "╚════════════════════════════════════════════════════════════╝"
    echo ""

    log "Configuration:"
    echo "  Konsul URL:          ${KONSUL_URL}"
    echo "  Duration:            ${DURATION}s"
    echo "  Concurrent Workers:  ${CONCURRENT_WORKERS}"
    echo ""

    check_konsul

    log "Starting ${CONCURRENT_WORKERS} workers..."

    # Start workers in background
    for i in $(seq 1 $CONCURRENT_WORKERS); do
        worker $i &
    done

    success "Workers started (PIDs: $(jobs -p | tr '\n' ' '))"

    # Progress bar
    log "Running load test for ${DURATION} seconds..."
    for i in $(seq 1 $DURATION); do
        printf "\r  Progress: [%-50s] %d%%" \
            $(printf '#%.0s' $(seq 1 $((i * 50 / DURATION)))) \
            $((i * 100 / DURATION))
        sleep 1
    done
    echo ""

    cleanup
}

# Run main function
main "$@"
