#!/bin/bash

# Continuous Traffic Generator for Observability Testing
# Generates steady traffic to populate metrics, logs, and traces

set -e

KONSUL_URL="${KONSUL_URL:-http://localhost:8888}"
INTERVAL="${INTERVAL:-5}"  # seconds between cycles

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Service pools
SERVICES=(
    "web:10.0.1.10:3000:production,frontend"
    "web:10.0.1.11:3000:production,frontend"
    "api:10.0.2.10:8080:production,backend"
    "cache:10.0.3.10:6379:production,redis"
    "db:10.0.4.10:5432:production,postgres"
)

KV_KEYS=(
    "config/app/timeout"
    "config/app/max-conn"
    "feature/new-ui"
    "feature/dark-mode"
    "env/log-level"
)

log() {
    echo -e "${BLUE}[$(date +'%H:%M:%S')]${NC} $1"
}

success() {
    echo -e "${GREEN}✓${NC} $1"
}

# Register services
register_services() {
    log "Registering services..."

    for service_def in "${SERVICES[@]}"; do
        IFS=':' read -r name addr port tags <<< "$service_def"
        local id="${name}-${addr##*.}"

        curl -sf -X POST "${KONSUL_URL}/services" \
            -H "Content-Type: application/json" \
            -d "{
                \"id\": \"${id}\",
                \"name\": \"${name}\",
                \"address\": \"${addr}\",
                \"port\": ${port},
                \"tags\": [\"${tags//,/\",\"}\"]
            }" > /dev/null 2>&1 || true
    done

    success "Services registered"
}

# Send heartbeats
send_heartbeats() {
    for service_def in "${SERVICES[@]}"; do
        IFS=':' read -r name addr port tags <<< "$service_def"
        local id="${name}-${addr##*.}"

        curl -sf -X POST "${KONSUL_URL}/services/${id}/heartbeat" \
            > /dev/null 2>&1 || true
    done
}

# Update KV store
update_kv() {
    local key="${KV_KEYS[$((RANDOM % ${#KV_KEYS[@]}))]}"
    local value="value-$(date +%s)"

    curl -sf -X PUT "${KONSUL_URL}/kv/${key}" \
        -H "Content-Type: application/json" \
        -d "{\"value\": \"${value}\"}" \
        > /dev/null 2>&1 || true
}

# Read KV store
read_kv() {
    local key="${KV_KEYS[$((RANDOM % ${#KV_KEYS[@]}))]}"

    curl -sf "${KONSUL_URL}/kv/${key}" \
        > /dev/null 2>&1 || true
}

# List services
list_services() {
    curl -sf "${KONSUL_URL}/services" \
        > /dev/null 2>&1 || true
}

# Health check
health_check() {
    curl -sf "${KONSUL_URL}/health" \
        > /dev/null 2>&1 || true
}

# Metrics check
metrics_check() {
    curl -sf "${KONSUL_URL}/metrics" \
        > /dev/null 2>&1 || true
}

# Stats
CYCLE=0
STAT_HEARTBEAT=0
STAT_KV=0
STAT_QUERY=0
STAT_HEALTH=0

increment_stat() {
    local key=$1
    case $key in
        heartbeat) ((STAT_HEARTBEAT++)) ;;
        kv) ((STAT_KV++)) ;;
        query) ((STAT_QUERY++)) ;;
        health) ((STAT_HEALTH++)) ;;
    esac
}

show_stats() {
    echo -ne "\r${YELLOW}Cycle: ${CYCLE} | "
    echo -n "Heartbeats: ${STAT_HEARTBEAT} | "
    echo -n "KV Ops: ${STAT_KV} | "
    echo -n "Queries: ${STAT_QUERY} | "
    echo -n "Health: ${STAT_HEALTH}${NC}"
}

cleanup() {
    echo ""
    log "Stopping traffic generator..."
    echo ""
    log "Final Statistics:"
    echo "  Cycles completed: ${CYCLE}"
    echo "  Heartbeats sent: ${STAT_HEARTBEAT}"
    echo "  KV operations: ${STAT_KV}"
    echo "  Service queries: ${STAT_QUERY}"
    echo "  Health checks: ${STAT_HEALTH}"
    exit 0
}

trap cleanup SIGINT SIGTERM

# Main loop
main() {
    echo "╔════════════════════════════════════════════════════════════╗"
    echo "║      Konsul Continuous Traffic Generator                  ║"
    echo "╚════════════════════════════════════════════════════════════╝"
    echo ""
    log "Target: ${KONSUL_URL}"
    log "Interval: ${INTERVAL}s"
    echo ""

    # Initial registration
    register_services
    sleep 2

    log "Starting continuous traffic (Ctrl+C to stop)..."
    echo ""

    while true; do
        ((CYCLE++))

        # Send heartbeats every cycle
        send_heartbeats
        increment_stat "heartbeat"

        # Perform random operations
        for i in {1..5}; do
            case $((RANDOM % 10)) in
                0|1|2)  # 30% - KV write
                    update_kv
                    increment_stat "kv"
                    ;;
                3|4)    # 20% - KV read
                    read_kv
                    increment_stat "kv"
                    ;;
                5|6|7)  # 30% - List services
                    list_services
                    increment_stat "query"
                    ;;
                8)      # 10% - Health check
                    health_check
                    increment_stat "health"
                    ;;
                9)      # 10% - Metrics
                    metrics_check
                    increment_stat "health"
                    ;;
            esac

            sleep 0.2
        done

        show_stats
        sleep "$INTERVAL"
    done
}

main "$@"
