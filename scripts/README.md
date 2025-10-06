# Konsul Testing Scripts

Collection of scripts for load testing, demos, and continuous traffic generation.

## Scripts Overview

### 1. load-test.sh - Load Testing Tool

Simulates realistic application traffic with multiple concurrent workers.

**Features:**
- Concurrent workers (default: 5)
- Weighted operation distribution
- Service registration/deregistration
- Heartbeat simulation
- KV store operations
- Statistics tracking

**Usage:**
```bash
# Basic usage (60 seconds, 5 workers)
./scripts/load-test.sh

# Custom duration and workers
DURATION=300 CONCURRENT_WORKERS=10 ./scripts/load-test.sh

# Different Konsul instance
KONSUL_URL=http://localhost:9999 ./scripts/load-test.sh

# All options
KONSUL_URL=http://localhost:8888 \
  DURATION=120 \
  CONCURRENT_WORKERS=8 \
  ./scripts/load-test.sh
```

**Operation Distribution:**
- 20% - Service registration
- 5% - Service deregistration
- 20% - Heartbeats
- 15% - KV set operations
- 10% - KV get operations
- 5% - KV delete operations
- 20% - Service list queries
- 5% - Health checks

**Example Output:**
```
╔════════════════════════════════════════════════════════════╗
║           Konsul Load Testing Script                      ║
╚════════════════════════════════════════════════════════════╝

[14:23:45] Configuration:
  Konsul URL:          http://localhost:8888
  Duration:            60s
  Concurrent Workers:  5

✓ Konsul is running
[14:23:46] Starting 5 workers...
✓ Workers started (PIDs: 12345 12346 12347 12348 12349)
[14:23:46] Running load test for 60 seconds...
  Progress: [####################] 100%

[14:24:46] Load test completed!
[14:24:46] Statistics:
  Total Requests:      1247
  Successful:          1238
  Failed:              9
  Success Rate:        99%
```

### 2. demo-scenario.sh - Interactive Demo

Demonstrates realistic microservices scenarios using konsulctl CLI.

**Features:**
- E-commerce platform deployment
- Service scaling simulation
- Dynamic configuration management
- Blue-green deployment
- Backup and recovery

**Usage:**
```bash
# Interactive mode
./scripts/demo-scenario.sh

# Specify konsulctl path
KONSULCTL=/path/to/konsulctl ./scripts/demo-scenario.sh

# Different Konsul instance
KONSUL_URL=http://localhost:9999 ./scripts/demo-scenario.sh
```

**Scenarios:**

#### Scenario 1: E-commerce Platform
Deploys a complete e-commerce stack:
- 2x Frontend (React)
- 1x API Gateway
- 4x Backend services (Product, Order, Payment)
- 3x Data layer (PostgreSQL, Redis)
- Configuration in KV store

#### Scenario 2: Service Scaling
Demonstrates auto-scaling:
- Scales product-service from 2 to 4 instances
- Simulates failing instance
- Canary deployment for order-service

#### Scenario 3: Configuration Management
Shows KV store usage:
- Feature flags
- Service discovery settings
- Environment-specific config

#### Scenario 4: Blue-Green Deployment
Zero-downtime deployment:
- Deploy green environment
- Run smoke tests
- Switch traffic
- Decommission blue

#### Scenario 5: Backup and Recovery
Disaster recovery demo:
- Create backup
- Simulate data loss
- Restore from backup

**Example Session:**
```
╔════════════════════════════════════════════════════════════╗
║           Konsul Demo Scenario Runner                     ║
╚════════════════════════════════════════════════════════════╝

Available Scenarios:
  1. E-commerce Platform Deployment
  2. Service Scaling and Health Checks
  3. Dynamic Configuration Management
  4. Blue-Green Deployment
  5. Backup and Recovery
  6. Run All Scenarios
  0. Exit

Select scenario (0-6): 1

╔════════════════════════════════════════════════════════════╗
║ Scenario 1: E-commerce Platform Deployment                ║
╚════════════════════════════════════════════════════════════╝

[14:30:12] Deploying e-commerce microservices...
▶ Registering frontend services...
✓ Frontend services registered (2 instances)
▶ Registering API gateway...
✓ API gateway registered
...
```

### 3. continuous-traffic.sh - Steady Traffic Generator

Generates continuous background traffic for observability testing.

**Features:**
- Persistent service pool
- Automatic heartbeats
- Random KV operations
- Real-time statistics
- Graceful shutdown

**Usage:**
```bash
# Default (5 second interval)
./scripts/continuous-traffic.sh

# Custom interval
INTERVAL=10 ./scripts/continuous-traffic.sh

# Run in background
./scripts/continuous-traffic.sh &

# Stop with Ctrl+C for statistics
```

**Operation Mix:**
- Heartbeats every cycle
- 30% KV write operations
- 20% KV read operations
- 30% Service list queries
- 20% Health/metrics checks

**Example Output:**
```
╔════════════════════════════════════════════════════════════╗
║      Konsul Continuous Traffic Generator                  ║
╚════════════════════════════════════════════════════════════╝

[14:35:20] Target: http://localhost:8888
[14:35:20] Interval: 5s

✓ Services registered
[14:35:22] Starting continuous traffic (Ctrl+C to stop)...

Cycle: 42 | Heartbeats: 42 | KV Ops: 127 | Queries: 89 | Health: 34

^C
[14:40:45] Stopping traffic generator...

[14:40:45] Final Statistics:
  Cycles completed: 42
  Heartbeats sent: 42
  KV operations: 127
  Service queries: 89
  Health checks: 34
```

