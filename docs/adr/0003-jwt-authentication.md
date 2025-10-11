# ADR-0003: JWT-Based Authentication

**Date**: 2024-09-20

**Status**: Accepted

**Deciders**: Konsul Core Team

**Tags**: security, authentication, api

## Context

Konsul needs an authentication mechanism to protect API endpoints from unauthorized access. Requirements:

- Stateless authentication (no session storage)
- Support for programmatic API access
- Token expiration and refresh capabilities
- Easy integration with existing systems
- Support for both short-lived and long-lived credentials
- Role-based access control (future)

The system must support:
- Human users (via web UI and CLI)
- Service-to-service communication
- CI/CD pipelines
- Monitoring systems

## Decision

We will implement a **dual authentication system**:

1. **JWT (JSON Web Tokens)** for human users and interactive sessions
2. **API Keys** for programmatic access and service-to-service communication

### JWT Design
- HS256 signing algorithm (symmetric key)
- Short-lived access tokens (15 minutes default)
- Long-lived refresh tokens (7 days default)
- Claims include: user_id, username, roles, expiry
- Tokens issued via `/auth/login` endpoint

### API Key Design
- Long-lived credentials with optional expiration
- Prefix-based format: `konsul_<random_string>`
- Stored with SHA-256 hash (only hash stored)
- Support permissions and metadata
- Manageable via authenticated API endpoints

## Alternatives Considered

### Alternative 1: OAuth2/OIDC
- **Pros**:
  - Industry standard for authorization
  - Supports external identity providers
  - Well-established flows
  - Good for enterprise integration
- **Cons**:
  - Complex implementation and setup
  - Requires external IdP or building auth server
  - Overkill for simple use cases
  - Higher operational complexity
- **Reason for rejection**: Too complex for initial version; can add later via API keys

### Alternative 2: mTLS (Mutual TLS)
- **Pros**:
  - Strong cryptographic authentication
  - No token management needed
  - Built into TLS protocol
  - Excellent for service mesh
- **Cons**:
  - Certificate management complexity
  - Difficult for human users
  - CLI and web UI integration challenging
  - No fine-grained permissions without additional layer
- **Reason for rejection**: Better suited for service-to-service; not user-friendly

### Alternative 3: Basic Authentication
- **Pros**:
  - Simple to implement
  - Widely supported
  - No token management
- **Cons**:
  - Requires credential storage
  - No expiration mechanism
  - Credentials sent with every request
  - Poor security model (credentials in headers)
- **Reason for rejection**: Insufficient security for production use

### Alternative 4: API Keys Only
- **Pros**:
  - Simple to implement and use
  - Good for programmatic access
  - Easy revocation
- **Cons**:
  - No expiration/refresh flow
  - Long-lived credentials riskier
  - Not ideal for human users
  - No built-in session management
- **Reason for rejection**: Need both interactive and programmatic patterns

## Consequences

### Positive
- Stateless authentication reduces server complexity
- JWT standard widely understood and supported
- Refresh tokens enable long sessions with short-lived access tokens
- API keys provide simple programmatic access
- Token expiration limits blast radius of compromised credentials
- Can add RBAC by extending JWT claims
- No database required for session storage

### Negative
- Cannot invalidate JWTs before expiration (unless blacklist added)
- JWT secret compromise affects all tokens
- Token size larger than session IDs (sent with each request)
- Need secure secret management for JWT signing key
- API key storage requires hashing and secure generation

### Neutral
- Middleware layer required for all protected endpoints
- Public paths must be explicitly configured
- Client libraries need token refresh logic
- Need monitoring for token usage patterns

## Implementation Notes

### Configuration
```go
Auth: AuthConfig{
    Enabled:       true,
    JWTSecret:     "secure-secret-min-32-chars",
    JWTExpiry:     15 * time.Minute,
    RefreshExpiry: 7 * 24 * time.Hour,
    RequireAuth:   true,
    PublicPaths:   []string{"/health", "/metrics"},
}
```

### Middleware Stack
1. Check if path is public → allow
2. Try JWT authentication (Bearer token)
3. Try API key authentication (X-API-Key or ApiKey header)
4. Reject if both fail

### Security Measures
- JWT secret minimum 32 characters
- API keys hashed with SHA-256 (only hash stored)
- Secure random generation (crypto/rand)
- Rate limiting per API key
- Audit logging for auth events (future)

### Future Enhancements
- Add token blacklist for early revocation
- Implement RBAC with fine-grained permissions
- Support OIDC for enterprise integration
- Add audit logging for compliance

## References

- [JWT.io - Introduction](https://jwt.io/introduction)
- [RFC 7519 - JSON Web Token](https://tools.ietf.org/html/rfc7519)
- [OWASP API Security](https://owasp.org/www-project-api-security/)
- [Konsul auth package](../../internal/auth/)
- [Konsul middleware package](../../internal/middleware/)

---

## Revision History

| Date | Author | Changes |
|------|--------|---------|
| 2024-09-20 | Konsul Team | Initial version |
