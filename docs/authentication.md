# Authentication - User Guide

Comprehensive guide for Konsul authentication and authorization.

## Overview

Konsul provides a dual authentication system designed for both human users and programmatic access:

- **JWT (JSON Web Tokens)** - For interactive sessions (CLI, Web UI)
- **API Keys** - For programmatic access (services, CI/CD, monitoring)

### Key Features

- ✅ Stateless authentication (no session storage)
- ✅ Token expiration and automatic refresh
- ✅ API key management with permissions
- ✅ Role-based access control ready
- ✅ Secure credential storage (hashed)
- ✅ Rate limiting per API key

---

## Quick Start

### Enable Authentication

```bash
KONSUL_AUTH_ENABLED=true \
KONSUL_JWT_SECRET="your-super-secret-key-minimum-32-characters" \
KONSUL_JWT_EXPIRY=15m \
KONSUL_REQUIRE_AUTH=true \
./konsul
```

### Login and Get Token

```bash
curl -X POST http://localhost:8500/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "username": "admin",
    "roles": ["admin"]
  }'
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_in": 900
}
```

### Use Token

```bash
curl http://localhost:8500/kv/mykey \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..."
```

---

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `KONSUL_AUTH_ENABLED` | `false` | Enable authentication system |
| `KONSUL_JWT_SECRET` | `` | JWT signing secret (required if auth enabled) |
| `KONSUL_JWT_EXPIRY` | `15m` | Access token lifetime |
| `KONSUL_REFRESH_EXPIRY` | `168h` (7 days) | Refresh token lifetime |
| `KONSUL_JWT_ISSUER` | `konsul` | JWT issuer name |
| `KONSUL_APIKEY_PREFIX` | `konsul` | API key prefix |
| `KONSUL_REQUIRE_AUTH` | `false` | Require auth for all endpoints |
| `KONSUL_PUBLIC_PATHS` | `/health,/health/live,/health/ready,/metrics` | Public paths (no auth) |

### Configuration File

```yaml
# config.yaml
auth:
  enabled: true
  jwt_secret: "your-secret-key-minimum-32-characters"
  jwt_expiry: 15m
  refresh_expiry: 168h
  jwt_issuer: konsul
  apikey_prefix: konsul
  require_auth: true
  public_paths:
    - /health
    - /health/live
    - /health/ready
    - /metrics
```

---

## JWT Authentication

### Login Flow

```
1. Client → POST /auth/login
              {user_id, username, roles}

2. Server ← Returns:
            - access_token (short-lived: 15 min)
            - refresh_token (long-lived: 7 days)

3. Client → API requests with:
            Authorization: Bearer <access_token>

4. When access_token expires → POST /auth/refresh
                                {refresh_token}

5. Server ← New access_token + refresh_token

6. Repeat steps 3-5
```

---

### Login

**Endpoint**: `POST /auth/login`

**Request:**
```json
{
  "user_id": "user123",
  "username": "admin",
  "roles": ["admin", "operator"]
}
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 900
}
```

**Example:**
```bash
# Login
RESPONSE=$(curl -s -X POST http://localhost:8500/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "username": "admin",
    "roles": ["admin"]
  }')

# Extract token
TOKEN=$(echo $RESPONSE | jq -r '.token')

# Use token
curl http://localhost:8500/kv/mykey \
  -H "Authorization: Bearer $TOKEN"
```

---

### Token Verification

**Endpoint**: `GET /auth/verify`

**Headers:** `Authorization: Bearer <token>`

**Response:**
```json
{
  "valid": true,
  "user_id": "user123",
  "username": "admin",
  "roles": ["admin"],
  "expires_at": "2025-01-12T15:30:00Z"
}
```

**Example:**
```bash
curl http://localhost:8500/auth/verify \
  -H "Authorization: Bearer $TOKEN"
```

---

### Token Refresh

**Endpoint**: `POST /auth/refresh`

**Request:**
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "username": "admin",
  "roles": ["admin"]
}
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs... (new)",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs... (new)",
  "expires_in": 900
}
```

**Example:**
```bash
curl -X POST http://localhost:8500/auth/refresh \
  -H "Content-Type: application/json" \
  -d "{
    \"refresh_token\": \"$REFRESH_TOKEN\",
    \"username\": \"admin\",
    \"roles\": [\"admin\"]
  }"