## Prerequisites

### All Scripts
```bash
# Konsul must be running
docker-compose -f docker-compose.observability.yml up -d
# or
./bin/konsul

# Verify
curl http://localhost:8888/health
```

### demo-scenario.sh Only
```bash
# Build konsulctl CLI
make build-cli

# Verify
./bin/konsulctl --help
```

## Use Cases

### 1. Testing Observability Stack

Generate traffic to populate metrics, logs, and traces:

```bash
# Terminal 1: Start observability stack
docker-compose -f docker-compose.observability.yml up -d

# Terminal 2: Generate continuous traffic
./scripts/continuous-traffic.sh

# Terminal 3: Run load test
DURATION=300 CONCURRENT_WORKERS=10 ./scripts/load-test.sh
```

Then view in Grafana:
- **Metrics**: http://localhost:3000 (Konsul Dashboard)
- **Logs**: http://localhost:3000/explore (Loki)
- **Traces**: http://localhost:3000/explore (Tempo)

### 2. Performance Testing

Test Konsul under load:

```bash
# High load test
DURATION=600 CONCURRENT_WORKERS=20 ./scripts/load-test.sh

# Monitor metrics in real-time
watch -n 2 'curl -s http://localhost:8888/metrics | grep konsul_http_requests_total'

# Check Grafana dashboard
open http://localhost:3000
```

### 3. Demo for Stakeholders

Run interactive scenarios:

```bash
# Run all scenarios
./scripts/demo-scenario.sh
# Select option 6 (Run All Scenarios)
```

### 4. CI/CD Integration

Use in automated testing:

```bash
#!/bin/bash
# ci-test.sh

# Start Konsul
docker-compose up -d konsul

# Wait for ready
timeout 30 bash -c 'until curl -sf http://localhost:8888/health; do sleep 1; done'

# Run load test
DURATION=60 CONCURRENT_WORKERS=5 ./scripts/load-test.sh

# Check success rate (should be >95%)
# (Add validation logic here)

# Cleanup
docker-compose down
```

### 5. Chaos Engineering

Simulate failures during traffic:

```bash
# Terminal 1: Generate traffic
./scripts/continuous-traffic.sh

# Terminal 2: Simulate failures
# Stop Konsul
docker stop konsul

# Restart after 30s
sleep 30
docker start konsul

# Observe recovery in logs and metrics
```

## Environment Variables

All scripts support:

| Variable | Default | Description |
|----------|---------|-------------|
| `KONSUL_URL` | `http://localhost:8888` | Konsul API endpoint |
| `DURATION` | `60` | Test duration in seconds (load-test only) |
| `CONCURRENT_WORKERS` | `5` | Number of workers (load-test only) |
| `INTERVAL` | `5` | Seconds between cycles (continuous-traffic only) |
| `KONSULCTL` | `./bin/konsulctl` | Path to konsulctl binary (demo only) |

## Tips

### Observing Traces

1. Start continuous traffic:
   ```bash
   ./scripts/continuous-traffic.sh
   ```

2. Open Grafana: http://localhost:3000

3. Navigate to **Explore** → Select **Tempo**

4. Search for traces:
   - Service name: `konsul`
   - Operation: `POST /services`

5. Click on a trace to see:
   - Span details
   - Request attributes
   - Linked logs (via trace_id)

### Viewing Logs

1. Generate traffic with load test

2. Open Grafana Explore → Select **Loki**

3. Query examples:
   ```logql
   # All Konsul logs
   {service="konsul"}

   # Errors only
   {service="konsul"} | json | level="error"

   # Slow requests (>100ms)
   {service="konsul"} | json | duration > 0.1

   # Specific trace
   {service="konsul"} | json | trace_id="abc123..."
   ```

### Monitoring Metrics

1. Run load test:
   ```bash
   DURATION=300 ./scripts/load-test.sh
   ```

2. View in Prometheus: http://localhost:9090

3. Example queries:
   ```promql
   # Request rate
   rate(konsul_http_requests_total[5m])

   # Error rate
   rate(konsul_http_requests_total{status=~"5.."}[5m])

   # P95 latency
   histogram_quantile(0.95, rate(konsul_http_request_duration_seconds_bucket[5m]))

   # Active services
   konsul_registered_services_total
   ```

## Troubleshooting

### Connection Refused

```bash
# Check if Konsul is running
docker ps | grep konsul
# or
ps aux | grep konsul

# Check health endpoint
curl http://localhost:8888/health
```

### konsulctl Not Found (demo-scenario.sh)

```bash
# Build CLI
make build-cli

# Or specify path
KONSULCTL=/path/to/konsulctl ./scripts/demo-scenario.sh
```

### High Error Rate in Load Test

```bash
# Check Konsul logs
docker logs konsul --tail 50

# Reduce load
CONCURRENT_WORKERS=2 ./scripts/load-test.sh

# Check system resources
docker stats konsul
```

### No Data in Grafana

```bash
# Verify Prometheus is scraping
curl http://localhost:9090/targets

# Check Konsul metrics endpoint
curl http://localhost:8888/metrics

# Verify datasources in Grafana
# http://localhost:3000 → Configuration → Data Sources
```

## Contributing

To add new scenarios or improve scripts:

1. Follow existing script structure
2. Use color functions for output
3. Add error handling
4. Update this README
5. Test with observability stack

## License

Same as Konsul project.
