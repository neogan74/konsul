# Rate Limiting Management API

Complete API reference for managing rate limits in Konsul.

## Overview

The Rate Limiting Management API provides endpoints for administrators to monitor and manage rate limiting in real-time. These endpoints allow you to:

- View rate limit statistics and configuration
- Monitor active rate-limited clients
- Reset rate limits for specific clients
- Dynamically update rate limit configuration
- Bulk reset operations

**Base Path**: `/admin/ratelimit`

**Authentication**: Requires JWT authentication (admin role)

**ACL**: Requires `admin:write` capability (if ACL is enabled)

---

## Table of Contents

- [Authentication](#authentication)
- [Endpoints](#endpoints)
  - [Get Statistics](#get-statistics)
  - [Get Configuration](#get-configuration)
  - [Get Active Clients](#get-active-clients)
  - [Get Client Status](#get-client-status)
  - [Reset IP Rate Limit](#reset-ip-rate-limit)
  - [Reset API Key Rate Limit](#reset-api-key-rate-limit)
  - [Reset All Rate Limits](#reset-all-rate-limits)
  - [Update Configuration](#update-configuration)
- [Data Models](#data-models)
- [Examples](#examples)
- [Error Responses](#error-responses)

---

## Authentication

All endpoints require authentication with admin privileges.

**Headers:**
```http
Authorization: Bearer <jwt_token>
```

**Example:**
```bash
# Login to get token
TOKEN=$(curl -X POST http://localhost:8888/auth/login \
  -H "Content-Type: application/json" \
  -d '{"user_id":"admin","username":"admin","roles":["admin"]}' | \
  jq -r '.token')

# Use token in requests
curl http://localhost:8888/admin/ratelimit/stats \
  -H "Authorization: Bearer $TOKEN"
```

---

## Endpoints

### Get Statistics

Get current rate limiting statistics.

**Endpoint:** `GET /admin/ratelimit/stats`

**Response:**
```json
{
  "success": true,
  "data": {
    "ip_limiters": 42,
    "apikey_limiters": 10
  }
}
```

**Fields:**
- `ip_limiters` (integer) - Number of active IP-based rate limiters
- `apikey_limiters` (integer) - Number of active API-key-based rate limiters

**Example:**
```bash
curl http://localhost:8888/admin/ratelimit/stats \
  -H "Authorization: Bearer $TOKEN"
```

---

### Get Configuration

Get current rate limit configuration.

**Endpoint:** `GET /admin/ratelimit/config`

**Response:**
```json
{
  "success": true,
  "config": {
    "enabled": true,
    "requests_per_sec": 100.0,
    "burst": 20,
    "by_ip": true,
    "by_apikey": true,
    "cleanup_interval": "5m0s"
  }
}
```

**Fields:**
- `enabled` (boolean) - Whether rate limiting is enabled
- `requests_per_sec` (float) - Tokens added per second
- `burst` (integer) - Maximum burst size (bucket capacity)
- `by_ip` (boolean) - Whether per-IP limiting is enabled
- `by_apikey` (boolean) - Whether per-API-key limiting is enabled
- `cleanup_interval` (string) - Cleanup interval for unused limiters

**Example:**
```bash
curl http://localhost:8888/admin/ratelimit/config \
  -H "Authorization: Bearer $TOKEN"
```

---

### Get Active Clients

Get list of currently rate-limited clients.

**Endpoint:** `GET /admin/ratelimit/clients`

**Query Parameters:**
- `type` (string, optional) - Filter by type: `all`, `ip`, `apikey` (default: `all`)

**Response:**
```json
{
  "success": true,
  "count": 3,
  "clients": [
    {
      "identifier": "192.168.1.100",
      "type": "ip",
      "tokens": 15.5,
      "max_tokens": 20,
      "rate": 100.0,
      "last_update": "2025-10-15T14:30:00Z"
    },
    {
      "identifier": "konsul_abc123",
      "type": "apikey",
      "tokens": 18.2,
      "max_tokens": 20,
      "rate": 100.0,
      "last_update": "2025-10-15T14:29:55Z"
    }
  ]
}
```

**Client Fields:**
- `identifier` (string) - Client identifier (IP address or API key ID)
- `type` (string) - `ip` or `apikey`
- `tokens` (float) - Current available tokens
- `max_tokens` (integer) - Maximum tokens (burst size)
- `rate` (float) - Token generation rate (per second)
- `last_update` (string) - Last activity timestamp (RFC3339)

**Examples:**
```bash
# Get all clients
curl http://localhost:8888/admin/ratelimit/clients \
  -H "Authorization: Bearer $TOKEN"

# Get only IP clients
curl "http://localhost:8888/admin/ratelimit/clients?type=ip" \
  -H "Authorization: Bearer $TOKEN"

# Get only API key clients
curl "http://localhost:8888/admin/ratelimit/clients?type=apikey" \
  -H "Authorization: Bearer $TOKEN"
```

---

### Get Client Status

Get rate limit status for a specific client.

**Endpoint:** `GET /admin/ratelimit/client/:identifier`

**Path Parameters:**
- `identifier` (string, required) - Client identifier (IP or API key ID)

**Response (Success):**
```json
{
  "success": true,
  "client": {
    "identifier": "192.168.1.100",
    "type": "ip",
    "tokens": 15.5,
    "max_tokens": 20,
    "rate": 100.0,
    "last_update": "2025-10-15T14:30:00Z"
  }
}
```

**Response (Not Found):**
```json
{
  "success": false,
  "error": "Client not found"
}
```

**Example:**
```bash
curl http://localhost:8888/admin/ratelimit/client/192.168.1.100 \
  -H "Authorization: Bearer $TOKEN"
```

---

### Reset IP Rate Limit

Reset rate limit for a specific IP address.

**Endpoint:** `POST /admin/ratelimit/reset/ip/:ip`

**Path Parameters:**
- `ip` (string, required) - IP address to reset

**Response:**
```json
{
  "success": true,
  "message": "Rate limit reset successfully",
  "ip": "192.168.1.100"
}
```

**Example:**
```bash
curl -X POST http://localhost:8888/admin/ratelimit/reset/ip/192.168.1.100 \
  -H "Authorization: Bearer $TOKEN"
```

**Use Case:**
- Accidentally rate-limited a legitimate user
- Testing after fixing client issues
- Emergency override for critical operations

---

### Reset API Key Rate Limit

Reset rate limit for a specific API key.

**Endpoint:** `POST /admin/ratelimit/reset/apikey/:key_id`

**Path Parameters:**
- `key_id` (string, required) - API key ID to reset

**Response:**
```json
{
  "success": true,
  "message": "Rate limit reset successfully",
  "key_id": "550e8400-e29b-41d4-a935-446655440000"
}
```

**Example:**
```bash
curl -X POST http://localhost:8888/admin/ratelimit/reset/apikey/550e8400-e29b-41d4-a935-446655440000 \
  -H "Authorization: Bearer $TOKEN"
```

**Use Case:**
- Partner experiencing temporary rate limits
- After system maintenance affecting API clients
- Emergency access for critical integrations

---

### Reset All Rate Limits

Reset all rate limiters (bulk operation).

**Endpoint:** `POST /admin/ratelimit/reset/all`

**Query Parameters:**
- `type` (string, optional) - Reset type: `all`, `ip`, `apikey` (default: `all`)

**Response:**
```json
{
  "success": true,
  "message": "All rate limiters reset",
  "type": "all"
}
```

**Examples:**
```bash
# Reset all limiters
curl -X POST http://localhost:8888/admin/ratelimit/reset/all \
  -H "Authorization: Bearer $TOKEN"

# Reset only IP limiters
curl -X POST "http://localhost:8888/admin/ratelimit/reset/all?type=ip" \
  -H "Authorization: Bearer $TOKEN"

# Reset only API key limiters
curl -X POST "http://localhost:8888/admin/ratelimit/reset/all?type=apikey" \
  -H "Authorization: Bearer $TOKEN"
```

**⚠️ Warning:** This operation affects all clients. Use with caution.

**Use Cases:**
- After system-wide maintenance
- After configuration changes
- Testing/development environments
- Emergency override during incidents

---

### Update Configuration

Dynamically update rate limit configuration.

**Endpoint:** `PUT /admin/ratelimit/config`

**Request Body:**
```json
{
  "requests_per_sec": 200.0,
  "burst": 50
}
```

**Fields:**
- `requests_per_sec` (float, optional) - New requests per second rate
- `burst` (integer, optional) - New burst size

**Response:**
```json
{
  "success": true,
  "message": "Configuration updated successfully",
  "config": {
    "requests_per_sec": 200.0,
    "burst": 50
  }
}
```

**Response (No Changes):**
```json
{
  "success": true,
  "message": "No changes applied"
}
```

**Example:**
```bash
curl -X PUT http://localhost:8888/admin/ratelimit/config \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "requests_per_sec": 200,
    "burst": 50
  }'
```

**Important Notes:**
- Changes apply to **new limiters only**
- Existing limiters retain their original configuration
- To apply to all clients, reset all limiters after updating config
- Configuration changes are **not persisted** (reverts on restart)

**Validation:**
- `requests_per_sec` must be > 0
- `burst` must be > 0

---

## Data Models

### ClientInfo

Represents a rate-limited client's status.

```typescript
interface ClientInfo {
  identifier: string;   // IP address or API key ID
  type: string;        // "ip" or "apikey"
  tokens: number;      // Current available tokens (float)
  max_tokens: number;  // Maximum burst size
  rate: number;        // Tokens per second
  last_update: string; // RFC3339 timestamp
}
```

### Config

Rate limiter configuration.

```typescript
interface Config {
  enabled: boolean;
  requests_per_sec: number;   // Tokens per second
  burst: number;              // Bucket capacity
  by_ip: boolean;             // Per-IP limiting enabled
  by_apikey: boolean;         // Per-API-key limiting enabled
  cleanup_interval: string;   // Duration string (e.g., "5m0s")
}
```

---

## Examples

### Monitor Rate Limits

**Get overview:**
```bash
#!/bin/bash
TOKEN="your-jwt-token"
BASE_URL="http://localhost:8888"

# Get stats
echo "=== Statistics ==="
curl -s "$BASE_URL/admin/ratelimit/stats" \
  -H "Authorization: Bearer $TOKEN" | jq .

# Get config
echo "=== Configuration ==="
curl -s "$BASE_URL/admin/ratelimit/config" \
  -H "Authorization: Bearer $TOKEN" | jq .

# Get active clients
echo "=== Active Clients ==="
curl -s "$BASE_URL/admin/ratelimit/clients" \
  -H "Authorization: Bearer $TOKEN" | jq .
```

---

### Reset Specific Client

**Reset by IP:**
```bash
IP="192.168.1.100"
curl -X POST "http://localhost:8888/admin/ratelimit/reset/ip/$IP" \
  -H "Authorization: Bearer $TOKEN"
```

**Reset by API key:**
```bash
KEY_ID="550e8400-e29b-41d4-a935-446655440000"
curl -X POST "http://localhost:8888/admin/ratelimit/reset/apikey/$KEY_ID" \
  -H "Authorization: Bearer $TOKEN"
```

---

### Update Rate Limits

**Increase limits during high traffic:**
```bash
curl -X PUT http://localhost:8888/admin/ratelimit/config \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "requests_per_sec": 500,
    "burst": 100
  }'

# Reset all to apply new config
curl -X POST "http://localhost:8888/admin/ratelimit/reset/all" \
  -H "Authorization: Bearer $TOKEN"
```

---

### Find Rate-Limited Clients

**Get clients with low tokens (near rate limit):**
```bash
curl -s http://localhost:8888/admin/ratelimit/clients \
  -H "Authorization: Bearer $TOKEN" | \
  jq '.clients[] | select(.tokens < 5)'
```

**Monitor specific IP:**
```bash
#!/bin/bash
IP="192.168.1.100"

while true; do
  curl -s "http://localhost:8888/admin/ratelimit/client/$IP" \
    -H "Authorization: Bearer $TOKEN" | \
    jq -r '.client | "\(.identifier): \(.tokens) tokens remaining"'
  sleep 5
done
```

---

### Emergency Reset Script

```bash
#!/bin/bash
# Emergency rate limit reset script

TOKEN="${KONSUL_ADMIN_TOKEN:-your-token}"
BASE_URL="${KONSUL_URL:-http://localhost:8888}"

echo "⚠️  This will reset ALL rate limiters!"
read -p "Continue? (yes/no) " -r
if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
    echo "Aborted."
    exit 1
fi

# Reset all
curl -X POST "$BASE_URL/admin/ratelimit/reset/all" \
  -H "Authorization: Bearer $TOKEN"

# Verify
echo ""
echo "Current stats:"
curl -s "$BASE_URL/admin/ratelimit/stats" \
  -H "Authorization: Bearer $TOKEN" | jq .
```

---

## Error Responses

### 400 Bad Request

**Missing parameter:**
```json
{
  "success": false,
  "error": "IP address is required"
}
```

**Invalid configuration:**
```json
{
  "success": false,
  "error": "requests_per_sec must be greater than 0"
}
```

---

### 401 Unauthorized

**Missing or invalid token:**
```json
{
  "error": "Unauthorized",
  "message": "Missing or invalid token"
}
```

---

### 403 Forbidden

**Insufficient permissions:**
```json
{
  "error": "Forbidden",
  "message": "Insufficient permissions"
}
```

---

### 404 Not Found

**Client not found:**
```json
{
  "success": false,
  "error": "Client not found"
}
```

---

## Best Practices

### 1. Monitor Before Resetting

**Always check status first:**
```bash
# Check client status
curl -s http://localhost:8888/admin/ratelimit/client/192.168.1.100 \
  -H "Authorization: Bearer $TOKEN" | jq .

# Then reset if needed
curl -X POST http://localhost:8888/admin/ratelimit/reset/ip/192.168.1.100 \
  -H "Authorization: Bearer $TOKEN"
```

---

### 2. Use Bulk Operations Carefully

**Bulk resets affect all clients:**
```bash
# Better: Reset specific type
curl -X POST "http://localhost:8888/admin/ratelimit/reset/all?type=ip" \
  -H "Authorization: Bearer $TOKEN"

# Instead of: Reset everything
# curl -X POST "http://localhost:8888/admin/ratelimit/reset/all" ...
```

---

### 3. Log Admin Actions

**Track who performed actions:**
```bash
# Actions are logged with admin username
# Check logs:
docker logs konsul | grep "Rate limit reset"
```

---

### 4. Test Configuration Changes

**Test in development first:**
```bash
# Development
KONSUL_URL=http://dev.konsul:8888
curl -X PUT "$KONSUL_URL/admin/ratelimit/config" ...

# Then production
KONSUL_URL=http://prod.konsul:8888
curl -X PUT "$KONSUL_URL/admin/ratelimit/config" ...
```

---

### 5. Automate Monitoring

**Prometheus alerts:**
```yaml
- alert: HighRateLimitClientCount
  expr: |
    sum(konsul_rate_limit_active_clients) > 1000
  for: 5m
  annotations:
    summary: "High number of rate-limited clients"
```

**Check via API:**
```bash
# Cron job to check stats
*/5 * * * * curl -s http://localhost:8888/admin/ratelimit/stats -H "Auth: Bearer $TOKEN" | ...
```

---

## Security Considerations

### 1. Require Strong Authentication

**Always enable authentication:**
```bash
KONSUL_AUTH_ENABLED=true
KONSUL_AUTH_REQUIRE_AUTH=true
```

---

### 2. Use ACL for Fine-Grained Control

**Restrict to admin users:**
```bash
KONSUL_ACL_ENABLED=true
# Only users with admin:write can access these endpoints
```

---

### 3. Audit Logging

**All actions are logged:**
```json
{
  "level": "info",
  "msg": "Rate limit reset for IP",
  "ip": "192.168.1.100",
  "admin_user": "admin@example.com",
  "timestamp": "2025-10-15T14:30:00Z"
}
```

---

### 4. Rate Limit the Admin API

**Consider separate limits for admin endpoints:**
```go
// Custom rate limit for admin routes
adminRateLimitRoutes.Use(middleware.RateLimitWithConfig(10.0, 5))
```

---

## Troubleshooting

### Issue: Cannot Access Endpoints

**Check authentication:**
```bash
# Verify token is valid
curl http://localhost:8888/auth/verify \
  -H "Authorization: Bearer $TOKEN"
```

**Check ACL permissions:**
```bash
# User must have admin:write capability
```

---

### Issue: Config Changes Not Applied

**Remember:** Changes only affect new limiters.

**Solution:**
```bash
# Update config
curl -X PUT http://localhost:8888/admin/ratelimit/config \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"requests_per_sec":200}'

# Reset all to apply
curl -X POST http://localhost:8888/admin/ratelimit/reset/all \
  -H "Authorization: Bearer $TOKEN"
```

---

### Issue: Client Not Found

**Client may have been cleaned up:**
```bash
# Clients idle for >5 minutes are automatically removed
# Check cleanup_interval in config
```

---

## See Also

- [Rate Limiting User Guide](rate-limiting.md)
- [Authentication Documentation](authentication.md)
- [ACL Guide](acl-guide.md)
- [Metrics Documentation](metrics.md)

---

## Changelog

- **2025-10-15**: Initial API documentation
- **Version**: 0.1.0
- **Status**: ✅ Production Ready

---

**Complete API reference for Konsul Rate Limiting Management**
