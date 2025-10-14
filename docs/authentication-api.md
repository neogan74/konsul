# Authentication - API Reference

Complete API reference for Konsul authentication system.

## Package `github.com/neogan74/konsul/internal/auth`

### Types

#### `JWTService`

Service for JWT token generation and validation.

```go
type JWTService struct {
    secretKey     []byte
    tokenExpiry   time.Duration
    refreshExpiry time.Duration
    issuer        string
}
```

**Constructor:**
```go
func NewJWTService(secretKey string, tokenExpiry, refreshExpiry time.Duration, issuer string) *JWTService
```

**Example:**
```go
jwtService := auth.NewJWTService(
    "your-secret-key-minimum-32-characters",
    15*time.Minute,  // Access token expiry
    7*24*time.Hour,  // Refresh token expiry
    "konsul",
)
```

---

#### `Claims`

JWT token claims structure.

```go
type Claims struct {
    UserID   string   `json:"user_id"`
    Username string   `json:"username"`
    Roles    []string `json:"roles"`
    jwt.RegisteredClaims
}
```

**Fields:**
- **UserID** - Unique user identifier
- **Username** - Human-readable username
- **Roles** - User roles for RBAC
- **RegisteredClaims** - Standard JWT claims (exp, iat, nbf, iss, sub)

---

#### `APIKeyService`

Service for API key management.

```go
type APIKeyService struct {
    keys   map[string]*APIKey
    mu     sync.RWMutex
    prefix string
}
```

**Constructor:**
```go
func NewAPIKeyService(prefix string) *APIKeyService
```

**Example:**
```go
apiKeyService := auth.NewAPIKeyService("konsul")
```

---

#### `APIKey`

API key structure.

```go
type APIKey struct {
    ID          string            `json:"id"`
    Name        string            `json:"name"`
    KeyHash     string            `json:"key_hash"`
    Permissions []string          `json:"permissions"`
    Metadata    map[string]string `json:"metadata"`
    CreatedAt   time.Time         `json:"created_at"`
    ExpiresAt   *time.Time        `json:"expires_at,omitempty"`
    LastUsedAt  *time.Time        `json:"last_used_at,omitempty"`
    Enabled     bool              `json:"enabled"`
}
```

**Fields:**
- **ID** - UUID identifier
- **Name** - Human-readable name
- **KeyHash** - SHA-256 hash of the key (never store plaintext)
- **Permissions** - Array of permission strings
- **Metadata** - Custom key-value pairs
- **CreatedAt** - Creation timestamp
- **ExpiresAt** - Optional expiration time
- **LastUsedAt** - Last usage timestamp (updated on validation)
- **Enabled** - Active status

---

### JWT Methods

#### `GenerateToken`

Generate access token.

```go
func (j *JWTService) GenerateToken(userID, username string, roles []string) (string, error)
```

**Parameters:**
- `userID` - Unique user ID
- `username` - Username
- `roles` - User roles

**Returns:** JWT token string or error

**Example:**
```go
token, err := jwtService.GenerateToken("user123", "admin", []string{"admin", "operator"})
```

**Token payload:**
```json
{
  "user_id": "user123",
  "username": "admin",
  "roles": ["admin", "operator"],
  "exp": 1705074600,
  "iat": 1705073700,
  "nbf": 1705073700,
  "iss": "konsul",
  "sub": "user123"
}
```

---

#### `GenerateRefreshToken`

Generate refresh token.

```go
func (j *JWTService) GenerateRefreshToken(userID string) (string, error)
```

**Parameters:**
- `userID` - User identifier

**Returns:** Refresh token string or error

**Example:**
```go
refreshToken, err := jwtService.GenerateRefreshToken("user123")
```

**Refresh token payload:**
```json
{
  "exp": 1705678500,  // 7 days from now
  "iat": 1705073700,
  "nbf": 1705073700,
  "iss": "konsul",
  "sub": "user123"
}
```

---

#### `ValidateToken`

Validate and parse access token.

```go
func (j *JWTService) ValidateToken(tokenString string) (*Claims, error)
```

**Parameters:**
- `tokenString` - JWT token to validate

**Returns:** Claims or error

**Errors:**
- `ErrTokenMissing` - Token string is empty
- `ErrTokenExpired` - Token has expired
- `ErrTokenInvalid` - Token signature invalid or malformed

**Example:**
```go
claims, err := jwtService.ValidateToken(tokenString)
if err != nil {
    if errors.Is(err, auth.ErrTokenExpired) {
        // Handle expired token - try refresh
    }
    return err
}

fmt.Printf("User: %s, Roles: %v\n", claims.Username, claims.Roles)
```

---

#### `ValidateRefreshToken`

Validate refresh token.

```go
func (j *JWTService) ValidateRefreshToken(tokenString string) (string, error)
```

**Parameters:**
- `tokenString` - Refresh token

**Returns:** UserID or error

**Example:**
```go
userID, err := jwtService.ValidateRefreshToken(refreshToken)
if err != nil {
    return err
}
```

---

#### `RefreshToken`

Generate new tokens from refresh token.

