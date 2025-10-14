# DNS Server - Troubleshooting Guide

Comprehensive troubleshooting guide for Konsul DNS server issues.

## Quick Diagnostics

### Health Check Command

```bash
# Test DNS server is responding
dig @localhost -p 8600 test.service.consul A +short

# Check if port is listening
netstat -tulpn | grep 8600

# Test with real DNS client
nslookup test.service.consul localhost -port=8600
```

---

## Common Issues

### Issue 1: DNS Server Not Starting

#### Symptom

```bash
# Konsul logs show:
ERROR: DNS UDP server failed: bind: address already in use
```

#### Diagnosis

```bash
# Check what's using port 8600
sudo lsof -i :8600
sudo netstat -tulpn | grep 8600

# Common culprits:
# - Another Konsul instance
# - Consul (HashiCorp)
# - systemd-resolved (on Ubuntu)
# - dnsmasq
```

#### Solution 1: Kill Conflicting Process

```bash
# Find process ID
sudo lsof -ti :8600

# Kill it
sudo kill $(sudo lsof -ti :8600)
```

#### Solution 2: Change DNS Port

```yaml
# config.yaml
dns:
  enabled: true
  port: 5353  # Use alternative port
```

#### Solution 3: Disable Systemd-Resolved (Ubuntu)

```bash
# Check if systemd-resolved is running
systemctl status systemd-resolved

# Disable stub resolver (frees port 53)
sudo sed -i 's/#DNSStubListener=yes/DNSStubListener=no/' /etc/systemd/resolved.conf
sudo systemctl restart systemd-resolved
```

---

### Issue 2: Query Returns NXDOMAIN

#### Symptom

```bash
dig @localhost -p 8600 web.service.consul A

# Response:
;; ->>HEADER<<- opcode: QUERY, status: NXDOMAIN
```

#### Diagnosis Steps

**1. Check Service is Registered**

```bash
curl http://localhost:8500/services | jq .
```

Expected output:
```json
[
  {
    "name": "web",
    "address": "10.0.0.1",
    "port": 8080
  }
]
```

**2. Verify Service Name**

```bash
# List all service names
curl http://localhost:8500/services | jq '.[].name'
```

**3. Check DNS Query Format**

Common mistakes:
- `web.consul` ❌ (missing .service)
- `web.service.local` ❌ (wrong domain)
- `_web.service.consul` ❌ (SRV needs _tcp)

Correct formats:
- `web.service.consul` ✅ (A record)
- `web.node.consul` ✅ (A record)
- `_web._tcp.service.consul` ✅ (SRV record)

#### Solution

**If service is missing:**
```bash
curl -X POST http://localhost:8500/services \
  -H "Content-Type: application/json" \
  -d '{
    "name": "web",
    "address": "10.0.0.1",
    "port": 8080
  }'
```

**If query format is wrong:**
```bash
# Use correct format
dig @localhost -p 8600 web.service.consul A
```

---

### Issue 3: Empty Response (No Records)

#### Symptom

```bash
dig @localhost -p 8600 web.service.consul A

# Response shows NOERROR but no answer section:
;; ANSWER SECTION:
# (empty)
```

#### Diagnosis

This is different from NXDOMAIN - server understood query but found nothing.

```bash
# Check service health/expiration
curl http://localhost:8500/services/web | jq .

# Check Konsul logs for errors
journalctl -u konsul -n 100 | grep -i error
```

#### Causes

**1. Service Expired (TTL)**

Services have TTL and need heartbeats:

```bash
# Check service details
curl http://localhost:8500/services/web | jq '.expires_at'
```

**2. Service Deregistered**

Service was removed:

```bash
# Verify service exists
curl http://localhost:8500/services/web
# Returns 404 if not found
```

**3. Store Corruption**

Rare but possible - service store state issue.

#### Solution

**Re-register service:**
```bash
curl -X POST http://localhost:8500/services \
  -H "Content-Type: application/json" \
  -d '{
    "name": "web",
    "address": "10.0.0.1",
    "port": 8080
  }'
```

**Send heartbeat:**
```bash
curl -X POST http://localhost:8500/services/web/heartbeat
```

---

