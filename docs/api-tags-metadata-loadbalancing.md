# Tags, Metadata, and Load Balancing API Reference

This document provides a comprehensive guide to using tags, metadata, and load balancing features in Konsul.

## Table of Contents

- [Overview](#overview)
- [Service Tags](#service-tags)
- [Service Metadata](#service-metadata)
- [HTTP API Endpoints](#http-api-endpoints)
- [GraphQL API](#graphql-api)
- [Load Balancing](#load-balancing)
- [Examples](#examples)
- [Best Practices](#best-practices)

## Overview

Konsul supports rich service discovery capabilities through:

- **Tags**: Simple string labels for categorizing and filtering services
- **Metadata**: Key-value pairs for storing additional service information
- **Load Balancing**: Intelligent service instance selection using multiple strategies

## Service Tags

### Description

Tags are strings that categorize and label services. They are useful for:
- Environment identification (e.g., `env:production`, `env:staging`)
- Version labeling (e.g., `version:v1`, `version:v2`)
- Protocol specification (e.g., `http`, `grpc`, `tcp`)
- Service grouping (e.g., `service:api`, `service:web`)

### Validation Rules

- **Maximum tags per service**: 64
- **Maximum tag length**: 255 characters
- **Allowed characters**: alphanumeric, `-`, `_`, `:`, `.`, `/`
- **Format**: `^[a-zA-Z0-9\-_:./]+$`
- **Constraints**: No empty tags, no duplicates

### Example Tags

```json
{
  "tags": [
    "service:api",
    "env:production",
    "version:v2",
    "http",
    "region:us-east-1"
  ]
}
```

## Service Metadata

### Description

Metadata provides additional structured information about services as key-value pairs:
- Team ownership (e.g., `team: platform`)
- Cost center tracking (e.g., `cost-center: engineering`)
- Deployment information (e.g., `deployment-id: abc123`)
- Custom application data

### Validation Rules

- **Maximum metadata keys**: 64
- **Key maximum length**: 128 characters
- **Value maximum length**: 512 characters
- **Key format**: `^[a-zA-Z0-9\-_]+$` (alphanumeric, `-`, `_` only)
- **Reserved prefixes**: `konsul_`, `_` (blocked)
- **Constraints**: No empty keys

### Example Metadata

```json
{
  "meta": {
    "team": "platform",
    "owner": "john.doe@example.com",
    "cost-center": "engineering",
    "deployment-id": "deploy-2024-001",
    "maintenance-window": "Sunday 02:00-04:00"
  }
}
```

## HTTP API Endpoints

### Service Registration

Register a service with tags and metadata:

```http
PUT /register
Content-Type: application/json

{
  "name": "api-server-1",
  "address": "10.0.1.10",
  "port": 8080,
  "tags": ["service:api", "env:production", "http"],
  "meta": {
    "team": "platform",
    "version": "2.1.0"
  }
}
```

**Response:**
```json
{
  "message": "service registered",
  "service": {
    "name": "api-server-1",
    "address": "10.0.1.10",
    "port": 8080,
    "tags": ["service:api", "env:production", "http"],
    "meta": {
      "team": "platform",
      "version": "2.1.0"
    }
  }
}
```

### Query Services by Tags

Query services that have ALL specified tags (AND logic):

```http
GET /services/query/tags?tags=service:api&tags=env:production
```

**Response:**
```json
{
  "count": 2,
  "services": [
    {
      "name": "api-server-1",
      "address": "10.0.1.10",
      "port": 8080,
      "tags": ["service:api", "env:production", "http"]
    },
    {
      "name": "api-server-2",
      "address": "10.0.1.11",
      "port": 8080,
      "tags": ["service:api", "env:production", "http"]
    }
  ],
  "query": {
    "tags": ["service:api", "env:production"]
  }
}
```

### Query Services by Metadata

Query services that match ALL specified metadata key-value pairs (AND logic):

```http
GET /services/query/metadata?team=platform&env=prod
```

**Response:**
```json
{
  "count": 3,
  "services": [
    {
      "name": "api-server-1",
      "address": "10.0.1.10",
      "port": 8080,
      "meta": {
        "team": "platform",
        "env": "prod"
      }
    }
  ],
  "query": {
    "metadata": {
      "team": "platform",
      "env": "prod"
    }
  }
}
```

### Combined Query (Tags + Metadata)

Query services matching both tags and metadata:

```http
GET /services/query?tags=service:api&tags=http&meta.team=platform&meta.env=prod
```

**Note**: Metadata filters are prefixed with `meta.` to distinguish them from tags.

**Response:**
```json
{
  "count": 2,
  "services": [...],
  "query": {
    "tags": ["service:api", "http"],
    "metadata": {
      "team": "platform",
      "env": "prod"
    }
  }
}
```

## GraphQL API

### Schema Types

#### Service Type

```graphql
type Service {
  name: String!
  address: String!
  port: Int!
  status: ServiceStatus!
  expiresAt: Time!
  tags: [String!]!
  metadata: [MetadataEntry!]!
  checks: [HealthCheck!]!
}

type MetadataEntry {
  key: String!
  value: String!
}

input MetadataFilter {
  key: String!
  value: String!
}
```

### Queries

#### Query by Tags

```graphql
query {
  servicesByTags(tags: ["service:api", "env:production"]) {
    name
    address
    port
    tags
    metadata {
      key
      value
    }
  }
}
```

#### Query by Metadata

```graphql
query {
  servicesByMetadata(filters: [
    { key: "team", value: "platform" },
    { key: "env", value: "prod" }
  ]) {
    name
    address
    port
    metadata {
      key
      value
    }
  }
}
```

#### Combined Query

```graphql
query {
  servicesByQuery(
    tags: ["service:api", "http"],
    metadata: [
      { key: "team", value: "platform" }
    ]
  ) {
    name
    address
    port
    tags
    metadata {
      key
      value
    }
  }
}
```

## Load Balancing

### Overview

Konsul provides client-side load balancing with three strategies:
- **Round-Robin**: Distributes requests evenly across all instances
- **Random**: Selects a random instance for each request
- **Least-Connections**: Selects the instance with the fewest active connections

### Endpoints

#### Get Current Strategy

```http
GET /lb/strategy
```

**Response:**
```json
{
  "strategy": "round-robin"
}
```

#### Update Strategy

```http
PUT /lb/strategy
Content-Type: application/json

{
  "strategy": "least-connections"
}
```

**Valid strategies**: `round-robin`, `random`, `least-connections`

**Response:**
```json
{
  "message": "strategy updated",
  "strategy": "least-connections"
}
```

#### Select Service Instance

Select an instance using a service tag:

```http
GET /lb/service/service:api
```

**Response:**
```json
{
  "service": {
    "name": "api-server-2",
    "address": "10.0.1.11",
    "port": 8080,
    "tags": ["service:api", "env:production"]
  },
  "strategy": "round-robin"
}
```

#### Select by Tags

Select an instance matching all specified tags:

```http
GET /lb/tags?tags=service:api&tags=env:production&tags=http
```

**Response:**
```json
{
  "service": {
    "name": "api-server-1",
    "address": "10.0.1.10",
    "port": 8080
  },
  "strategy": "round-robin",
  "query": {
    "tags": ["service:api", "env:production", "http"]
  }
}
```

#### Select by Metadata

Select an instance matching all specified metadata:

```http
GET /lb/metadata?team=platform&version=2.1.0
```

**Response:**
```json
{
  "service": {
    "name": "api-server-3",
    "address": "10.0.1.12",
    "port": 8080
  },
  "strategy": "least-connections",
  "query": {
    "metadata": {
      "team": "platform",
      "version": "2.1.0"
    }
  }
}
```

#### Combined Load Balancing

Select using both tags and metadata:

```http
GET /lb/query?tags=service:api&meta.team=platform&meta.env=prod
```

**Response:**
```json
{
  "service": {
    "name": "api-server-1",
    "address": "10.0.1.10",
    "port": 8080
  },
  "strategy": "round-robin",
  "query": {
    "tags": ["service:api"],
    "metadata": {
      "team": "platform",
      "env": "prod"
    }
  }
}
```

## Examples

### Example 1: Multi-Environment API Deployment

Register three API servers in different environments:

```bash
# Production instance 1
curl -X PUT http://localhost:8500/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "api-prod-1",
    "address": "10.0.1.10",
    "port": 8080,
    "tags": ["service:api", "env:production", "http", "version:v2"],
    "meta": {
      "team": "platform",
      "datacenter": "us-east-1",
      "deployment": "2024-11-01"
    }
  }'

# Production instance 2
curl -X PUT http://localhost:8500/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "api-prod-2",
    "address": "10.0.1.11",
    "port": 8080,
    "tags": ["service:api", "env:production", "http", "version:v2"],
    "meta": {
      "team": "platform",
      "datacenter": "us-west-1",
      "deployment": "2024-11-01"
    }
  }'

# Staging instance
curl -X PUT http://localhost:8500/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "api-staging-1",
    "address": "10.0.2.10",
    "port": 8080,
    "tags": ["service:api", "env:staging", "http", "version:v3-beta"],
    "meta": {
      "team": "platform",
      "datacenter": "us-east-1"
    }
  }'
```

Query production API servers:

```bash
curl "http://localhost:8500/services/query/tags?tags=service:api&tags=env:production"
```

Load balance across production instances:

```bash
# Set load balancing strategy
curl -X PUT http://localhost:8500/lb/strategy \
  -H "Content-Type: application/json" \
  -d '{"strategy": "round-robin"}'

# Get next instance
curl "http://localhost:8500/lb/tags?tags=service:api&tags=env:production"
```

### Example 2: Team-Based Service Discovery

Register services owned by different teams:

```bash
# Platform team service
curl -X PUT http://localhost:8500/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "auth-service-1",
    "address": "10.0.3.10",
    "port": 9000,
    "tags": ["service:auth", "grpc"],
    "meta": {
      "team": "platform",
      "owner": "platform-team@example.com",
      "sla": "99.99"
    }
  }'

# Data team service
curl -X PUT http://localhost:8500/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "analytics-service-1",
    "address": "10.0.4.10",
    "port": 8080,
    "tags": ["service:analytics", "http"],
    "meta": {
      "team": "data",
      "owner": "data-team@example.com",
      "sla": "99.9"
    }
  }'
```

Query all platform team services:

```bash
curl "http://localhost:8500/services/query/metadata?team=platform"
```

### Example 3: GraphQL Query

```graphql
query PlatformProductionServices {
  servicesByQuery(
    tags: ["env:production"],
    metadata: [
      { key: "team", value: "platform" }
    ]
  ) {
    name
    address
    port
    tags
    metadata {
      key
      value
    }
    status
  }
}
```

## Best Practices

### Tagging Strategy

1. **Use consistent naming conventions**:
   - Environment: `env:production`, `env:staging`, `env:dev`
   - Service grouping: `service:api`, `service:web`, `service:worker`
   - Versions: `version:v1`, `version:v2`
   - Protocols: `http`, `grpc`, `tcp`

2. **Keep tags simple and queryable**:
   - Prefer `env:production` over `environment:production-cluster-1`
   - Use metadata for complex values

3. **Use service tags for load balancing**:
   - Tag instances with `service:api` to enable load balancing across all API instances
   - Each instance must have a unique name (e.g., `api-1`, `api-2`, `api-3`)

### Metadata Best Practices

1. **Store operational information**:
   - Team ownership
   - Contact information
   - Deployment identifiers
   - SLA requirements
   - Maintenance windows

2. **Use meaningful keys**:
   - `team` instead of `t`
   - `deployment-id` instead of `did`

3. **Avoid storing sensitive data**:
   - Do not store passwords, API keys, or secrets in metadata
   - Use configuration management for sensitive values

### Load Balancing Best Practices

1. **Choose the right strategy**:
   - **Round-Robin**: Best for stateless services with uniform capacity
   - **Random**: Good for distributed systems, simpler than round-robin
   - **Least-Connections**: Best for stateful or long-running connections

2. **Use consistent service tags**:
   - Tag all instances of a logical service with the same `service:name` tag
   - Example: `service:api` for all API server instances

3. **Track connections for least-connections strategy**:
   ```go
   // After selecting a service
   service, ok := balancer.SelectService("service:api")

   // Track connection lifecycle
   balancer.IncrementConnections(service)
   defer balancer.DecrementConnections(service)

   // Make request to service...
   ```

### Query Optimization

1. **Use specific queries**:
   - Query by tags when possible (indexed, O(1) lookup)
   - Combine tags and metadata for precise results

2. **Limit query complexity**:
   - Avoid querying with too many filters
   - Consider denormalizing data into tags for frequently queried attributes

3. **Cache query results**:
   - For frequently accessed services, cache the query result
   - Respect TTL and service expiration

## Error Handling

### Validation Errors

```json
{
  "error": "tag validation failed: tag exceeds maximum length of 255 characters"
}
```

Common validation errors:
- Tag format violation
- Tag or metadata count exceeds maximum
- Reserved metadata key prefix
- Empty keys or values

### Not Found Errors

```json
{
  "error": "No service instances matching query"
}
```

### Strategy Update Errors

```json
{
  "error": "Invalid strategy. Must be one of: round-robin, random, least-connections"
}
```

## Summary

Konsul's tags, metadata, and load balancing features provide:

- ✅ Flexible service categorization with tags
- ✅ Rich metadata for operational context
- ✅ Fast indexed queries (O(1) lookup)
- ✅ Multiple load balancing strategies
- ✅ REST and GraphQL APIs
- ✅ Comprehensive validation

For more information, see:
- [Main README](../README.md)
- [Architecture Documentation](./architecture.md)
- [API Examples](./examples/)
