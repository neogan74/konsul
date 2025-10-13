# Template Engine Documentation Index

Complete documentation for the Konsul template engine.

## Quick Links

| Document | Description | Audience |
|----------|-------------|----------|
| [User Guide](template-engine.md) | End-user documentation and examples | Users |
| [API Reference](template-engine-api.md) | Complete API and function reference | Developers |
| [Implementation Guide](template-engine-implementation.md) | Internal architecture and design | Contributors |
| [Troubleshooting](template-engine-troubleshooting.md) | Debug and fix common issues | Operators |
| [Performance](template-engine-performance.md) | Optimization and tuning | DevOps |
| [ADR-0015](adr/0015-template-engine.md) | Architecture decision record | Architects |

---

## For End Users

### Getting Started

1. **[User Guide](template-engine.md)** - Start here!
   - Quick start tutorial
   - Template function reference
   - CLI usage examples
   - Best practices

2. **[Example Templates](../examples/templates/README.md)**
   - nginx configuration
   - HAProxy configuration
   - Application environment files
   - Simple examples

### When You Have Problems

3. **[Troubleshooting Guide](template-engine-troubleshooting.md)**
   - Common issues and solutions
   - Debugging techniques
   - Error messages explained
   - How to get help

---

## For Developers

### Understanding the Code

4. **[API Reference](template-engine-api.md)**
   - Complete type definitions
   - Function signatures
   - Usage examples
   - Error handling

5. **[Implementation Guide](template-engine-implementation.md)**
   - Architecture overview
   - Component deep dive
   - Design patterns
   - Data flow diagrams
   - How to extend the engine

### Code Examples

```go
// Basic usage
engine := template.New(
    template.Config{
        Templates: []template.TemplateConfig{{
            Source:      "app.conf.tpl",
            Destination: "/etc/app.conf",
            Perms:       0644,
        }},
        Once: true,
    },
    kvStore,
    serviceStore,
    logger.GetDefault(),
)

if err := engine.RunOnce(); err != nil {
    log.Fatal(err)
}
```

---

## For Operators

### Production Deployment

6. **[Performance Guide](template-engine-performance.md)**
   - Benchmarks and metrics
   - Configuration tuning
   - Template optimization
   - System tuning
   - Scaling strategies

### Monitoring and Alerting

- Key metrics to track
- Alerting thresholds
- Profiling in production
- Capacity planning

---

## For Architects

### Design Decisions

7. **[ADR-0015: Template Engine](adr/0015-template-engine.md)**
   - Context and motivation
   - Architecture decisions
   - Alternatives considered
   - Trade-offs and consequences
   - Implementation phases

8. **[Implementation Summary](TEMPLATE_IMPLEMENTATION.md)**
   - What was implemented
   - Test results
   - File structure
   - Known limitations
   - Future enhancements

---

## Documentation Map

### By Topic