```go
func (j *JWTService) RefreshToken(refreshTokenString string, username string, roles []string) (string, string, error)
```

**Parameters:**
- `refreshTokenString` - Current refresh token
- `username` - Username
- `roles` - User roles

**Returns:** New access token, new refresh token, error

**Example:**
```go
newToken, newRefreshToken, err := jwtService.RefreshToken(
    oldRefreshToken,
    "admin",
    []string{"admin"},
)
```

---

### API Key Methods

#### `GenerateAPIKey`

Create new API key.

```go
func (a *APIKeyService) GenerateAPIKey(
    name string,
    permissions []string,
    metadata map[string]string,
    expiresAt *time.Time,
) (string, *APIKey, error)
```

**Parameters:**
- `name` - Key name (required)
- `permissions` - Permission array
- `metadata` - Custom metadata
- `expiresAt` - Optional expiration time

**Returns:**
- Key string (format: `prefix_hexstring`)
- APIKey object
- Error

**Example:**
```go
expiresAt := time.Now().Add(30 * 24 * time.Hour)
keyString, apiKey, err := apiKeyService.GenerateAPIKey(
    "production-api",
    []string{"read", "write"},
    map[string]string{"env": "prod"},
    &expiresAt,
)

// keyString: "konsul_a1b2c3d4e5f6..."
// Save keyString - it won't be shown again!
```

**Key generation:**
- 32 random bytes from `crypto/rand`
- Hex encoded
- Prefixed with service prefix
- Hashed with SHA-256 for storage

---

#### `ValidateAPIKey`

Validate API key and return metadata.

```go
func (a *APIKeyService) ValidateAPIKey(keyString string) (*APIKey, error)
```

**Parameters:**
- `keyString` - Full API key string

**Returns:** APIKey object or error

**Errors:**
- `ErrAPIKeyNotFound` - Key doesn't exist
- `ErrAPIKeyExpired` - Key has expired
- `ErrAPIKeyDisabled` - Key is disabled/revoked

**Side effects:** Updates `LastUsedAt` timestamp

**Example:**
```go
apiKey, err := apiKeyService.ValidateAPIKey("konsul_a1b2c3d4...")
if err != nil {
    if errors.Is(err, auth.ErrAPIKeyExpired) {
        return fmt.Errorf("API key has expired")
    }
    return err
}

fmt.Printf("Key: %s, Permissions: %v\n", apiKey.Name, apiKey.Permissions)
```

---

#### `RevokeAPIKey`

Disable API key (soft delete).

```go
func (a *APIKeyService) RevokeAPIKey(keyID string) error
```

**Parameters:**
- `keyID` - API key UUID

**Returns:** Error if key not found

**Effect:** Sets `Enabled = false`

**Example:**
```go
err := apiKeyService.RevokeAPIKey("123e4567-e89b-12d3-a456-426614174000")
```

---

#### `DeleteAPIKey`

Permanently delete API key.

```go
func (a *APIKeyService) DeleteAPIKey(keyID string) error
```

**Parameters:**
- `keyID` - API key UUID

**Returns:** Error if key not found

**Effect:** Removes from storage permanently

---

#### `ListAPIKeys`

List all API keys (without hashes).

```go
func (a *APIKeyService) ListAPIKeys() []*APIKey
```

**Returns:** Array of APIKey objects (KeyHash omitted)

**Example:**
```go
keys := apiKeyService.ListAPIKeys()
for _, key := range keys {
    fmt.Printf("ID: %s, Name: %s, Enabled: %v\n", key.ID, key.Name, key.Enabled)
}
```

---

#### `GetAPIKey`

Get specific API key by ID.

```go
func (a *APIKeyService) GetAPIKey(keyID string) (*APIKey, error)
```

**Parameters:**
- `keyID` - API key UUID

**Returns:** APIKey object or error

---

#### `UpdateAPIKey`

Update API key properties.

```go
func (a *APIKeyService) UpdateAPIKey(
    keyID string,
    name string,
    permissions []string,
    metadata map[string]string,
    enabled *bool,
) error
```

**Parameters:**
- `keyID` - API key UUID
- `name` - New name (empty = no change)
- `permissions` - New permissions (nil = no change)
- `metadata` - New metadata (nil = no change)
- `enabled` - New enabled status (nil = no change)

**Example:**
```go
enabled := false
err := apiKeyService.UpdateAPIKey(
    "123e4567...",
    "updated-name",
    []string{"read"},
    nil,
    &enabled,
)
```

---

#### `HasPermission`

Check if API key has specific permission.

```go
func (a *APIKeyService) HasPermission(apiKey *APIKey, permission string) bool
```

**Parameters:**
- `apiKey` - API key object
- `permission` - Permission to check

**Returns:** True if key has permission

**Special permissions:**
- `"*"` - Wildcard, grants all permissions

**Example:**
```go
if apiKeyService.HasPermission(apiKey, "write") {
    // Allow write operation
}
```

---

## HTTP API Endpoints

### Authentication Endpoints

#### Login

