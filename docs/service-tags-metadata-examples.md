# Service Tags and Metadata - Usage Examples

**Based on**: ADR-0017 (Service Tags and Metadata)
**Created**: 2025-10-28

This document provides practical examples for using service tags and metadata in Konsul.

---

## Table of Contents

1. [Basic Registration](#basic-registration)
2. [Multi-Environment Deployments](#multi-environment-deployments)
3. [Canary Deployments](#canary-deployments)
4. [Geographic Routing](#geographic-routing)
5. [Protocol-Based Discovery](#protocol-based-discovery)
6. [Team Ownership Tracking](#team-ownership-tracking)
7. [Cost Management](#cost-management)
8. [Service Mesh Integration](#service-mesh-integration)
9. [Blue-Green Deployments](#blue-green-deployments)
10. [Feature Flags](#feature-flags)

---

## Basic Registration

### Register Service with Tags

**HTTP API**:
```bash
curl -X POST http://localhost:8500/v1/agent/service/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "api-service",
    "address": "10.0.1.50",
    "port": 8080,
    "tags": ["http", "v1.0.0", "production"]
  }'
```

**konsulctl CLI**:
```bash
konsulctl service register api-service \
  --address 10.0.1.50 \
  --port 8080 \
  --tag http \
  --tag v1.0.0 \
  --tag production
```

**Go Client**:
```go
package main

import (
    "github.com/neogan74/konsul/pkg/client"
)

func main() {
    c := client.NewClient("http://localhost:8500")

    service := &client.Service{
        Name:    "api-service",
        Address: "10.0.1.50",
        Port:    8080,
        Tags:    []string{"http", "v1.0.0", "production"},
    }

    err := c.Service.Register(service)
    if err != nil {
        log.Fatal(err)
    }
}
```

### Register Service with Metadata

**HTTP API**:
```bash
curl -X POST http://localhost:8500/v1/agent/service/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "api-service",
    "address": "10.0.1.50",
    "port": 8080,
    "tags": ["http", "production"],
    "meta": {
      "team": "platform",
      "owner": "alice@example.com",
      "version": "1.0.0",
      "git-commit": "abc123def"
    }
  }'
```

**konsulctl CLI**:
```bash
konsulctl service register api-service \
  --address 10.0.1.50 \
  --port 8080 \
  --tag http \
  --tag production \
  --meta team=platform \
  --meta owner=alice@example.com \
  --meta version=1.0.0 \
  --meta git-commit=abc123def
```

---

## Multi-Environment Deployments

### Scenario: Separate Production, Staging, and Development

**Register Production Service**:
```bash
curl -X POST http://localhost:8500/v1/agent/service/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "web-app",
    "address": "10.0.1.100",
    "port": 8080,
    "tags": ["env:production", "http", "version:2.1.0"],
    "meta": {
      "environment": "production",
      "datacenter": "us-east-1",
      "replicas": "5"
    }
  }'
```

**Register Staging Service**:
```bash
curl -X POST http://localhost:8500/v1/agent/service/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "web-app",
    "address": "10.0.2.100",
    "port": 8080,
    "tags": ["env:staging", "http", "version:2.2.0-rc1"],
    "meta": {
      "environment": "staging",
      "datacenter": "us-east-1",
      "replicas": "2"
    }
  }'
```

**Register Development Service**:
```bash
curl -X POST http://localhost:8500/v1/agent/service/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "web-app",
    "address": "10.0.3.100",
    "port": 8080,
    "tags": ["env:development", "http", "version:2.2.0-dev"],
    "meta": {
      "environment": "development",
      "datacenter": "us-east-1",
      "replicas": "1"
    }
  }'
```

### Query Production Services Only

**HTTP API**:
```bash
curl "http://localhost:8500/v1/catalog/services?tag=env:production"
```

**konsulctl CLI**:
```bash
konsulctl service list --tag env:production
```

**DNS Query**:
```bash
dig @localhost -p 8600 production.web-app.service.konsul
```

**Application Code (Go)**:
```go
func getProductionServices(c *client.Client) ([]client.Service, error) {
    return c.Catalog.Query(client.QueryOptions{
        Tags: []string{"env:production"},
    })
}
```

---

## Canary Deployments

### Scenario: Gradual rollout of v2.0.0

**Register Stable Version (v1.0.0)**:
```bash
# Register 3 instances of stable version
for i in {1..3}; do
  curl -X POST http://localhost:8500/v1/agent/service/register \
    -H "Content-Type: application/json" \
    -d "{
      \"name\": \"api-service\",
      \"address\": \"10.0.1.${i}\",
      \"port\": 8080,
      \"tags\": [\"version:v1.0.0\", \"stable\", \"http\"],
      \"meta\": {
        \"deployment\": \"stable\",
        \"build-date\": \"2025-10-01\"
      }
    }"
done
```

**Register Canary Version (v2.0.0)**:
```bash
# Register 1 canary instance (25% traffic)
curl -X POST http://localhost:8500/v1/agent/service/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "api-service",
    "address": "10.0.1.10",
    "port": 8080,
    "tags": ["version:v2.0.0", "canary", "http"],
    "meta": {
      "deployment": "canary",
      "build-date": "2025-10-28"
    }
  }'
```

### Query Canary Instances

**HTTP API**:
```bash
curl "http://localhost:8500/v1/catalog/services?tag=canary"
```

**Query Stable Instances**:
```bash
curl "http://localhost:8500/v1/catalog/services?tag=stable"
```

**Query Specific Version**:
```bash
curl "http://localhost:8500/v1/catalog/services?tag=version:v2.0.0"
```

### Load Balancer Configuration (NGINX)

```nginx
upstream api_stable {
    # Query stable instances
    server 10.0.1.1:8080;
    server 10.0.1.2:8080;
    server 10.0.1.3:8080;
}

upstream api_canary {
    # Query canary instances
    server 10.0.1.10:8080;
}

server {
    listen 80;

    location / {
        # Route 75% to stable, 25% to canary
        if ($request_id ~ "^[0-3]") {
            proxy_pass http://api_canary;
        }
        proxy_pass http://api_stable;
    }
}
```

### Promote Canary to Stable

```bash
# Update canary instance to stable tag
curl -X POST http://localhost:8500/v1/agent/service/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "api-service",
    "address": "10.0.1.10",
    "port": 8080,
    "tags": ["version:v2.0.0", "stable", "http"],
    "meta": {
      "deployment": "stable",
      "build-date": "2025-10-28"
    }
  }'

# Gradually deregister old stable instances
konsulctl service deregister api-service --address 10.0.1.1
```

---

## Geographic Routing

### Scenario: Multi-Region Deployment

**Register Services in US-East-1**:
```bash
curl -X POST http://localhost:8500/v1/agent/service/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "api-service",
    "address": "10.1.0.50",
    "port": 8080,
    "tags": [
      "region:us-east-1",
      "az:us-east-1a",
      "datacenter:aws",
      "http"
    ],
    "meta": {
      "region": "us-east-1",
      "availability-zone": "us-east-1a",
      "provider": "aws"
    }
  }'
```

**Register Services in US-West-2**:
```bash
curl -X POST http://localhost:8500/v1/agent/service/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "api-service",
    "address": "10.2.0.50",
    "port": 8080,
    "tags": [
      "region:us-west-2",
      "az:us-west-2a",
      "datacenter:aws",
      "http"
    ],
    "meta": {
      "region": "us-west-2",
      "availability-zone": "us-west-2a",
      "provider": "aws"
    }
  }'
```

**Register Services in EU-West-1**:
```bash
curl -X POST http://localhost:8500/v1/agent/service/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "api-service",
    "address": "10.3.0.50",
    "port": 8080,
    "tags": [
      "region:eu-west-1",
      "az:eu-west-1a",
      "datacenter:aws",
      "http"
    ],
    "meta": {
      "region": "eu-west-1",
      "availability-zone": "eu-west-1a",
      "provider": "aws"
    }
  }'
```

### Query Services by Region

**US-East-1 Services**:
```bash
curl "http://localhost:8500/v1/catalog/services?tag=region:us-east-1"
```

**EU Services**:
```bash
curl "http://localhost:8500/v1/catalog/services?tag=region:eu-west-1"
```

**DNS Query**:
```bash
dig @localhost -p 8600 us-east-1.api-service.service.konsul
```

### Application Code: Nearest Datacenter Routing

```go
package main

import (
    "github.com/neogan74/konsul/pkg/client"
)

func getServicesInRegion(c *client.Client, region string) ([]client.Service, error) {
    return c.Catalog.Query(client.QueryOptions{
        Tags: []string{fmt.Sprintf("region:%s", region)},
    })
}

func routeToNearestService(clientRegion string) (string, error) {
    c := client.NewClient("http://localhost:8500")

    services, err := getServicesInRegion(c, clientRegion)
    if err != nil {
        return "", err
    }

    if len(services) == 0 {
        // Fallback to another region
        services, err = getServicesInRegion(c, "us-east-1")
        if err != nil {
            return "", err
        }
    }

    // Return first available service
    return fmt.Sprintf("%s:%d", services[0].Address, services[0].Port), nil
}
```

---

## Protocol-Based Discovery

### Scenario: Service Mesh with Multiple Protocols

**Register HTTP Service**:
```bash
curl -X POST http://localhost:8500/v1/agent/service/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "web-frontend",
    "address": "10.0.1.100",
    "port": 8080,
    "tags": ["protocol:http", "frontend"],
    "meta": {
      "protocol": "http",
      "ssl": "true"
    }
  }'
```

**Register gRPC Service**:
```bash
curl -X POST http://localhost:8500/v1/agent/service/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "user-service",
    "address": "10.0.1.101",
    "port": 9090,
    "tags": ["protocol:grpc", "backend"],
    "meta": {
      "protocol": "grpc",
      "tls": "true"
    }
  }'
```

**Register TCP Service**:
```bash
curl -X POST http://localhost:8500/v1/agent/service/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "database-proxy",
    "address": "10.0.1.102",
    "port": 5432,
    "tags": ["protocol:tcp", "database"],
    "meta": {
      "protocol": "tcp",
      "database-type": "postgresql"
    }
  }'
```

### Query by Protocol

**All HTTP Services**:
```bash
curl "http://localhost:8500/v1/catalog/services?tag=protocol:http"
```

**All gRPC Services**:
```bash
curl "http://localhost:8500/v1/catalog/services?tag=protocol:grpc"
```

### Service Mesh Sidecar Configuration

```yaml
# Envoy proxy configuration
static_resources:
  clusters:
  - name: http_services
    type: LOGICAL_DNS
    dns_lookup_family: V4_ONLY
    load_assignment:
      cluster_name: http_services
      endpoints:
      - lb_endpoints:
        # Query: http.service.konsul (DNS)
        - endpoint:
            address:
              socket_address:
                address: http.service.konsul
                port_value: 8080

  - name: grpc_services
    type: LOGICAL_DNS
    http2_protocol_options: {}
    dns_lookup_family: V4_ONLY
    load_assignment:
      cluster_name: grpc_services
      endpoints:
      - lb_endpoints:
        # Query: grpc.service.konsul (DNS)
        - endpoint:
            address:
              socket_address:
                address: grpc.service.konsul
                port_value: 9090
```

---

## Team Ownership Tracking

### Scenario: Track Service Ownership for Oncall and Alerts

**Register Services with Ownership**:

**Platform Team Service**:
```bash
curl -X POST http://localhost:8500/v1/agent/service/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "auth-service",
    "address": "10.0.1.50",
    "port": 8080,
    "tags": ["backend", "critical"],
    "meta": {
      "team": "platform",
      "owner": "alice@example.com",
      "oncall-slack": "#platform-oncall",
      "pagerduty": "platform-pd",
      "documentation": "https://wiki.example.com/auth-service",
      "git-repo": "https://github.com/example/auth-service",
      "sla-tier": "tier-1"
    }
  }'
```

**Data Team Service**:
```bash
curl -X POST http://localhost:8500/v1/agent/service/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "analytics-service",
    "address": "10.0.1.51",
    "port": 8081,
    "tags": ["backend", "batch"],
    "meta": {
      "team": "data",
      "owner": "bob@example.com",
      "oncall-slack": "#data-oncall",
      "pagerduty": "data-pd",
      "documentation": "https://wiki.example.com/analytics-service",
      "git-repo": "https://github.com/example/analytics-service",
      "sla-tier": "tier-2"
    }
  }'
```

### Query Services by Team

**Platform Team Services**:
```bash
curl "http://localhost:8500/v1/catalog/services?meta=team:platform"
```

**Data Team Services**:
```bash
curl "http://localhost:8500/v1/catalog/services?meta=team:data"
```

**konsulctl**:
```bash
konsulctl service list --meta team:platform
```

### Alert Routing Configuration

**Prometheus Alertmanager**:
```yaml
route:
  group_by: ['alertname', 'service']
  receiver: 'default'
  routes:
  - match:
      team: platform
    receiver: platform-oncall
  - match:
      team: data
    receiver: data-oncall

receivers:
- name: 'platform-oncall'
  slack_configs:
  - channel: '#platform-oncall'
    api_url: 'https://hooks.slack.com/...'

- name: 'data-oncall'
  slack_configs:
  - channel: '#data-oncall'
    api_url: 'https://hooks.slack.com/...'
```

### Generate Team Dashboard

**Script to List Services by Team**:
```bash
#!/bin/bash

teams=("platform" "data" "frontend" "mobile")

for team in "${teams[@]}"; do
  echo "=== $team Team Services ==="
  konsulctl service list --meta team:$team --format json | \
    jq -r '.[] | "\(.name) - Owner: \(.meta.owner) - Oncall: \(.meta["oncall-slack"])"'
  echo
done
```

---

## Cost Management

### Scenario: Track Infrastructure Costs by Project

**Register Services with Cost Metadata**:

**Project Alpha Services**:
```bash
curl -X POST http://localhost:8500/v1/agent/service/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "alpha-api",
    "address": "10.0.1.100",
    "port": 8080,
    "tags": ["project:alpha", "production"],
    "meta": {
      "project": "alpha",
      "cost-center": "engineering",
      "billing-code": "ENG-001",
      "budget-owner": "alice@example.com",
      "instance-type": "t3.large",
      "monthly-cost": "50"
    }
  }'
```

**Project Beta Services**:
```bash
curl -X POST http://localhost:8500/v1/agent/service/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "beta-api",
    "address": "10.0.1.101",
    "port": 8080,
    "tags": ["project:beta", "production"],
    "meta": {
      "project": "beta",
      "cost-center": "product",
      "billing-code": "PROD-002",
      "budget-owner": "bob@example.com",
      "instance-type": "t3.xlarge",
      "monthly-cost": "100"
    }
  }'
```

### Query Services by Project

```bash
curl "http://localhost:8500/v1/catalog/services?meta=project:alpha"
```

### Generate Cost Report

**Script**:
```bash
#!/bin/bash

echo "=== Monthly Cost Report ==="
echo

projects=("alpha" "beta")

total_cost=0

for project in "${projects[@]}"; do
  echo "Project: $project"

  services=$(curl -s "http://localhost:8500/v1/catalog/services?meta=project:$project")

  project_cost=0
  echo "$services" | jq -r '.[] |
    "  \(.name): $\(.meta["monthly-cost"]) (\(.meta["instance-type"]})"'

  project_cost=$(echo "$services" | jq -r '[.[] | .meta["monthly-cost"] | tonumber] | add')
  total_cost=$(echo "$total_cost + $project_cost" | bc)

  echo "  Project Total: \$$project_cost"
  echo
done

echo "=== Total Monthly Cost: \$$total_cost ==="
```

---

## Service Mesh Integration

### Scenario: Istio Service Mesh with Traffic Routing

**Register Services with Mesh Metadata**:
```bash
curl -X POST http://localhost:8500/v1/agent/service/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "payment-service",
    "address": "10.0.1.100",
    "port": 8080,
    "tags": [
      "mesh:enabled",
      "version:v1",
      "protocol:http"
    ],
    "meta": {
      "mesh-version": "1.0",
      "sidecar": "envoy",
      "mTLS": "strict",
      "circuit-breaker": "enabled",
      "retry-policy": "3x-backoff"
    }
  }'
```

### Istio VirtualService Configuration

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: payment-service
spec:
  hosts:
  - payment-service
  http:
  - match:
    - headers:
        x-canary:
          exact: "true"
    route:
    - destination:
        host: payment-service
        subset: canary  # Query: tag=canary
  - route:
    - destination:
        host: payment-service
        subset: stable  # Query: tag=stable
---
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: payment-service
spec:
  host: payment-service
  trafficPolicy:
    connectionPool:
      tcp:
        maxConnections: 100
      http:
        http1MaxPendingRequests: 50
        http2MaxRequests: 100
    outlierDetection:
      consecutiveErrors: 7
      interval: 5m
      baseEjectionTime: 15m
  subsets:
  - name: stable
    labels:
      version: v1
  - name: canary
    labels:
      version: v2
```

---

## Blue-Green Deployments

### Scenario: Zero-Downtime Deployment

**Register Blue Environment (Current)**:
```bash
for i in {1..3}; do
  curl -X POST http://localhost:8500/v1/agent/service/register \
    -H "Content-Type: application/json" \
    -d "{
      \"name\": \"app-service\",
      \"address\": \"10.0.1.${i}\",
      \"port\": 8080,
      \"tags\": [\"deployment:blue\", \"active\", \"version:1.0.0\"],
      \"meta\": {
        \"deployment\": \"blue\",
        \"status\": \"active\"
      }
    }"
done
```

**Register Green Environment (New Version)**:
```bash
for i in {11..13}; do
  curl -X POST http://localhost:8500/v1/agent/service/register \
    -H "Content-Type: application/json" \
    -d "{
      \"name\": \"app-service\",
      \"address\": \"10.0.1.${i}\",
      \"port\": 8080,
      \"tags\": [\"deployment:green\", \"standby\", \"version:2.0.0\"],
      \"meta\": {
        \"deployment\": \"green\",
        \"status\": \"standby\"
      }
    }"
done
```

### Query Active Deployment

```bash
curl "http://localhost:8500/v1/catalog/services?tag=active"
```

### Switch to Green (Promote)

```bash
# Update green to active
for i in {11..13}; do
  curl -X POST http://localhost:8500/v1/agent/service/register \
    -H "Content-Type: application/json" \
    -d "{
      \"name\": \"app-service\",
      \"address\": \"10.0.1.${i}\",
      \"port\": 8080,
      \"tags\": [\"deployment:green\", \"active\", \"version:2.0.0\"],
      \"meta\": {
        \"deployment\": \"green\",
        \"status\": \"active\"
      }
    }"
done

# Update blue to standby
for i in {1..3}; do
  curl -X POST http://localhost:8500/v1/agent/service/register \
    -H "Content-Type: application/json" \
    -d "{
      \"name\": \"app-service\",
      \"address\": \"10.0.1.${i}\",
      \"port\": 8080,
      \"tags\": [\"deployment:blue\", \"standby\", \"version:1.0.0\"],
      \"meta\": {
        \"deployment\": \"blue\",
        \"status\": \"standby\"
      }
    }"
done
```

### Rollback to Blue

```bash
# Just reverse the tags
# Update blue to active, green to standby
```

---

## Feature Flags

### Scenario: Feature-Gated Services

**Register Services with Feature Flags**:

**Service with New Feature Enabled**:
```bash
curl -X POST http://localhost:8500/v1/agent/service/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "api-service",
    "address": "10.0.1.100",
    "port": 8080,
    "tags": [
      "feature:new-dashboard",
      "feature:advanced-analytics",
      "version:2.0.0"
    ],
    "meta": {
      "features": "new-dashboard,advanced-analytics",
      "feature-flags-version": "2.0"
    }
  }'
```

**Service with Legacy Features**:
```bash
curl -X POST http://localhost:8500/v1/agent/service/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "api-service",
    "address": "10.0.1.101",
    "port": 8080,
    "tags": [
      "version:1.0.0"
    ],
    "meta": {
      "features": "legacy",
      "feature-flags-version": "1.0"
    }
  }'
