#!/bin/bash

# Konsul Demo Scenario Script
# Demonstrates realistic microservices deployment using konsulctl

set -e

# Configuration
KONSUL_URL="${KONSUL_URL:-http://localhost:8888}"
KONSULCTL="${KONSULCTL:-../bin/konsulctl}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}[$(date +'%H:%M:%S')]${NC} $1"
}

success() {
    echo -e "${GREEN}✓${NC} $1"
}

error() {
    echo -e "${RED}✗${NC} $1"
}

step() {
    echo -e "${CYAN}▶${NC} $1"
}

pause() {
    local delay=${1:-2}
    sleep $delay
}

banner() {
    echo ""
    echo -e "${MAGENTA}╔════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${MAGENTA}║${NC} $1"
    echo -e "${MAGENTA}╚════════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

# Check prerequisites
check_prerequisites() {
    banner "Checking Prerequisites"

    step "Checking konsulctl..."
    if [ ! -f "$KONSULCTL" ]; then
        error "konsulctl not found at $KONSULCTL"
        log "Build it with: make build-cli"
        exit 1
    fi
    success "konsulctl found"

    step "Checking Konsul availability..."
    if curl -sf "${KONSUL_URL}/health" > /dev/null 2>&1; then
        success "Konsul is running at ${KONSUL_URL}"
    else
        error "Konsul is not accessible at ${KONSUL_URL}"
        exit 1
    fi

    pause
}

# Scenario 1: E-commerce Platform Deployment
scenario_ecommerce() {
    banner "Scenario 1: E-commerce Platform Deployment"

    log "Deploying e-commerce microservices..."
    pause

    step "Registering frontend services..."
    $KONSULCTL service register \
        --id web-frontend-1 \
        --name web-frontend \
        --address 10.0.1.10 \
        --port 3000 \
        --tags production,frontend,react

    $KONSULCTL service register \
        --id web-frontend-2 \
        --name web-frontend \
        --address 10.0.1.11 \
        --port 3000 \
        --tags production,frontend,react
    success "Frontend services registered (2 instances)"
    pause 1

    step "Registering API gateway..."
    $KONSULCTL service register \
        --id api-gateway-1 \
        --name api-gateway \
        --address 10.0.2.10 \
        --port 8080 \
        --tags production,api,gateway
    success "API gateway registered"
    pause 1

    step "Registering backend services..."

    # Product service
    $KONSULCTL service register \
        --id product-service-1 \
        --name product-service \
        --address 10.0.3.10 \
        --port 9001 \
        --tags production,backend,java

    $KONSULCTL service register \
        --id product-service-2 \
        --name product-service \
        --address 10.0.3.11 \
        --port 9001 \
        --tags production,backend,java

    # Order service
    $KONSULCTL service register \
        --id order-service-1 \
        --name order-service \
        --address 10.0.3.20 \
        --port 9002 \
        --tags production,backend,go

    # Payment service
    $KONSULCTL service register \
        --id payment-service-1 \
        --name payment-service \
        --address 10.0.3.30 \
        --port 9003 \
        --tags production,backend,nodejs,pci-compliant

    success "Backend services registered (4 services)"
    pause 1

    step "Registering data layer..."

    # PostgreSQL
    $KONSULCTL service register \
        --id postgres-primary \
        --name postgres \
        --address 10.0.4.10 \
        --port 5432 \
        --tags production,database,primary

    $KONSULCTL service register \
        --id postgres-replica-1 \
        --name postgres \
        --address 10.0.4.11 \
        --port 5432 \
        --tags production,database,replica

    # Redis
    $KONSULCTL service register \
        --id redis-cache-1 \
        --name redis \
        --address 10.0.4.20 \
        --port 6379 \
        --tags production,cache,redis

    success "Data layer registered (3 instances)"
    pause 1

    step "Storing application configuration..."

    $KONSULCTL kv set config/ecommerce/database/host "postgres-primary.service.consul"
    $KONSULCTL kv set config/ecommerce/database/port "5432"
    $KONSULCTL kv set config/ecommerce/redis/host "redis-cache-1.service.consul"
    $KONSULCTL kv set config/ecommerce/api/rate-limit "100"
    $KONSULCTL kv set config/ecommerce/feature-flags/new-checkout "true"

    success "Configuration stored (5 keys)"
    pause 1

    step "Current deployment status..."
    $KONSULCTL service list
    pause 2

    log "E-commerce platform deployed successfully!"
    pause 2
}

# Scenario 2: Service Scaling and Health Checks
scenario_scaling() {
    banner "Scenario 2: Service Scaling Demo"

    log "Simulating traffic increase and auto-scaling..."
    pause

    step "Scaling product-service (adding 2 more instances)..."
    $KONSULCTL service register \
        product-service \
        10.0.3.12 \
        9001

    $KONSULCTL service register \
        product-service \
        10.0.3.13 \
        9001

    success "Product service scaled to 4 instances"
    pause 1

    step "Sending heartbeats to keep services healthy..."
    for i in 1 2 3 4; do
        $KONSULCTL service heartbeat product-service-$i 2>/dev/null || true
    done
    success "Heartbeats sent"
    pause 1

    step "Simulating a failing instance (product-service-3)..."
    log "Instance stopped sending heartbeats (will expire after TTL)..."
    pause 2

    step "Adding canary deployment for order-service..."
    $KONSULCTL service register \
        order-service \
        10.0.3.21 \
        9002

    success "Canary instance deployed"
    pause 2

    log "Scaling demonstration complete!"
    pause 2
}

# Scenario 3: Configuration Management
scenario_config() {
    banner "Scenario 3: Dynamic Configuration Management"

    log "Managing application configuration via KV store..."
    pause

    step "Setting feature flags..."
    $KONSULCTL kv set feature-flags/new-ui-redesign "false"
    $KONSULCTL kv set feature-flags/enable-recommendations "true"
    $KONSULCTL kv set feature-flags/dark-mode "true"
    success "Feature flags configured"
    pause 1

    step "Storing service discovery config..."
    $KONSULCTL kv set discovery/timeout-ms "5000"
    $KONSULCTL kv set discovery/retry-attempts "3"
    $KONSULCTL kv set discovery/health-check-interval "30s"
    success "Discovery settings stored"
    pause 1

    step "Adding environment-specific settings..."
    $KONSULCTL kv set env/production/log-level "info"
    $KONSULCTL kv set env/production/debug-mode "false"
    $KONSULCTL kv set env/production/max-connections "1000"
    success "Environment config set"
    pause 1

    step "Listing all configuration keys..."
    $KONSULCTL kv list
    pause 2

    step "Reading a specific config..."
    $KONSULCTL kv get feature-flags/enable-recommendations
    pause 2

    log "Configuration management complete!"
    pause 2
}

# Scenario 4: Service Migration
scenario_migration() {
    banner "Scenario 4: Blue-Green Deployment"

    log "Performing blue-green deployment for payment service..."
    pause

    step "Current state: Blue (v1.0) in production..."
    log "Blue: payment-service-1 (v1.0)"
    pause 1

    step "Deploying Green environment (v2.0)..."
    $KONSULCTL service register \
        --id payment-service-2-green \
        --name payment-service-v2 \
        --address 10.0.3.31 \
        --port 9003 \
        --tags staging,backend,nodejs,green,v2.0

    success "Green environment deployed"
    pause 1

    step "Running smoke tests on green..."
    log "Tests passed ✓"
    pause 1

    step "Switching traffic to green..."

    # Update DNS/config to point to v2
    $KONSULCTL kv set config/ecommerce/payment-service/active-version "v2.0"

    # Register green as production
    $KONSULCTL service register \
        --id payment-service-2 \
        --name payment-service \
        --address 10.0.3.31 \
        --port 9003 \
        --tags production,backend,nodejs,v2.0

    success "Traffic switched to green (v2.0)"
    pause 1

    step "Decommissioning blue (v1.0)..."
    $KONSULCTL service deregister payment-service-1 2>/dev/null || true
    success "Blue environment decommissioned"
    pause 2

    log "Blue-green deployment complete!"
    pause 2
}

# Scenario 5: Disaster Recovery
scenario_disaster_recovery() {
    banner "Scenario 5: Backup and Recovery"

    log "Demonstrating backup and recovery capabilities..."
    pause

    step "Creating backup of current state..."
    local backup_file="/tmp/konsul-backup-$(date +%Y%m%d-%H%M%S).json"

    if [ -f "$KONSULCTL" ]; then
        $KONSULCTL backup create --output "$backup_file" || {
            # Fallback to API
            curl -sf "${KONSUL_URL}/backup" > "$backup_file"
        }
    fi

    success "Backup created: $backup_file"
    pause 1

    step "Simulating data loss (deleting some KV entries)..."
    $KONSULCTL kv delete feature-flags/new-ui-redesign 2>/dev/null || true
    $KONSULCTL kv delete config/ecommerce/api/rate-limit 2>/dev/null || true
    log "Data deleted"
    pause 1

    step "Restoring from backup..."
    if [ -f "$KONSULCTL" ]; then
        $KONSULCTL backup restore --input "$backup_file" || {
            curl -sf -X POST "${KONSUL_URL}/backup/restore" \
                --data-binary "@${backup_file}"
        }
    fi
    success "Backup restored"
    pause 1

    step "Verifying restoration..."
    $KONSULCTL kv get feature-flags/new-ui-redesign || log "Verification skipped"
    pause 2

    log "Disaster recovery complete!"
    pause 2
}

# Cleanup
cleanup() {
    banner "Cleanup"

    log "Would you like to clean up all demo data? (y/N)"
    read -t 10 -n 1 -r answer || answer="n"
    echo

    if [[ $answer =~ ^[Yy]$ ]]; then
        step "Deregistering services..."

        # Get all service IDs and deregister
        service_ids=(
            "web-frontend-1" "web-frontend-2"
            "api-gateway-1"
            "product-service-1" "product-service-2" "product-service-3" "product-service-4"
            "order-service-1" "order-service-2-canary"
            "payment-service-1" "payment-service-2" "payment-service-2-green"
            "postgres-primary" "postgres-replica-1"
            "redis-cache-1"
        )

        for service_id in "${service_ids[@]}"; do
            $KONSULCTL service deregister "$service_id" 2>/dev/null || true
        done

        success "Services deregistered"

        step "Deleting KV entries..."
        # Note: konsulctl might not have a delete-all command, so we skip this
        log "KV cleanup skipped (manual cleanup required)"

        success "Cleanup complete!"
    else
        log "Cleanup skipped. Services and KV data retained."
    fi

    pause
}

# Main menu
main() {
    echo ""
    echo -e "${MAGENTA}╔════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${MAGENTA}║                                                            ║${NC}"
    echo -e "${MAGENTA}║           Konsul Demo Scenario Runner                     ║${NC}"
    echo -e "${MAGENTA}║                                                            ║${NC}"
    echo -e "${MAGENTA}╚════════════════════════════════════════════════════════════╝${NC}"
    echo ""

    check_prerequisites

    banner "Available Scenarios"
    echo "  1. E-commerce Platform Deployment"
    echo "  2. Service Scaling and Health Checks"
    echo "  3. Dynamic Configuration Management"
    echo "  4. Blue-Green Deployment"
    echo "  5. Backup and Recovery"
    echo "  6. Run All Scenarios"
    echo "  0. Exit"
    echo ""

    read -p "Select scenario (0-6): " choice

    case $choice in
        1)
            scenario_ecommerce
            ;;
        2)
            scenario_scaling
            ;;
        3)
            scenario_config
            ;;
        4)
            scenario_migration
            ;;
        5)
            scenario_disaster_recovery
            ;;
        6)
            scenario_ecommerce
            scenario_scaling
            scenario_config
            scenario_migration
            scenario_disaster_recovery
            ;;
        0)
            log "Exiting..."
            exit 0
            ;;
        *)
            error "Invalid choice"
            exit 1
            ;;
    esac

    cleanup
}

# Run main
main "$@"
