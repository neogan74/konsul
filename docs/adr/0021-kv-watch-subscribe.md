# ADR-0021: KV Store Watch/Subscribe System

**Date**: 2025-11-05

**Status**: Proposed

**Deciders**: Konsul Core Team

**Tags**: kv-store, watch, subscribe, real-time, websocket, sse

## Context

Currently, Konsul's KV store only supports synchronous CRUD operations. Applications need to poll the KV store to detect changes, which is inefficient and creates unnecessary load.

### Current Limitations

1. **No change notifications** - Clients must poll to detect updates
2. **Inefficient** - Polling wastes resources and increases latency
3. **No reactive patterns** - Can't build event-driven applications
4. **Missing key feature** - Consul and etcd both provide watch functionality
5. **Poor DX** - Developers have to implement polling logic

### Requirements

1. **Watch specific keys** - Subscribe to changes on individual keys
2. **Watch key prefixes** - Subscribe to changes on keys matching a prefix (e.g., `app/config/*`)
3. **Initial value** - Provide current value when watch starts
4. **Change events** - Notify on set, delete, and modify operations
5. **Multiple transports** - Support WebSocket and Server-Sent Events (SSE)
6. **ACL integration** - Respect ACL policies (only notify if user can read)
7. **Performance** - Handle thousands of concurrent watchers efficiently
8. **Reliability** - Handle disconnections, reconnections, and missed events
9. **API design** - Simple, intuitive API for clients

## Decision

We will implement a **Watch/Subscribe system** with the following design:

### Architecture

```
┌─────────────┐
│   Client    │
│  (Browser/  │
│   Go/CLI)   │
└─────┬───────┘
      │
      │ WebSocket/SSE
      │
┌─────▼───────────────────────────────────────┐
│        Fiber HTTP Handler                   │
│  /kv/watch/:key (WebSocket)                 │
│  /kv/watch/:key (SSE - Accept: text/event) │
└─────┬───────────────────────────────────────┘
      │
┌─────▼──────────────────────────────────────┐
│         WatchManager                        │
│  - Manages all active watchers              │
│  - Routes events to appropriate subscribers │
│  - Handles ACL checks                       │
│  - Tracks connections                       │
└─────┬──────────────────────────────────────┘
      │
┌─────▼──────────────────────────────────────┐
│         KVStore (Enhanced)                  │
│  - Notifies WatchManager on changes        │
│  - Set/Delete trigger events                │
└────────────────────────────────────────────┘
```

### Data Structures

```go
// WatchEvent represents a change to a key
type WatchEvent struct {
    Type      EventType `json:"type"`       // "set", "delete"
    Key       string    `json:"key"`
    Value     string    `json:"value,omitempty"`
    OldValue  string    `json:"old_value,omitempty"`
    Timestamp int64     `json:"timestamp"`
}

type EventType string

const (
    EventTypeSet    EventType = "set"
    EventTypeDelete EventType = "delete"
)

// Watcher represents a single watch subscription
type Watcher struct {
    ID        string
    Pattern   string           // Key or prefix to watch (supports * and **)
    Events    chan WatchEvent  // Buffered channel for events
    ACLPolicies []string       // Policies for ACL checks
    CreatedAt time.Time
    Transport TransportType    // "websocket" or "sse"
}

type TransportType string

const (
    TransportWebSocket TransportType = "websocket"
    TransportSSE       TransportType = "sse"
)

// WatchManager manages all active watchers
type WatchManager struct {
    watchers   map[string]*Watcher  // ID -> Watcher
    patterns   map[string][]string  // Pattern -> Watcher IDs
    mu         sync.RWMutex
    aclEval    *acl.Evaluator
    log        logger.Logger
}
```

### API Design

#### WebSocket API

**Endpoint**: `GET /kv/watch/:key`

**Connection upgrade**: Client sends WebSocket handshake

**Flow**:
1. Client connects with `Upgrade: websocket`
2. Server sends initial value (if key exists)
3. Server sends events as they occur
4. Client can close connection anytime