```

### Route Traffic Based on Feature Flag

**Query Services with Feature**:
```bash
curl "http://localhost:8500/v1/catalog/services?tag=feature:new-dashboard"
```

**Application Code**:
```go
func getServiceWithFeature(c *client.Client, feature string) (*client.Service, error) {
    services, err := c.Catalog.Query(client.QueryOptions{
        Tags: []string{fmt.Sprintf("feature:%s", feature)},
    })
    if err != nil {
        return nil, err
    }

    if len(services) == 0 {
        // Fallback to services without the feature
        services, err = c.Catalog.Query(client.QueryOptions{})
        if err != nil {
            return nil, err
        }
    }

    return &services[0], nil
}
```

---

## Advanced Queries

### Combine Multiple Filters

**Production services in US-East-1**:
```bash
curl "http://localhost:8500/v1/catalog/services?tag=env:production&tag=region:us-east-1"
```

**HTTP services owned by platform team**:
```bash
curl "http://localhost:8500/v1/catalog/services?tag=protocol:http&meta=team:platform"
```

**Tier-1 services with canary deployment**:
```bash
curl "http://localhost:8500/v1/catalog/services?tag=canary&meta=sla-tier:tier-1"
```

### CLI Examples

**List all production HTTP services**:
```bash
konsulctl service list --tag env:production --tag protocol:http
```

**List all platform team services**:
```bash
konsulctl service list --meta team:platform
```

**List canary services in staging**:
```bash
konsulctl service list --tag canary --tag env:staging
```

---

## Monitoring & Alerting

### Prometheus Queries with Service Metadata

**Query by Tag**:
```promql
sum by (service) (konsul_service_tags_total{service=~".*"})
```

**Alert on Services without Tags**:
```yaml
- alert: ServiceMissingTags
  expr: konsul_service_tags_total == 0
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "Service {{ $labels.service }} has no tags"
```

**Track Services by Environment**:
```promql
count(konsul_catalog_queries_total{filter_type="tag", tag="env:production"})
```

---

## Best Practices

### Tag Naming Conventions

**Good**:
```
env:production
version:v1.2.3
region:us-east-1
protocol:http
canary
stable
```

**Bad**:
```
PRODUCTION           # Use env:production
v1.2.3              # Use version:v1.2.3
US-EAST-1           # Use lowercase
http_protocol       # Use protocol:http
```

### Metadata Key Conventions

**Good**:
```json
{
  "team": "platform",
  "owner": "alice@example.com",
  "cost-center": "engineering",
  "git-repo": "https://github.com/..."
}
```

**Bad**:
```json
{
  "Team": "platform",           // Use lowercase
  "owner_email": "alice",      // Use owner
  "cost center": "engineering" // No spaces
}
```

---

## Summary

This document demonstrated:
- ✅ Basic service registration with tags and metadata
- ✅ Multi-environment deployments
- ✅ Canary and blue-green deployments
- ✅ Geographic routing
- ✅ Protocol-based discovery
- ✅ Team ownership tracking
- ✅ Cost management
- ✅ Service mesh integration
- ✅ Feature flags
- ✅ Advanced filtering

For implementation details, see `service-tags-metadata-tasks.md`.