### Issue 4: Wrong IP Address Returned

#### Symptom

```bash
dig @localhost -p 8600 web.service.consul A +short
# Returns: 192.168.1.100
# Expected: 10.0.0.1
```

#### Diagnosis

**1. Check DNS Cache**

```bash
# Flush system DNS cache
sudo systemd-resolve --flush-caches  # Linux
sudo dscacheutil -flushcache         # macOS

# Query with no caching
dig @localhost -p 8600 web.service.consul A +nocache +norecurse
```

**2. Verify Registration**

```bash
curl http://localhost:8500/services | jq '.[] | select(.name=="web")'
```

**3. Check for Multiple Instances**

```bash
# DNS may return multiple IPs
dig @localhost -p 8600 web.service.consul A +short
# 192.168.1.100
# 10.0.0.1
# (both are valid - load balancing)
```

#### Solution

**If wrong IP is registered:**
```bash
# Deregister all instances
curl -X DELETE http://localhost:8500/services/web

# Re-register with correct IP
curl -X POST http://localhost:8500/services \
  -d '{"name":"web","address":"10.0.0.1","port":8080}'
```

---

### Issue 5: SRV Query Returns No Records

#### Symptom

```bash
dig @localhost -p 8600 _web._tcp.service.consul SRV
# ;; ANSWER SECTION:
# (empty)
```

But A query works:
```bash
dig @localhost -p 8600 web.service.consul A
# Returns IP address
```

#### Diagnosis

**Check query format:**

```bash
# Wrong formats:
_web.service.consul             ❌ (missing protocol)
_web._udp.service.consul        ❌ (wrong protocol)
web._tcp.service.consul         ❌ (missing underscore)

# Correct format:
_web._tcp.service.consul        ✅
```

#### Solution

Use correct SRV format:
```bash
dig @localhost -p 8600 _web._tcp.service.consul SRV
```

---

### Issue 6: Connection Timeout

#### Symptom

```bash
dig @localhost -p 8600 web.service.consul A
# ;; connection timed out; no servers could be reached
```

#### Diagnosis

**1. Check DNS server is running**

```bash
# Check process
pgrep -f konsul
ps aux | grep konsul

# Check port is listening
sudo netstat -tulpn | grep 8600
```

**2. Check firewall**

```bash
# Ubuntu/Debian
sudo ufw status | grep 8600

# CentOS/RHEL
sudo firewall-cmd --list-ports | grep 8600

# iptables
sudo iptables -L -n | grep 8600
```

**3. Check network connectivity**

```bash
# Test UDP port
nc -vuz localhost 8600

# Test TCP port
nc -vz localhost 8600

# Check if UDP works but TCP doesn't (or vice versa)
dig @localhost -p 8600 web.service.consul A +tcp   # Force TCP
dig @localhost -p 8600 web.service.consul A +notcp # Force UDP
```

#### Solution

**Start Konsul DNS server:**
```bash
# If not running
systemctl start konsul

# Check status
systemctl status konsul
```

**Open firewall port:**
```bash
# Ubuntu/Debian
sudo ufw allow 8600/udp
sudo ufw allow 8600/tcp

# CentOS/RHEL
sudo firewall-cmd --permanent --add-port=8600/udp
sudo firewall-cmd --permanent --add-port=8600/tcp
sudo firewall-cmd --reload
```

---

### Issue 7: Slow DNS Responses

#### Symptom

```bash
time dig @localhost -p 8600 web.service.consul A
# real    0m0.500s  # Should be < 10ms
```

#### Diagnosis

**1. Measure actual DNS latency**

```bash
# Time multiple queries
for i in {1..10}; do
  time dig @localhost -p 8600 web.service.consul A +short
done | grep real
```

**2. Check service store size**

```bash
# Count registered services
curl http://localhost:8500/services | jq 'length'

# Large stores (>10k services) may slow down queries
```

**3. Check system load**

```bash
uptime
top -b -n 1 | head -20
```

#### Solution

**1. Optimize service store** (if large)

Future: Add indexing by service name

**2. Reduce DNS client timeout**

```bash
# In /etc/resolv.conf
options timeout:1
```

**3. Use local Konsul instance**