**Example**:
```javascript
// JavaScript client
const ws = new WebSocket('ws://localhost:8888/kv/watch/app/config', {
  headers: { 'Authorization': 'Bearer ' + token }
});

ws.onmessage = (event) => {
  const watchEvent = JSON.parse(event.data);
  console.log('Key changed:', watchEvent);
  // {type: "set", key: "app/config", value: "new-value", timestamp: 1234567890}
};

ws.onerror = (error) => console.error('WebSocket error:', error);
ws.onclose = () => console.log('Connection closed');
```

**Messages**:
```json
// Initial value (sent immediately on connect)
{
  "type": "set",
  "key": "app/config",
  "value": "current-value",
  "timestamp": 1699200000
}

// Update event
{
  "type": "set",
  "key": "app/config",
  "value": "new-value",
  "old_value": "current-value",
  "timestamp": 1699200100
}

// Delete event
{
  "type": "delete",
  "key": "app/config",
  "old_value": "new-value",
  "timestamp": 1699200200
}
```

#### Server-Sent Events (SSE) API

**Endpoint**: `GET /kv/watch/:key`

**Headers**: `Accept: text/event-stream`

**Flow**:
1. Client connects with `Accept: text/event-stream`
2. Server keeps connection open
3. Server sends events as `data:` lines
4. Client can close connection anytime

**Example**:
```javascript
// JavaScript client
const eventSource = new EventSource(
  'http://localhost:8888/kv/watch/app/config',
  { headers: { 'Authorization': 'Bearer ' + token }}
);

eventSource.addEventListener('kv-change', (e) => {
  const watchEvent = JSON.parse(e.data);
  console.log('Key changed:', watchEvent);
});

eventSource.onerror = (error) => {
  console.error('SSE error:', error);
  eventSource.close();
};
```

**Messages**:
```
event: kv-change
data: {"type":"set","key":"app/config","value":"current-value","timestamp":1699200000}

event: kv-change
data: {"type":"set","key":"app/config","value":"new-value","old_value":"current-value","timestamp":1699200100}

event: kv-change
data: {"type":"delete","key":"app/config","old_value":"new-value","timestamp":1699200200}
```

#### Prefix Watching

Watch all keys matching a prefix:

```
# Watch all keys under app/config/
GET /kv/watch/app/config/*

# Watch all keys recursively under app/
GET /kv/watch/app/**
```

**Events for prefix watches**:
```json
{
  "type": "set",
  "key": "app/config/database",
  "value": "postgres://localhost",
  "timestamp": 1699200000
}

{
  "type": "set",
  "key": "app/config/redis",
  "value": "redis://localhost",
  "timestamp": 1699200100
}
```

### Pattern Matching

Support same pattern matching as ACL system:

- **Exact match**: `app/config` - matches only `app/config`
- **Single-level wildcard**: `app/*` - matches `app/config`, `app/data`, but not `app/config/nested`
- **Multi-level wildcard**: `app/**` - matches all keys under `app/`

### ACL Integration

1. **Watch requires `read` capability** on the watched key/pattern
2. **Events filtered by ACL** - Only send events for keys user can read
3. **Pattern expansion respects ACL** - Don't reveal keys user can't access

**Example**:
```
User has policy:
  kv: [
    {path: "app/config/*", capabilities: ["read"]},
    {path: "app/secrets/*", capabilities: ["deny"]}
  ]

# Allowed
ws://localhost:8888/kv/watch/app/config/database  ✓

# Denied (403 Forbidden on connection)
ws://localhost:8888/kv/watch/app/secrets/password  ✗

# Partial events (only app/config/* events sent)
ws://localhost:8888/kv/watch/app/**  ✓ (filtered)
```

### Enhanced KV Store