```

---

### Token Claims

JWT tokens contain these claims:

```json
{
  "user_id": "user123",
  "username": "admin",
  "roles": ["admin", "operator"],
  "exp": 1705074600,    // Expiration time (Unix)
  "iat": 1705073700,    // Issued at
  "nbf": 1705073700,    // Not before
  "iss": "konsul",      // Issuer
  "sub": "user123"      // Subject (user_id)
}
```

**Decode token** (for debugging):
```bash
# Install jwt-cli: https://github.com/mike-engel/jwt-cli
jwt decode $TOKEN
```

---

## API Key Authentication

API keys are designed for programmatic access - services, CI/CD pipelines, monitoring tools.

### Create API Key

**Endpoint**: `POST /auth/apikeys` *(requires JWT auth)*

**Request:**
```json
{
  "name": "production-service",
  "permissions": ["read", "write"],
  "metadata": {
    "env": "production",
    "service": "web-api"
  },
  "expires_in": 31536000
}
```

**Response:**
```json
{
  "key": "konsul_a1b2c3d4e5f6...",
  "api_key": {
    "id": "123e4567-e89b-12d3-a456-426614174000",
    "name": "production-service",
    "permissions": ["read", "write"],
    "metadata": {
      "env": "production",
      "service": "web-api"
    },
    "created_at": "2025-01-12T10:30:00Z",
    "expires_at": "2026-01-12T10:30:00Z",
    "enabled": true
  }
}
```

⚠️ **Important**: Save the `key` value - it won't be shown again!

**Example:**
```bash
# Create API key (requires JWT token)
curl -X POST http://localhost:8500/auth/apikeys \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "ci-cd-pipeline",
    "permissions": ["read", "write"],
    "metadata": {"tool": "github-actions"},
    "expires_in": 2592000
  }' | jq .

# Save the key
API_KEY=$(curl -s ... | jq -r '.key')
```

---

### Use API Key

**Method 1: X-API-Key Header** (Recommended)

```bash
curl http://localhost:8500/kv/mykey \
  -H "X-API-Key: konsul_a1b2c3d4e5f6..."
```

**Method 2: Authorization Header**

```bash
curl http://localhost:8500/kv/mykey \
  -H "Authorization: ApiKey konsul_a1b2c3d4e5f6..."
```

---

### List API Keys

**Endpoint**: `GET /auth/apikeys` *(requires JWT auth)*

**Response:**
```json
[
  {
    "id": "123e4567-e89b-12d3-a456-426614174000",
    "name": "production-service",
    "permissions": ["read", "write"],
    "created_at": "2025-01-12T10:30:00Z",
    "expires_at": "2026-01-12T10:30:00Z",
    "last_used_at": "2025-01-12T12:45:00Z",
    "enabled": true
  }
]
```

**Example:**
```bash
curl http://localhost:8500/auth/apikeys \
  -H "Authorization: Bearer $JWT_TOKEN" | jq .
```

---

### Get Specific API Key

**Endpoint**: `GET /auth/apikeys/:id` *(requires JWT auth)*

```bash
curl http://localhost:8500/auth/apikeys/123e4567-e89b-12d3-a456-426614174000 \
  -H "Authorization: Bearer $JWT_TOKEN"
```

---

### Update API Key

**Endpoint**: `PUT /auth/apikeys/:id` *(requires JWT auth)*

**Request:**
```json
{
  "name": "production-service-updated",
  "permissions": ["read", "write", "delete"],
  "metadata": {"env": "production", "version": "v2"},
  "enabled": true
}
```

**Example:**
```bash
curl -X PUT http://localhost:8500/auth/apikeys/123e4567... \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "updated-name",
    "permissions": ["read"]
  }'
```

---

### Revoke API Key

**Endpoint**: `POST /auth/apikeys/:id/revoke` *(requires JWT auth)*

**Effect**: Disables the API key (can be re-enabled with update)

```bash
curl -X POST http://localhost:8500/auth/apikeys/123e4567.../revoke \
  -H "Authorization: Bearer $JWT_TOKEN"
```

---

### Delete API Key

**Endpoint**: `DELETE /auth/apikeys/:id` *(requires JWT auth)*

**Effect**: Permanently deletes the API key

```bash
curl -X DELETE http://localhost:8500/auth/apikeys/123e4567... \
  -H "Authorization: Bearer $JWT_TOKEN"