Avoid network latency - run Konsul on same host.

---

## Advanced Troubleshooting

### Debug Logging

Enable debug logs to see every DNS query:

```yaml
# config.yaml
log:
  level: debug
```

**Restart Konsul:**
```bash
systemctl restart konsul
```

**Watch logs:**
```bash
journalctl -u konsul -f | grep DNS
```

**Example debug output:**
```
DEBUG DNS query received name=web.service.consul. type=A
DEBUG A query processed service=web records=1
```

---

### DNS Message Analysis

Use `dig` to see full DNS message:

```bash
dig @localhost -p 8600 web.service.consul A +noall +answer +additional +stats

# Output:
;; ANSWER SECTION:
web.service.consul.     30      IN      A       10.0.0.1

;; Query time: 2 msec
;; SERVER: 127.0.0.1#8600(127.0.0.1)
;; WHEN: Mon Jan 01 12:00:00 UTC 2024
;; MSG SIZE  rcvd: 64
```

**Key metrics:**
- **Query time**: Should be < 10ms
- **MSG SIZE**: Should be < 512 bytes (UDP limit)
- **Answer section**: Should contain expected records

---

### Packet Capture

Capture actual DNS packets:

```bash
# Capture DNS traffic on port 8600
sudo tcpdump -i any -n port 8600 -vv

# Save to file for analysis
sudo tcpdump -i any -n port 8600 -w dns-capture.pcap

# Analyze with Wireshark
wireshark dns-capture.pcap
```

**Look for:**
- Query format correctness
- Response codes (NOERROR, NXDOMAIN, etc.)
- Response times
- Packet loss

---

### Health Check Script

**File**: `/usr/local/bin/konsul-dns-health.sh`

```bash
#!/bin/bash

# DNS Health Check Script

set -e

# Configuration
DNS_SERVER="localhost"
DNS_PORT="8600"
TEST_SERVICE="health-check"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "=== Konsul DNS Health Check ==="
echo

# 1. Check if port is listening
echo -n "Port $DNS_PORT listening... "
if netstat -tuln | grep -q ":$DNS_PORT "; then
    echo -e "${GREEN}OK${NC}"
else
    echo -e "${RED}FAIL${NC}"
    echo "Error: Port $DNS_PORT is not listening"
    exit 1
fi

# 2. Test UDP connectivity
echo -n "UDP connectivity... "
if timeout 2 bash -c "echo '' | nc -u $DNS_SERVER $DNS_PORT"; then
    echo -e "${GREEN}OK${NC}"
else
    echo -e "${RED}FAIL${NC}"
    exit 1
fi

# 3. Test TCP connectivity
echo -n "TCP connectivity... "
if nc -zv $DNS_SERVER $DNS_PORT 2>&1 | grep -q "succeeded"; then
    echo -e "${GREEN}OK${NC}"
else
    echo -e "${RED}FAIL${NC}"
    exit 1
fi

# 4. Register test service
echo -n "Registering test service... "
REGISTER_RESPONSE=$(curl -s -X POST http://$DNS_SERVER:8500/services \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"$TEST_SERVICE\",\"address\":\"127.0.0.1\",\"port\":9999}" \
  -w "%{http_code}")

if [ "$REGISTER_RESPONSE" = "200" ]; then
    echo -e "${GREEN}OK${NC}"
else
    echo -e "${RED}FAIL${NC} (HTTP $REGISTER_RESPONSE)"
    exit 1
fi

# 5. Test A query
echo -n "A query... "
A_RESULT=$(dig @$DNS_SERVER -p $DNS_PORT $TEST_SERVICE.service.consul A +short)
if [ "$A_RESULT" = "127.0.0.1" ]; then
    echo -e "${GREEN}OK${NC}"
else
    echo -e "${RED}FAIL${NC}"
    echo "Expected: 127.0.0.1, Got: $A_RESULT"
    exit 1
fi

# 6. Test SRV query
echo -n "SRV query... "
SRV_RESULT=$(dig @$DNS_SERVER -p $DNS_PORT _$TEST_SERVICE._tcp.service.consul SRV +short)
if echo "$SRV_RESULT" | grep -q "9999"; then
    echo -e "${GREEN}OK${NC}"
else
    echo -e "${RED}FAIL${NC}"
    echo "Port 9999 not found in SRV response"
    exit 1
fi

# 7. Cleanup
echo -n "Cleanup... "
curl -s -X DELETE http://$DNS_SERVER:8500/services/$TEST_SERVICE > /dev/null
echo -e "${GREEN}OK${NC}"

echo
echo -e "${GREEN}All checks passed!${NC}"
```

