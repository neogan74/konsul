# Architecture Decision Records (ADRs)

This directory contains Architecture Decision Records (ADRs) for Konsul. ADRs document significant architectural decisions made during the development of this project.

## What is an ADR?

An Architecture Decision Record (ADR) is a document that captures an important architectural decision made along with its context and consequences.

## ADR Format

Each ADR follows a standard format (see [template.md](template.md)) with the following sections:

- **Status**: Proposed, Accepted, Deprecated, Superseded
- **Context**: The issue motivating this decision
- **Decision**: The change being proposed or decided
- **Consequences**: The resulting context after applying the decision

## ADR Index

| ADR | Title | Status |
|-----|-------|--------|
| [ADR-0001](0001-use-fiber-web-framework.md) | Use Fiber Web Framework | Accepted |
| [ADR-0002](0002-badger-for-persistence.md) | BadgerDB for Persistence Layer | Accepted |
| [ADR-0003](0003-jwt-authentication.md) | JWT-Based Authentication | Accepted |
| [ADR-0004](0004-prometheus-metrics.md) | Prometheus for Metrics | Accepted |
| [ADR-0005](0005-structured-logging.md) | Structured Logging with Custom Logger | Accepted |
| [ADR-0006](0006-dns-service-discovery.md) | DNS Interface for Service Discovery | Accepted |
| [ADR-0007](0007-opentelemetry-tracing.md) | OpenTelemetry for Distributed Tracing | Accepted |
| [ADR-0008](0008-migrate-fiber-to-chi.md) | Migrate from Fiber to Chi Router | Proposed |
| [ADR-0009](0009-react-admin-ui.md) | React-Based Admin UI with Vite and Tailwind CSS | Accepted |
| [ADR-0010](0010-acl-system.md) | Access Control List (ACL) System | Proposed |
| [ADR-0011](0011-raft-clustering-ha.md) | Raft Consensus for Clustering and High Availability | Proposed |
| [ADR-0012](0012-cli-tool-konsulctl.md) | Command-Line Interface Tool (konsulctl) | Accepted |
| [ADR-0013](0013-token-bucket-rate-limiting.md) | Token Bucket Rate Limiting | Accepted |
| [ADR-0014](0014-rate-limiting-management-api.md) | Rate Limiting Management API and Observability | Proposed |
| [ADR-0015](0015-template-engine.md) | Template Engine for Configuration Management | Proposed |
| [ADR-0016](0016-graphql-api-interface.md) | GraphQL API Interface | Proposed |
| [ADR-0017](0017-service-tags-metadata.md) | Service Tags and Metadata | Proposed |
| [ADR-0018](0018-load-balancing-strategies.md) | Load Balancing Strategies | Proposed |
| [ADR-0019](0019-audit-logging.md) | Audit Logging for Operational Changes | Accepted |
| [ADR-0020](0020-compare-and-swap-operations.md) | Compare-And-Swap (CAS) Operations | Accepted |
| [ADR-0021](0021-kv-watch-subscribe.md) | KV Store Watch/Subscribe System | Proposed |

## Creating a New ADR

1. Copy the [template.md](template.md) file
2. Name it with the next sequential number: `XXXX-short-title.md`
3. Fill in the sections with relevant information
4. Update this README with a link to the new ADR
5. Submit as part of your pull request

## Lifecycle

ADRs are immutable once accepted. If a decision needs to be changed:

1. Create a new ADR that supersedes the old one
2. Update the old ADR's status to "Superseded by ADR-XXXX"
3. Reference the old ADR in the new one's context

## Resources

- [Michael Nygard's ADR article](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions)
- [GitHub ADR organization](https://adr.github.io/)
