# Research: Go Technical Patterns for Namespace Implementation

## 1. Go Context Propagation in Fiber

**stdlib context vs Fiber context**:
- `fiber.Ctx` is NOT a `context.Context` — it wraps `fasthttp.RequestCtx`
- `c.Locals(key, value)` is Fiber's equivalent of `context.WithValue`
- For passing to non-Fiber code (store methods), extract and pass explicitly

**Pattern for namespace propagation**:
```go
// In middleware: store in Fiber locals
c.Locals("namespace", ns)

// In handler: extract and pass to store
ns := c.Locals("namespace").(string)
result, err := h.store.GetWithNamespace(ctx, ns, key)

// Alternative: wrap in a NamespacedStore at handler construction
store := NewNamespacedKVStore(h.kv, ns)
```

**Recommendation**: Pass namespace explicitly to store methods rather than threading through context — avoids hidden dependencies and is easier to test.

## 2. BadgerDB Key-Prefix Namespacing

**Key format**: `ns:<namespace>:<original-key>`
- Example: `ns:default:mykey`, `ns:team-a:config/db`
- Prefix scan: `db.NewIterator` with `Prefix: []byte("ns:team-a:")`

**Performance**: BadgerDB LSM tree — prefix scans are O(matches), not O(total keys). No perf concern.

**Existing data migration**:
```go
// Option A: On-read migration (lazy, dual-path)
func (s *KVStore) Get(ns, key string) ([]byte, error) {
    // Try namespaced key first
    val, err := s.db.Get([]byte("ns:" + ns + ":" + key))
    if errors.Is(err, badger.ErrKeyNotFound) && ns == "default" {
        // Fall back to legacy key (migration shim)
        return s.db.Get([]byte(key))
    }
    return val, err
}

// Option B: Startup migration (one-time, clean)
func MigrateToNamespacedKeys(db *badger.DB) error {
    // Scan all keys without "ns:" prefix, rewrite under "ns:default:"
}
```

**Recommendation**: Option B (startup migration) — cleaner, removes dual-path forever. Add `migrated` flag in BadgerDB metadata key.

## 3. Raft FSM Command Namespacing

**Backwards-compatible command extension**:
```go
// Before
type KVSetCommand struct {
    Key   string
    Value []byte
    TTL   time.Duration
}

// After — zero-value "default" namespace is backwards compatible
type KVSetCommand struct {
    Namespace string `json:"namespace,omitempty"` // "" → "default" in FSM
    Key       string `json:"key"`
    Value     []byte `json:"value"`
    TTL       time.Duration `json:"ttl,omitempty"`
}
```

**FSM Apply**: Normalize empty namespace to `"default"` at Apply boundary:
```go
func normalizeNS(ns string) string {
    if ns == "" { return "default" }
    return ns
}
```

**Raft log replay safety**: Old log entries without namespace field will deserialize with `""` → normalized to `"default"` ✓

## 4. Interface Extension Patterns

**Option A: Add namespace param to existing methods (breaking)**:
```go
type KVStore interface {
    Get(ns, key string) ([]byte, error)  // breaks all callers
}
```

**Option B: New namespace-aware interface, embed old one**:
```go
type KVStore interface { Get(key string) ([]byte, error) }

type NamespacedKVStore interface {
    KVStore  // embeds for compatibility
    GetInNamespace(ns, key string) ([]byte, error)
}
```

**Option C: Wrapper/decorator (preferred — no interface change)**:
```go
type NamespacedKV struct {
    store    KVStore
    namespace string
}
func (n *NamespacedKV) Get(key string) ([]byte, error) {
    return n.store.Get(n.namespace + ":" + key)
}
```

**Recommendation**: Option C — wrap existing store with namespace prefix logic. Existing callers unchanged; handlers construct a `NamespacedKV` from the request namespace.

**For new namespace-management API** (CRUD namespaces): new `NamespaceStore` interface entirely — no entanglement with existing stores.

## Summary of Recommended Patterns

| Concern | Pattern |
|---|---|
| Fiber middleware | `c.Locals("namespace", ns)` |
| Store calls | Explicit `ns` param or NamespacedKV wrapper |
| BadgerDB keys | `ns:<ns>:<key>` with startup migration |
| Raft commands | Add `Namespace string` with omitempty; normalize in FSM |
| Interface compat | Wrapper/decorator over existing store interfaces |
| Validation | Regex `^[a-z0-9][a-z0-9-]{0,62}$` at middleware boundary |

## Unresolved Questions
- Should NamespacedKV be constructed per-request or cached per-namespace?
- How to handle KV watch/subscribe across namespaces?
- Snapshot format: do snapshots include namespace prefix in keys, or is namespace a separate index?