```go
// Add to KVStore
type KVStore struct {
    Data         map[string]string
    Mutex        sync.RWMutex
    engine       persistence.Engine
    log          logger.Logger
    watchManager *WatchManager  // NEW
}

// Modified Set method
func (kv *KVStore) Set(key, value string) {
    kv.Mutex.Lock()
    defer kv.Mutex.Unlock()

    oldValue, existed := kv.Data[key]
    kv.Data[key] = value

    // Persist
    if kv.engine != nil {
        if err := kv.engine.Set(key, []byte(value)); err != nil {
            kv.log.Error("Failed to persist key", logger.Error(err))
        }
    }

    // Notify watchers
    if kv.watchManager != nil {
        kv.watchManager.Notify(WatchEvent{
            Type:      EventTypeSet,
            Key:       key,
            Value:     value,
            OldValue:  oldValue,
            Timestamp: time.Now().Unix(),
        })
    }
}

// Modified Delete method
func (kv *KVStore) Delete(key string) {
    kv.Mutex.Lock()
    defer kv.Mutex.Unlock()

    oldValue, existed := kv.Data[key]
    delete(kv.Data, key)

    // Persist
    if kv.engine != nil {
        if err := kv.engine.Delete(key); err != nil {
            kv.log.Error("Failed to delete key", logger.Error(err))
        }
    }

    // Notify watchers
    if kv.watchManager != nil && existed {
        kv.watchManager.Notify(WatchEvent{
            Type:      EventTypeDelete,
            Key:       key,
            OldValue:  oldValue,
            Timestamp: time.Now().Unix(),
        })
    }
}
```

### WatchManager Implementation

```go
type WatchManager struct {
    watchers  map[string]*Watcher
    patterns  map[string][]string  // Pattern -> []WatcherID
    mu        sync.RWMutex
    aclEval   *acl.Evaluator
    log       logger.Logger
}

func (wm *WatchManager) AddWatcher(pattern string, policies []string, transport TransportType) (*Watcher, error) {
    wm.mu.Lock()
    defer wm.mu.Unlock()

    watcher := &Watcher{
        ID:          uuid.New().String(),
        Pattern:     pattern,
        Events:      make(chan WatchEvent, 100), // Buffered channel
        ACLPolicies: policies,
        CreatedAt:   time.Now(),
        Transport:   transport,
    }

    wm.watchers[watcher.ID] = watcher
    wm.patterns[pattern] = append(wm.patterns[pattern], watcher.ID)

    wm.log.Info("Watcher added",
        logger.String("id", watcher.ID),
        logger.String("pattern", pattern))

    return watcher, nil
}

func (wm *WatchManager) RemoveWatcher(id string) {
    wm.mu.Lock()
    defer wm.mu.Unlock()

    watcher, exists := wm.watchers[id]
    if !exists {
        return
    }

    // Remove from patterns map
    ids := wm.patterns[watcher.Pattern]
    for i, wid := range ids {
        if wid == id {
            wm.patterns[watcher.Pattern] = append(ids[:i], ids[i+1:]...)
            break
        }
    }

    // Close channel and remove
    close(watcher.Events)
    delete(wm.watchers, id)

    wm.log.Info("Watcher removed", logger.String("id", id))
}

func (wm *WatchManager) Notify(event WatchEvent) {
    wm.mu.RLock()
    defer wm.mu.RUnlock()

    // Find all watchers matching this key
    for pattern, watcherIDs := range wm.patterns {
        if wm.matchesPattern(event.Key, pattern) {
            for _, id := range watcherIDs {
                watcher := wm.watchers[id]

                // Check ACL permissions
                if !wm.canWatch(watcher, event.Key) {
                    continue // Skip this watcher
                }

                // Send event (non-blocking)
                select {
                case watcher.Events <- event:
                    // Event sent
                default:
                    // Channel full, log warning
                    wm.log.Warn("Watcher channel full, dropping event",
                        logger.String("watcher_id", id),
                        logger.String("key", event.Key))
                }
            }
        }
    }
}

func (wm *WatchManager) matchesPattern(key, pattern string) bool {
    // Use same pattern matching as ACL system
    // Exact match, *, and **
    // ... implementation ...
}

func (wm *WatchManager) canWatch(watcher *Watcher, key string) bool {
    // Check if watcher's policies allow reading this key
    resource := acl.NewKVResource(key)
    return wm.aclEval.Evaluate(watcher.ACLPolicies, resource, acl.CapabilityRead)
}
```

### Handler Implementation