#### **Installation & Setup**
- [User Guide: Installation](template-engine.md#installation)
- [User Guide: Quick Start](template-engine.md#quick-start)

#### **Template Syntax**
- [User Guide: Template Functions](template-engine.md#template-functions)
- [API Reference: Template Functions](template-engine-api.md#template-functions)
- [Examples](../examples/templates/README.md)

#### **Configuration**
- [API Reference: Config Types](template-engine-api.md#types)
- [Performance: Configuration Tuning](template-engine-performance.md#configuration-tuning)

#### **CLI Tool**
- [User Guide: CLI Usage](template-engine.md#cli-usage)
- [API Reference: CLI Tool](template-engine-api.md#cli-tool)

#### **Debugging**
- [Troubleshooting: Common Issues](template-engine-troubleshooting.md#common-issues)
- [Troubleshooting: Debugging Techniques](template-engine-troubleshooting.md#debugging-techniques)

#### **Performance**
- [Performance: Benchmarks](template-engine-performance.md#benchmarks)
- [Performance: Optimization Strategies](template-engine-performance.md#optimization-strategies)
- [Performance: Monitoring](template-engine-performance.md#monitoring)

#### **Development**
- [Implementation: Architecture](template-engine-implementation.md#architecture-overview)
- [Implementation: Component Deep Dive](template-engine-implementation.md#component-deep-dive)
- [Implementation: Extending](template-engine-implementation.md#extending-the-engine)

#### **Testing**
- [Implementation: Testing Strategy](template-engine-implementation.md#testing-strategy)
- [Implementation Summary: Test Results](TEMPLATE_IMPLEMENTATION.md#test-results)

---

## Documentation Statistics

| Document | Pages | Words | Audience |
|----------|-------|-------|----------|
| User Guide | 15 | 4,500 | Users |
| API Reference | 25 | 7,000 | Developers |
| Implementation Guide | 35 | 10,000 | Contributors |
| Troubleshooting | 20 | 6,000 | Operators |
| Performance | 18 | 5,500 | DevOps |
| ADR-0015 | 12 | 4,000 | Architects |
| **Total** | **125** | **37,000** | All |

---

## Quick Reference

### Template Functions

| Function | Syntax | Returns | Example |
|----------|--------|---------|---------|
| kv | `{{ kv "key" }}` | string | `{{ kv "config/host" }}` |
| kvTree | `{{ kvTree "prefix" }}` | []KVPair | `{{ range kvTree "config/" }}` |
| service | `{{ service "name" }}` | []Service | `{{ range service "web" }}` |
| services | `{{ services }}` | []Service | `{{ range services }}` |
| env | `{{ env "VAR" }}` | string | `{{ env "HOME" }}` |
| file | `{{ file "path" }}` | string | `{{ file "/etc/hostname" }}` |

[Full function reference →](template-engine-api.md#template-functions)

### CLI Commands

```bash
# Once mode
konsul-template -template app.tpl -dest app.conf -once

# Watch mode
konsul-template -template app.tpl -dest app.conf

# Dry-run
konsul-template -template app.tpl -dest app.conf -dry -once
```

[Full CLI reference →](template-engine-api.md#cli-tool)

### Common Tasks

| Task | Documentation |
|------|---------------|
| Create your first template | [User Guide: Quick Start](template-engine.md#quick-start) |
| Add a custom function | [Implementation: Extending](template-engine-implementation.md#adding-new-template-functions) |
| Debug template errors | [Troubleshooting: Template Debugging](template-engine-troubleshooting.md#template-debugging) |
| Optimize performance | [Performance: Template Optimization](template-engine-performance.md#template-optimization) |
| Deploy to production | [Performance: Production Checklist](template-engine-performance.md#performance-checklist) |

---

## Contributing

### Documentation Standards

- **User-facing docs**: Clear, concise, example-driven
- **Technical docs**: Detailed, accurate, code examples
- **API docs**: Complete, consistent, well-formatted

### How to Update

1. Edit relevant .md file in `/docs`
2. Update this index if adding new docs
3. Test all code examples
4. Submit PR with description

### Documentation TODOs

Future documentation to add:

- [ ] Video tutorials
- [ ] Interactive examples
- [ ] Multi-language examples (Python, JavaScript)
- [ ] Migration guide from consul-template
- [ ] Production deployment guide
- [ ] Docker/Kubernetes deployment
- [ ] Security hardening guide
- [ ] Backup and recovery procedures

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 0.1.0 | 2025-10-10 | Initial documentation release |

---

## Feedback

Found an issue or have a suggestion?

- **Documentation bugs**: [GitHub Issues](https://github.com/yourusername/konsul/issues)
- **Feature requests**: [GitHub Discussions](https://github.com/yourusername/konsul/discussions)
- **Questions**: See [Troubleshooting Guide](template-engine-troubleshooting.md#getting-help)

---

## License

Documentation is licensed under [Creative Commons BY 4.0](https://creativecommons.org/licenses/by/4.0/).

Code examples are licensed under [MIT License](../LICENSE).