```

---

## Public Paths

Certain endpoints don't require authentication:

**Default public paths:**
- `/health` - Health check
- `/health/live` - Liveness probe
- `/health/ready` - Readiness probe
- `/metrics` - Prometheus metrics

**Add custom public paths:**
```bash
KONSUL_PUBLIC_PATHS="/health,/health/live,/health/ready,/metrics,/docs,/api/v1/public" \
./konsul
```

---

## Security Best Practices

### 1. JWT Secret Management

**Requirements:**
- Minimum 32 characters
- Use cryptographically random value
- Never commit to version control
- Rotate periodically

**Generate secure secret:**
```bash
# Linux/macOS
openssl rand -base64 48

# Or
head -c 48 /dev/urandom | base64
```

**Store securely:**
```bash
# Use secrets manager
export KONSUL_JWT_SECRET=$(aws secretsmanager get-secret-value --secret-id konsul-jwt-secret --query SecretString --output text)

# Or Kubernetes secret
kubectl create secret generic konsul-auth \
  --from-literal=jwt-secret=$(openssl rand -base64 48)
```

---

### 2. Token Expiration

**Access tokens should be short-lived:**
```bash
# Recommended for production
KONSUL_JWT_EXPIRY=15m      # 15 minutes
KONSUL_REFRESH_EXPIRY=168h # 7 days
```

**Balance security vs usability:**
- Too short (1m) → constant refreshes, poor UX
- Too long (24h) → higher risk if compromised
- Sweet spot: 15-30 minutes

---

### 3. API Key Security

**Best practices:**
- Set expiration for all keys
- Use specific permissions (not `["*"]`)
- Add metadata for tracking
- Regularly audit and rotate
- Revoke unused keys
- Monitor `last_used_at`

**Example secure API key:**
```json
{
  "name": "github-actions-read-only",
  "permissions": ["read"],
  "metadata": {"service": "ci", "repo": "myapp"},
  "expires_in": 2592000  // 30 days
}
```

---

### 4. TLS/HTTPS

**Always use TLS in production:**
```bash
KONSUL_TLS_ENABLED=true \
KONSUL_TLS_CERT_FILE=/etc/konsul/tls/cert.pem \
KONSUL_TLS_KEY_FILE=/etc/konsul/tls/key.pem \
KONSUL_AUTH_ENABLED=true \
./konsul
```

**Why**: Prevents token theft via network sniffing

---

### 5. Rate Limiting

Enable rate limiting per API key:
```bash
KONSUL_RATE_LIMIT_ENABLED=true \
KONSUL_RATE_LIMIT_BY_APIKEY=true \
KONSUL_RATE_LIMIT_REQUESTS_PER_SEC=100 \
./konsul
```

---

## Integration Examples

### CLI Tool

```bash
#!/bin/bash
# konsul-client.sh

KONSUL_URL="http://localhost:8500"
TOKEN_FILE="$HOME/.konsul/token"

login() {
    RESPONSE=$(curl -s -X POST $KONSUL_URL/auth/login \
      -H "Content-Type: application/json" \
      -d "{
        \"user_id\": \"$1\",
        \"username\": \"$1\",
        \"roles\": [\"user\"]
      }")

    TOKEN=$(echo $RESPONSE | jq -r '.token')
    REFRESH=$(echo $RESPONSE | jq -r '.refresh_token')

    echo $TOKEN > $TOKEN_FILE
    echo $REFRESH >> $TOKEN_FILE
    chmod 600 $TOKEN_FILE

    echo "Logged in successfully"
}

api_call() {
    TOKEN=$(head -1 $TOKEN_FILE)
    curl -s "$KONSUL_URL$1" \
      -H "Authorization: Bearer $TOKEN"
}

# Usage
login "admin"
api_call "/kv/config/host"
```

---

### Go Application

```go
package main

import (
    "fmt"
    "net/http"
    "time"
)

type KonsulClient struct {
    baseURL      string
    accessToken  string
    refreshToken string
    httpClient   *http.Client
}

func (c *KonsulClient) Login(userID, username string, roles []string) error {
    // POST /auth/login
    // Store tokens
}

func (c *KonsulClient) Request(method, path string) (*http.Response, error) {
    req, _ := http.NewRequest(method, c.baseURL+path, nil)
    req.Header.Set("Authorization", "Bearer "+c.accessToken)

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }

    // If 401, try refresh
    if resp.StatusCode == 401 {
        if err := c.Refresh(); err != nil {
            return nil, err
        }
        // Retry request
        return c.Request(method, path)
    }

    return resp, nil
}
```

---

### Python Application

```python
import requests
import time
from datetime import datetime, timedelta