```go
// WebSocket handler
func (h *KVHandler) WatchWebSocket(c *fiber.Ctx) error {
    key := c.Params("key")
    log := middleware.GetLogger(c)
    claims := middleware.GetClaims(c)

    // Check ACL permission
    resource := acl.NewKVResource(key)
    if !h.aclEval.Evaluate(claims.Policies, resource, acl.CapabilityRead) {
        return middleware.Forbidden(c, "insufficient permissions to watch this key")
    }

    // Upgrade to WebSocket
    return websocket.New(func(conn *websocket.Conn) {
        defer conn.Close()

        // Add watcher
        watcher, err := h.watchManager.AddWatcher(key, claims.Policies, TransportWebSocket)
        if err != nil {
            log.Error("Failed to add watcher", logger.Error(err))
            return
        }
        defer h.watchManager.RemoveWatcher(watcher.ID)

        // Send initial value
        if value, ok := h.store.Get(key); ok {
            initialEvent := WatchEvent{
                Type:      EventTypeSet,
                Key:       key,
                Value:     value,
                Timestamp: time.Now().Unix(),
            }
            conn.WriteJSON(initialEvent)
        }

        // Stream events
        for event := range watcher.Events {
            if err := conn.WriteJSON(event); err != nil {
                log.Error("Failed to write event", logger.Error(err))
                break
            }
        }
    })(c)
}

// SSE handler
func (h *KVHandler) WatchSSE(c *fiber.Ctx) error {
    key := c.Params("key")
    log := middleware.GetLogger(c)
    claims := middleware.GetClaims(c)

    // Check ACL permission
    resource := acl.NewKVResource(key)
    if !h.aclEval.Evaluate(claims.Policies, resource, acl.CapabilityRead) {
        return middleware.Forbidden(c, "insufficient permissions to watch this key")
    }

    // Set SSE headers
    c.Set("Content-Type", "text/event-stream")
    c.Set("Cache-Control", "no-cache")
    c.Set("Connection", "keep-alive")
    c.Set("X-Accel-Buffering", "no")

    // Add watcher
    watcher, err := h.watchManager.AddWatcher(key, claims.Policies, TransportSSE)
    if err != nil {
        return err
    }
    defer h.watchManager.RemoveWatcher(watcher.ID)

    // Send initial value
    if value, ok := h.store.Get(key); ok {
        initialEvent := WatchEvent{
            Type:      EventTypeSet,
            Key:       key,
            Value:     value,
            Timestamp: time.Now().Unix(),
        }
        c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
            fmt.Fprintf(w, "event: kv-change\ndata: %s\n\n", toJSON(initialEvent))
            w.Flush()
        })
    }

    // Stream events
    for event := range watcher.Events {
        fmt.Fprintf(c.Response().BodyWriter(), "event: kv-change\ndata: %s\n\n", toJSON(event))
        if err := c.Response().BodyWriter().(*bufio.Writer).Flush(); err != nil {
            break
        }
    }

    return nil
}
```

## Alternatives Considered

### Alternative 1: Long Polling

**Pros**:
- Simple HTTP, no WebSocket needed
- Works everywhere
- Easy to implement

**Cons**:
- Inefficient (constant reconnections)
- Higher latency
- More server load
- Poor user experience

**Reason for rejection**: WebSocket/SSE are more efficient and provide better UX

### Alternative 2: Webhook Callbacks

**Pros**:
- Server pushes to client
- No persistent connections
- Standard HTTP

**Cons**:
- Requires client to have public endpoint
- Complex NAT/firewall traversal
- Security challenges (verify callbacks)
- Not suitable for browsers

**Reason for rejection**: Not practical for many use cases (browsers, internal services)

### Alternative 3: gRPC Streaming

**Pros**:
- Efficient binary protocol
- Built-in streaming
- Strong typing

**Cons**:
- Requires gRPC client library
- Not browser-friendly (needs grpc-web)
- Adds complexity
- Overkill for simple watch

**Reason for rejection**: WebSocket/SSE are more widely supported and simpler

### Alternative 4: Redis Pub/Sub

**Pros**:
- Battle-tested
- Scalable
- Already know pattern

**Cons**:
- Requires external Redis
- Adds operational complexity
- Not integrated with ACL
- Overkill for single-node

**Reason for rejection**: Want embedded solution, can add Redis later for clustering

## Consequences

### Positive