```http
POST /auth/login
Content-Type: application/json

{
  "user_id": "user123",
  "username": "admin",
  "roles": ["admin"]
}
```

**Response (200 OK):**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_in": 900
}
```

---

#### Verify Token

```http
GET /auth/verify
Authorization: Bearer <token>
```

**Response (200 OK):**
```json
{
  "valid": true,
  "user_id": "user123",
  "username": "admin",
  "roles": ["admin"],
  "expires_at": "2025-01-12T15:30:00Z"
}
```

---

#### Refresh Token

```http
POST /auth/refresh
Content-Type: application/json

{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "username": "admin",
  "roles": ["admin"]
}
```

**Response (200 OK):**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs... (new)",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs... (new)",
  "expires_in": 900
}
```

---

### API Key Endpoints

All API key endpoints require JWT authentication.

#### Create API Key

```http
POST /auth/apikeys
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "name": "production-service",
  "permissions": ["read", "write"],
  "metadata": {"env": "production"},
  "expires_in": 31536000
}
```

**Response (201 Created):**
```json
{
  "key": "konsul_a1b2c3d4e5f6...",
  "api_key": {
    "id": "123e4567-e89b-12d3-a456-426614174000",
    "name": "production-service",
    "permissions": ["read", "write"],
    "metadata": {"env": "production"},
    "created_at": "2025-01-12T10:30:00Z",
    "expires_at": "2026-01-12T10:30:00Z",
    "enabled": true
  }
}
```

---

#### List API Keys

```http
GET /auth/apikeys
Authorization: Bearer <jwt_token>
```

**Response (200 OK):**
```json
[
  {
    "id": "123e4567-e89b-12d3-a456-426614174000",
    "name": "production-service",
    "permissions": ["read", "write"],
    "metadata": {"env": "production"},
    "created_at": "2025-01-12T10:30:00Z",
    "expires_at": "2026-01-12T10:30:00Z",
    "last_used_at": "2025-01-12T12:45:00Z",
    "enabled": true
  }
]
```

---

#### Get API Key

```http
GET /auth/apikeys/:id
Authorization: Bearer <jwt_token>
```

---

#### Update API Key

```http
PUT /auth/apikeys/:id
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "name": "updated-name",
  "permissions": ["read"],
  "enabled": true
}
```

---

#### Revoke API Key

```http
POST /auth/apikeys/:id/revoke
Authorization: Bearer <jwt_token>
```

---

#### Delete API Key

```http
DELETE /auth/apikeys/:id
Authorization: Bearer <jwt_token>
```

---

## Error Responses

### JWT Errors

| Error Code | HTTP Status | Meaning |
|------------|-------------|---------|
| `token_missing` | 401 | No token provided |
| `token_expired` | 401 | Token has expired - use refresh |
| `token_invalid` | 401 | Token signature invalid |

**Error response format:**
```json
{
  "error": "token_expired",
  "message": "Token has expired"
}
```

---

### API Key Errors

| Error Code | HTTP Status | Meaning |
|------------|-------------|---------|
| `apikey_not_found` | 401 | API key doesn't exist |
| `apikey_expired` | 401 | API key has expired |
| `apikey_disabled` | 401 | API key is revoked/disabled |

---

## Security Implementation

### JWT Signing

**Algorithm:** HS256 (HMAC with SHA-256)

**Why HS256:**
- Symmetric key (simpler key management)
- Fast verification
- Sufficient for service-to-service auth
- No public/private key complexity

**Token structure:**
```
eyJhbGciOiJIUzI1NiIs...    # Header
.eyJ1c2VyX2lkIjoiYWRt...   # Payload (claims)
.SflKxwRJSMeKKF2QT4f...     # Signature
```

---

### API Key Hashing

**Hash algorithm:** SHA-256

**Storage:**
- ✅ Hash stored
- ❌ Plaintext NEVER stored

**Validation process:**
```
1. Receive API key: "konsul_abc123..."
2. Hash with SHA-256
3. Look up hash in storage
4. Compare hashes (constant time)
```

**Why secure:**
- Database compromise doesn't reveal keys
- Rainbow table attacks ineffective
- Constant-time comparison prevents timing attacks

---

### Random Generation

**Source:** `crypto/rand` (cryptographically secure)

```go
keyBytes := make([]byte, 32)
rand.Read(keyBytes)  // 32 bytes = 256 bits of entropy
```

---

## Performance Characteristics

### JWT Operations

| Operation | Latency | Notes |
|-----------|---------|-------|
| Generate | ~100µs | HMAC signing |
| Validate | ~50µs | HMAC verification |
| Parse | ~20µs | JSON decode |

### API Key Operations

| Operation | Latency | Notes |
|-----------|---------|-------|
| Generate | ~200µs | Crypto random + SHA-256 |
| Validate | ~50µs | SHA-256 + map lookup |
| List | ~10µs | In-memory |

---

## See Also

- [Authentication User Guide](authentication.md)
- [Authentication Implementation](authentication-implementation.md)
- [ADR-0003](adr/0003-jwt-authentication.md)
- [JWT Specification](https://tools.ietf.org/html/rfc7519)