class KonsulClient:
    def __init__(self, base_url):
        self.base_url = base_url
        self.access_token = None
        self.refresh_token = None
        self.token_expires = None

    def login(self, user_id, username, roles):
        response = requests.post(
            f"{self.base_url}/auth/login",
            json={
                "user_id": user_id,
                "username": username,
                "roles": roles
            }
        )
        data = response.json()

        self.access_token = data['token']
        self.refresh_token = data['refresh_token']
        self.token_expires = datetime.now() + timedelta(seconds=data['expires_in'])

    def _ensure_valid_token(self):
        if datetime.now() >= self.token_expires:
            self.refresh()

    def get(self, path):
        self._ensure_valid_token()
        response = requests.get(
            f"{self.base_url}{path}",
            headers={"Authorization": f"Bearer {self.access_token}"}
        )
        return response.json()

# Usage
client = KonsulClient("http://localhost:8500")
client.login("user123", "admin", ["admin"])
data = client.get("/kv/config/host")
```

---

### Docker Compose

```yaml
version: '3.8'

services:
  konsul:
    image: konsul:latest
    environment:
      - KONSUL_AUTH_ENABLED=true
      - KONSUL_JWT_SECRET=${JWT_SECRET}
      - KONSUL_JWT_EXPIRY=15m
      - KONSUL_REQUIRE_AUTH=true
    secrets:
      - jwt_secret
    ports:
      - "8500:8500"

  app:
    image: myapp:latest
    environment:
      - KONSUL_URL=http://konsul:8500
      - KONSUL_API_KEY=${KONSUL_API_KEY}
    depends_on:
      - konsul

secrets:
  jwt_secret:
    file: ./secrets/jwt-secret.txt
```

---

## Monitoring

### Metrics

Authentication metrics are exposed via Prometheus:

```
# Total auth attempts
konsul_auth_attempts_total{method="jwt",status="success"} 1000
konsul_auth_attempts_total{method="jwt",status="failure"} 5
konsul_auth_attempts_total{method="apikey",status="success"} 500

# Token operations
konsul_jwt_token_generated_total 1005
konsul_jwt_token_refreshed_total 200
konsul_jwt_token_validation_errors_total 5

# API key operations
konsul_apikey_created_total 50
konsul_apikey_revoked_total 5
konsul_apikey_validation_errors_total 3
```

---

### Logging

Auth events are logged:

```json
{
  "level": "info",
  "msg": "JWT token generated",
  "user_id": "user123",
  "username": "admin",
  "expires_in": 900
}

{
  "level": "info",
  "msg": "API key validated",
  "key_id": "123e4567-e89b-12d3-a456-426614174000",
  "key_name": "production-service"
}

{
  "level": "warn",
  "msg": "Authentication failed",
  "method": "jwt",
  "error": "token has expired"
}
```

---

## Troubleshooting

### Issue: "token has expired"

**Cause**: Access token expired (default 15 minutes)

**Solution**: Use refresh token to get new access token

```bash
curl -X POST http://localhost:8500/auth/refresh \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\": \"$REFRESH_TOKEN\", \"username\": \"admin\", \"roles\": [\"admin\"]}"
```

---

### Issue: "token is invalid"

**Causes:**
- Token tampered with
- Wrong JWT secret on server
- Malformed token

**Diagnosis:**
```bash
# Decode token to inspect
jwt decode $TOKEN

# Check server logs
journalctl -u konsul | grep "token"
```

---

### Issue: "API key not found"

**Causes:**
- API key deleted or revoked
- Wrong API key value
- API key expired

**Diagnosis:**
```bash
# List all API keys
curl http://localhost:8500/auth/apikeys \
  -H "Authorization: Bearer $JWT_TOKEN" | jq .

# Check specific key
curl http://localhost:8500/auth/apikeys/$KEY_ID \
  -H "Authorization: Bearer $JWT_TOKEN"
```

---

## See Also

- [Authentication API Reference](authentication-api.md)
- [Authentication Implementation](authentication-implementation.md)
- [ADR-0003](adr/0003-jwt-authentication.md)
- [JWT.io](https://jwt.io/)
- [RFC 7519 - JSON Web Token](https://tools.ietf.org/html/rfc7519)