- **Real-time reactivity** - Applications can react to changes immediately
- **Efficient** - No polling overhead
- **Scalable** - Can handle many concurrent watchers
- **ACL integrated** - Respects permissions automatically
- **Standard protocols** - WebSocket and SSE widely supported
- **Great DX** - Simple, intuitive API
- **Foundation for clustering** - Watch can extend to multi-node later

### Negative

- **Complexity** - Adds goroutines, channels, connection management
- **Memory usage** - Each watcher consumes memory
- **Testing overhead** - Need to test WebSocket/SSE connections
- **Debugging** - Harder to debug real-time connections
- **Reconnection logic** - Clients need to handle disconnections

### Neutral

- Need to choose buffer size for event channels
- Need metrics for watch statistics
- Need limits on max watchers per client

## Implementation Plan

### Phase 1: Core Watch System (Week 1)
- [ ] Implement `WatchManager` with pattern matching
- [ ] Add watch support to `KVStore` (notify on changes)
- [ ] Write unit tests for `WatchManager`

### Phase 2: WebSocket Transport (Week 1)
- [ ] Implement WebSocket handler
- [ ] Add ACL checks
- [ ] Handle connection lifecycle
- [ ] Write integration tests

### Phase 3: SSE Transport (Week 2)
- [ ] Implement SSE handler
- [ ] Add ACL checks
- [ ] Handle connection lifecycle
- [ ] Write integration tests

### Phase 4: CLI Support (Week 2)
- [ ] Add `konsulctl kv watch <key>` command
- [ ] Add `konsulctl kv watch --prefix <prefix>` for prefix watches
- [ ] Pretty-print events in terminal

### Phase 5: Metrics & Monitoring (Week 2)
- [ ] Add Prometheus metrics
  - `konsul_watchers_active{transport}` - Active watchers
  - `konsul_watch_events_total{key,event_type}` - Events sent
  - `konsul_watch_connections_total{transport,status}` - Connections
- [ ] Add logging for watch events
- [ ] Add health checks

### Phase 6: Documentation (Week 3)
- [ ] API documentation
- [ ] Client examples (Go, JavaScript, curl)
- [ ] Update README
- [ ] Add to Admin UI

## Metrics

```
# Active watchers by transport
konsul_watchers_active{transport="websocket"} 45
konsul_watchers_active{transport="sse"} 12

# Events sent
konsul_watch_events_total{event_type="set"} 1234
konsul_watch_events_total{event_type="delete"} 56

# Connection lifecycle
konsul_watch_connections_total{transport="websocket",status="opened"} 150
konsul_watch_connections_total{transport="websocket",status="closed"} 105
konsul_watch_connections_total{transport="sse",status="opened"} 30
konsul_watch_connections_total{transport="sse",status="closed"} 18

# ACL denials
konsul_watch_acl_denials_total{reason="no_permission"} 5

# Event channel metrics
konsul_watch_events_dropped_total{reason="channel_full"} 2
```

## Configuration

```bash
# Enable watch feature
KONSUL_WATCH_ENABLED=true

# Max watchers per client
KONSUL_WATCH_MAX_PER_CLIENT=100

# Event buffer size per watcher
KONSUL_WATCH_BUFFER_SIZE=100

# Connection timeout
KONSUL_WATCH_TIMEOUT=5m
```

## Future Enhancements

- [ ] Watch history (missed events while disconnected)
- [ ] Backpressure handling (slow consumers)
- [ ] Watch filters (only notify on specific conditions)
- [ ] Batch events (combine multiple changes)
- [ ] Cross-datacenter watch (cluster mode)
- [ ] GraphQL subscriptions integration

## References

- [Consul Watch](https://www.consul.io/docs/dynamic-app-config/watches)
- [etcd Watch](https://etcd.io/docs/latest/learning/api/#watch-api)
- [WebSocket RFC 6455](https://datatracker.ietf.org/doc/html/rfc6455)
- [Server-Sent Events](https://html.spec.whatwg.org/multipage/server-sent-events.html)
- [Fiber WebSocket](https://docs.gofiber.io/contrib/websocket/)

---

## Revision History

| Date | Author | Changes |
|------|--------|---------|
| 2025-11-05 | Konsul Team | Initial proposal |