**Usage:**
```bash
chmod +x /usr/local/bin/konsul-dns-health.sh
/usr/local/bin/konsul-dns-health.sh
```

---

## Error Messages Reference

### `bind: address already in use`

**Meaning**: Port 8600 is already bound by another process

**Solution**: Kill other process or change port

---

### `i/o timeout`

**Meaning**: DNS query didn't receive response within timeout

**Causes**:
- Server not running
- Firewall blocking
- Network issue

---

### `NXDOMAIN`

**Meaning**: Domain name doesn't exist

**Causes**:
- Service not registered
- Wrong query format
- Service expired

---

### `SERVFAIL`

**Meaning**: Server encountered an error processing query

**Causes**:
- Internal server error
- Store lookup failed

---

## Performance Troubleshooting

### Benchmark DNS Queries

```bash
#!/bin/bash

# Benchmark DNS performance

COUNT=1000
SERVICE="web"

echo "Running $COUNT DNS queries..."

START=$(date +%s%N)

for i in $(seq 1 $COUNT); do
  dig @localhost -p 8600 $SERVICE.service.consul A +short > /dev/null
done

END=$(date +%s%N)
DURATION=$(( ($END - $START) / 1000000 ))
AVG=$(( $DURATION / $COUNT ))

echo "Total time: ${DURATION}ms"
echo "Average latency: ${AVG}ms per query"
echo "Queries per second: $(( $COUNT * 1000 / $DURATION ))"
```

**Expected results**:
- Average latency: < 5ms
- QPS: > 200 queries/second (single threaded)

---

### Monitor DNS Metrics

```bash
# Watch DNS query rate
watch -n 1 'journalctl -u konsul --since "1 minute ago" | grep "DNS query" | wc -l'

# Count queries by type
journalctl -u konsul --since "1 hour ago" | grep "DNS query" | \
  awk '{print $NF}' | sort | uniq -c
```

---

## Prevention Best Practices

### 1. Service Health Monitoring

Monitor service registration and heartbeats:

```bash
# Cron job to check service health
*/5 * * * * /usr/local/bin/check-services.sh
```

```bash
#!/bin/bash
# check-services.sh

EXPECTED_SERVICES=("web" "api" "db")

for service in "${EXPECTED_SERVICES[@]}"; do
  if ! curl -sf http://localhost:8500/services/$service > /dev/null; then
    echo "ERROR: Service $service not registered"
    # Alert via email, Slack, etc.
  fi
done
```

---

### 2. DNS Query Monitoring

```bash
# Alert if DNS queries fail
*/1 * * * * dig @localhost -p 8600 web.service.consul A +short || \
  echo "DNS query failed" | mail -s "DNS Alert" admin@example.com
```

---

### 3. Log Rotation

Prevent disk space issues:

```bash
# /etc/logrotate.d/konsul
/var/log/konsul/*.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    create 644 konsul konsul
}
```

---

## Getting Help

### Information to Collect

When reporting DNS issues, include:

1. **Konsul version:**
```bash
konsul --version
```

2. **DNS configuration:**
```bash
cat /etc/konsul/config.yaml | grep -A5 dns:
```

3. **Query that fails:**
```bash
dig @localhost -p 8600 service.service.consul A
```

4. **Service registration:**
```bash
curl http://localhost:8500/services | jq .
```

5. **Logs:**
```bash
journalctl -u konsul -n 100 --no-pager
```

6. **System info:**
```bash
uname -a
cat /etc/os-release
```

---

## See Also

- [DNS User Guide](dns-service-discovery.md)
- [DNS API Reference](dns-api.md)
- [DNS Implementation Guide](dns-implementation.md)
- [GitHub Issues](https://github.com/yourusername/konsul/issues)
