# Documentation Organization Guide

This document describes the current organization of Konsul documentation and provides recommendations for future improvements.

---

## Current Structure

### Flat Organization (Current)

```
docs/
├── README.md                           # Documentation hub
├── INDEX.md                            # Complete searchable index
├── TODO.md                             # Future work
│
├── adr/                                # Architecture Decision Records (18 files)
│   ├── README.md
│   ├── template.md
│   └── 0001-*.md through 0015-*.md
│
├── Core Features (11 files)
│   ├── dns-*.md (5 files)             # DNS feature docs
│   ├── template-engine-*.md (7 files) # Template feature docs
│   ├── persistence-*.md (3 files)     # Persistence feature docs
│   └── acl-guide.md                   # ACL documentation
│
├── Security (3 files)
│   ├── authentication.md
│   ├── authentication-api.md
│   └── rate-limiting.md
│
├── Observability (3 files)
│   ├── metrics.md
│   ├── logging.md
│   └── tracing.md
│
├── Tools (2 files)
│   ├── konsulctl.md
│   └── admin-ui.md
│
└── Operations (1 file)
    └── deployment.md
```

**Pros:**
- ✅ Easy to browse
- ✅ Simple to navigate
- ✅ No nested navigation
- ✅ Clear file names

**Cons:**
- ❌ 29 files in one directory
- ❌ Difficult to see feature boundaries
- ❌ Related docs not grouped together
- ❌ Scales poorly as more features added

---

## Recommended: Feature-Based Organization

### Option A: By Feature (Recommended)

```
docs/
├── README.md                          # Documentation hub
├── INDEX.md                           # Complete index
├── getting-started.md                 # Quick start guide
├── TODO.md                            # Future work
│
├── adr/                               # Architecture Decision Records
│   └── (existing ADR files)
│
├── features/                          # Feature-specific documentation
│   ├── authentication/
│   │   ├── README.md                 # authentication.md
│   │   ├── api-reference.md          # authentication-api.md
│   │   └── examples/
│   │
│   ├── dns/
│   │   ├── README.md                 # dns-service-discovery.md
│   │   ├── api-reference.md          # dns-api.md
│   │   ├── implementation.md         # dns-implementation.md
│   │   ├── troubleshooting.md        # dns-troubleshooting.md
│   │   ├── complete-guide.md         # DNS_DOCS_COMPLETE.md
│   │   └── index.md                  # DNS_DOCS_INDEX.md
│   │
│   ├── template-engine/
│   │   ├── README.md                 # template-engine.md
│   │   ├── api-reference.md          # template-engine-api.md
│   │   ├── implementation.md         # template-engine-implementation.md
│   │   ├── performance.md            # template-engine-performance.md
│   │   ├── troubleshooting.md        # template-engine-troubleshooting.md
│   │   ├── complete-guide.md         # TEMPLATE_DOCS_COMPLETE.md
│   │   └── index.md                  # TEMPLATE_DOCS_INDEX.md
│   │
│   ├── persistence/
│   │   ├── README.md                 # Overview
│   │   ├── api-reference.md          # persistence-api.md
│   │   ├── implementation.md         # persistence-implementation.md
│   │   └── badger.md                 # persistence-badger.md
│   │
│   ├── rate-limiting/
│   │   └── README.md                 # rate-limiting.md
│   │
│   ├── acl/
│   │   └── README.md                 # acl-guide.md
│   │
│   └── observability/
│       ├── metrics.md
│       ├── logging.md
│       └── tracing.md
│
├── tools/                             # CLI and UI tools
│   ├── konsulctl.md
│   └── admin-ui.md
│
└── operations/                        # Deployment and operations
    └── deployment.md
```

**Pros:**
- ✅ Clear feature boundaries
- ✅ Related docs grouped together
- ✅ Scales well with new features
- ✅ Easy to navigate by feature
- ✅ Each feature can have its own README

**Cons:**
- ❌ Deeper directory structure
- ❌ Requires updating links
- ❌ More navigation to find files

---

### Option B: By Document Type

```
docs/
├── README.md
├── INDEX.md
│
├── adr/                               # Architecture decisions
│
├── guides/                            # User-facing guides
│   ├── authentication.md
│   ├── dns.md
│   ├── templates.md
│   ├── rate-limiting.md
│   ├── logging.md
│   ├── tracing.md
│   └── acl.md
│
├── api/                               # API references
│   ├── authentication.md
│   ├── dns.md
│   ├── persistence.md
│   └── templates.md
│
├── implementation/                    # Technical deep dives
│   ├── dns.md
│   ├── persistence.md
│   ├── templates.md
│   └── badger.md
│
├── troubleshooting/                   # Problem solving
│   ├── dns.md
│   └── templates.md
│
├── tools/                             # CLI and UI
│   ├── konsulctl.md
│   └── admin-ui.md
│
└── operations/                        # Deployment
    ├── deployment.md
    └── metrics.md
```

**Pros:**
- ✅ Clear document purpose
- ✅ Easy to find all API docs
- ✅ Easy to find all guides

**Cons:**
- ❌ Feature docs scattered across directories
- ❌ Harder to see complete feature documentation
- ❌ Duplicate organization effort

---

### Option C: Hybrid Approach

```
docs/
├── README.md
├── INDEX.md
│
├── guides/                            # High-level user guides
│   ├── getting-started.md
│   ├── authentication.md
│   ├── service-discovery.md
│   └── monitoring.md
│
├── features/                          # Detailed feature docs
│   ├── dns/
│   ├── templates/
│   ├── persistence/
│   └── acl/
│
├── reference/                         # API references
│   ├── rest-api.md
│   ├── cli.md
│   └── configuration.md
│
├── operations/                        # Ops guides
│   ├── deployment.md
│   └── troubleshooting.md
│
└── adr/                               # Architecture decisions
```

---

## Migration Plan

If you decide to reorganize, here's a safe migration approach:

### Phase 1: Create New Structure (No Breaking Changes)
1. Create new directory structure
2. Copy files to new locations
3. Keep old files in place
4. Update internal links in new locations

### Phase 2: Add Redirects
1. Replace old files with redirect notes:
   ```markdown
   # This file has moved

   This documentation has been reorganized.

   **New location**: [features/dns/README.md](features/dns/README.md)

   You will be automatically redirected in 3 seconds...
   ```

### Phase 3: Update External Links
1. Update links in main README.md
2. Update links in code comments
3. Update links in CI/CD
4. Announce change in release notes

### Phase 4: Remove Old Files
1. After grace period (1-2 releases)
2. Remove old files
3. Keep redirects or create 404 pages

---

## Current Recommendation

**Keep the flat structure with improved navigation:**

✅ **Current approach (with INDEX.md)** is best because:
- Documentation is still growing
- Flat structure is easier to maintain
- INDEX.md provides excellent navigation
- No breaking changes needed
- Easy to grep and search

**When to reorganize:**
Consider reorganization when:
- Documentation exceeds 50+ files
- Multiple contributors working on different features
- Need to maintain multiple versions
- Feature documentation becomes unwieldy

---

## Navigation Improvements (Already Implemented)

✅ **INDEX.md** - Complete searchable index
✅ **README.md** - Hub with categorized links
✅ **Feature indexes** - DNS_DOCS_INDEX.md, TEMPLATE_DOCS_INDEX.md
✅ **Complete guides** - DNS_DOCS_COMPLETE.md, TEMPLATE_DOCS_COMPLETE.md
✅ **ADR README** - Architecture decision index

---

## Best Practices

### File Naming Conventions

**Current (Good):**
```
feature-name.md              # Main guide
feature-name-api.md          # API reference
feature-name-implementation.md # Technical details
feature-name-troubleshooting.md # Problem solving
```

**Alternative (if reorganizing):**
```
features/feature-name/
├── README.md                # Main guide
├── api-reference.md         # API docs
├── implementation.md        # Technical details
└── troubleshooting.md       # Issues
```

### Link Management

**Use relative links:**
```markdown
[Authentication Guide](authentication.md)
[DNS API](dns-api.md)
```

**Not absolute paths:**
```markdown
<!-- Avoid -->
[Guide](/docs/authentication.md)
[API](https://github.com/.../docs/dns-api.md)
```

### Cross-referencing

Always link related documentation:
```markdown
## See Also

- [Authentication API](authentication-api.md)
- [Rate Limiting](rate-limiting.md)
- [ADR-0003: JWT Authentication](adr/0003-jwt-authentication.md)
```

---

## Tools for Documentation

### Search
```bash
# Find all references to a feature
grep -r "rate limit" docs/

# Find broken links (requires link-checker)
find docs -name "*.md" -exec markdown-link-check {} \;
```

### Generate TOC
```bash
# Auto-generate table of contents
doctoc docs/README.md
```

### Validate Links
```bash
# Check for broken links
npm install -g markdown-link-check
markdown-link-check docs/**/*.md
```

---

## Conclusion

**Current Recommendation**: Keep the flat structure with the new INDEX.md navigation system. This provides:
- Easy navigation via INDEX.md
- Simple file organization
- No breaking changes
- Easy maintenance

**Future**: Reorganize into feature-based structure when documentation grows beyond 50 files or when managing multiple contributors.

---

**Last Updated**: 2025-10-15
**Version**: 1.0
