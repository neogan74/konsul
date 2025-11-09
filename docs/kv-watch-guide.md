# KV Watch/Subscribe Guide

The Konsul KV Watch feature allows you to subscribe to real-time notifications when key-value pairs change. This enables building reactive applications that respond immediately to configuration changes.

## Table of Contents

- [Overview](#overview)
- [Supported Transports](#supported-transports)
- [Pattern Matching](#pattern-matching)
- [Authentication & ACL](#authentication--acl)
- [Usage Examples](#usage-examples)
  - [CLI (konsulctl)](#cli-konsulctl)
  - [JavaScript (WebSocket)](#javascript-websocket)
  - [JavaScript (Server-Sent Events)](#javascript-server-sent-events)
  - [Go (WebSocket)](#go-websocket)
  - [curl (SSE)](#curl-sse)
- [Event Format](#event-format)
- [Configuration](#configuration)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## Overview

The watch feature provides real-time notifications for KV store changes through two transport mechanisms:
- **WebSocket**: Bidirectional, persistent connection for streaming updates
- **Server-Sent Events (SSE)**: Unidirectional, HTTP-based streaming

## Supported Transports

### WebSocket
- **Endpoint**: `GET /kv/watch/:key`
- **Protocol**: `ws://` or `wss://` (TLS)
- **Header**: `Upgrade: websocket`
- **Best for**: Bidirectional communication, browser applications, high-frequency updates

### Server-Sent Events (SSE)
- **Endpoint**: `GET /kv/watch/:key`
- **Protocol**: `http://` or `https://`
- **Header**: `Accept: text/event-stream`
- **Best for**: Server-to-client streaming, simpler implementation, HTTP-only environments

## Pattern Matching

Watch supports three pattern types:

| Pattern | Example | Matches | Doesn't Match |
|---------|---------|---------|---------------|
| **Exact** | `app/config` | `app/config` only | `app/config/db`, `app/data` |
| **Single-level** | `app/*` | `app/config`, `app/data` | `app/config/db` |
| **Multi-level** | `app/**` | `app/config`, `app/config/db`, `app/data/cache` | `other/config` |

## Authentication & ACL

### Authentication

When authentication is enabled, include a JWT token:

**WebSocket**:
```javascript
const ws = new WebSocket('ws://localhost:8888/kv/watch/app/config', {
  headers: { 'Authorization': 'Bearer ' + token }
});
```

**SSE**:
```javascript
const eventSource = new EventSource('http://localhost:8888/kv/watch/app/config', {
  headers: { 'Authorization': 'Bearer ' + token }
});
```

### ACL Permissions

To watch a key pattern, you need `read` permission:

```json
{
  "name": "watch-config",
  "kv": [
    {
      "path": "app/config/*",
      "capabilities": ["read"]
    }
  ]
}
```

**Important**: You'll only receive events for keys you have permission to read. If watching `app/**` but only have permission for `app/config/*`, you'll only get events for `app/config/*` keys.

## Usage Examples

### CLI (konsulctl)

#### Watch a single key

```bash
konsulctl kv watch app/config
```

#### Watch with pattern

```bash
konsulctl kv watch 'app/*'
konsulctl kv watch 'app/**'
```

#### Watch with SSE transport

```bash
konsulctl kv watch app/config --transport=sse
```

#### Watch with authentication

```bash
export KONSUL_TOKEN="your-jwt-token"
konsulctl kv watch app/config --server=https://konsul.example.com
```

#### Watch with TLS

```bash
konsulctl kv watch app/config \
  --server=https://localhost:8888 \
  --ca-cert=/path/to/ca.crt \
  --client-cert=/path/to/client.crt \
  --client-key=/path/to/client.key
```

**Example Output**:
```
Watching app/config (WebSocket)...
Press Ctrl+C to stop

[2025-01-08 15:30:45] CREATE app/config: production-mode
[2025-01-08 15:31:12] UPDATE app/config: production-mode -> debug-mode
[2025-01-08 15:32:05] DELETE app/config (was: debug-mode)
```

### JavaScript (WebSocket)

```javascript
// Create WebSocket connection
const ws = new WebSocket('ws://localhost:8888/kv/watch/app/config');

// Connection opened
ws.addEventListener('open', (event) => {
  console.log('Connected to watch endpoint');
});

// Listen for messages
ws.addEventListener('message', (event) => {
  const watchEvent = JSON.parse(event.data);

  switch (watchEvent.type) {
    case 'set':
      if (watchEvent.old_value) {
        console.log(`Updated ${watchEvent.key}: ${watchEvent.old_value} → ${watchEvent.value}`);
      } else {
        console.log(`Created ${watchEvent.key}: ${watchEvent.value}`);
      }
      break;

    case 'delete':
      console.log(`Deleted ${watchEvent.key} (was: ${watchEvent.old_value})`);
      break;
  }
});

// Listen for errors
ws.addEventListener('error', (error) => {
  console.error('WebSocket error:', error);
});

// Connection closed
ws.addEventListener('close', (event) => {
  console.log('WebSocket connection closed');
});
```

#### With Authentication

```javascript
const ws = new WebSocket('ws://localhost:8888/kv/watch/app/config', {
  headers: {
    'Authorization': 'Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...'
  }
});
```

#### Watch Pattern

```javascript
// Watch all keys under app/
const ws = new WebSocket('ws://localhost:8888/kv/watch/app/**');

ws.addEventListener('message', (event) => {
  const watchEvent = JSON.parse(event.data);
  console.log(`Change detected: ${watchEvent.key}`);
});
```

### JavaScript (Server-Sent Events)

```javascript
// Create EventSource connection
const eventSource = new EventSource('http://localhost:8888/kv/watch/app/config');

// Listen for kv-change events
eventSource.addEventListener('kv-change', (event) => {
  const watchEvent = JSON.parse(event.data);

  console.log(`Event: ${watchEvent.type}`);
  console.log(`Key: ${watchEvent.key}`);
  console.log(`Value: ${watchEvent.value}`);
  console.log(`Timestamp: ${new Date(watchEvent.timestamp * 1000)}`);
});

// Listen for errors
eventSource.addEventListener('error', (error) => {
  console.error('SSE error:', error);
  if (eventSource.readyState === EventSource.CLOSED) {
    console.log('Connection closed');
  }
});

// Close connection when done
// eventSource.close();
```

#### With Authentication

```javascript
// Note: EventSource doesn't support custom headers in standard browsers
// Use a library like eventsource-polyfill or fetch with ReadableStream

import { EventSourcePolyfill } from 'eventsource';

const eventSource = new EventSourcePolyfill('http://localhost:8888/kv/watch/app/config', {
  headers: {
    'Authorization': 'Bearer your-jwt-token'
  }
});
```

#### React Hook Example

```javascript
import { useEffect, useState } from 'react';

function useKVWatch(key) {
  const [value, setValue] = useState(null);
  const [error, setError] = useState(null);

  useEffect(() => {
    const ws = new WebSocket(`ws://localhost:8888/kv/watch/${key}`);

    ws.addEventListener('message', (event) => {
      const watchEvent = JSON.parse(event.data);
      if (watchEvent.type === 'set') {
        setValue(watchEvent.value);
      } else if (watchEvent.type === 'delete') {
        setValue(null);
      }
    });

    ws.addEventListener('error', (err) => {
      setError(err);
    });

    return () => {
      ws.close();
    };
  }, [key]);

  return { value, error };
}

// Usage
function App() {
  const { value, error } = useKVWatch('app/config');

  return (
    <div>
      <h1>Config Value: {value}</h1>
      {error && <p>Error: {error.message}</p>}
    </div>
  );
}
```

### Go (WebSocket)

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/url"
    "os"
    "os/signal"

    "github.com/gorilla/websocket"
)

type WatchEvent struct {
    Type      string `json:"type"`
    Key       string `json:"key"`
    Value     string `json:"value,omitempty"`
    OldValue  string `json:"old_value,omitempty"`
    Timestamp int64  `json:"timestamp"`
}

func main() {
    // Connect to watch endpoint
    u := url.URL{Scheme: "ws", Host: "localhost:8888", Path: "/kv/watch/app/config"}
    log.Printf("Connecting to %s", u.String())

    c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
    if err != nil {
        log.Fatal("dial:", err)
    }
    defer c.Close()

    // Setup signal handler for graceful shutdown
    interrupt := make(chan os.Signal, 1)
    signal.Notify(interrupt, os.Interrupt)

    done := make(chan struct{})

    // Read events in goroutine
    go func() {
        defer close(done)
        for {
            var event WatchEvent
            err := c.ReadJSON(&event)
            if err != nil {
                log.Println("read:", err)
                return
            }

            switch event.Type {
            case "set":
                if event.OldValue != "" {
                    log.Printf("UPDATE %s: %s → %s\n", event.Key, event.OldValue, event.Value)
                } else {
                    log.Printf("CREATE %s: %s\n", event.Key, event.Value)
                }
            case "delete":
                log.Printf("DELETE %s (was: %s)\n", event.Key, event.OldValue)
            }
        }
    }()

    // Wait for interrupt
    <-interrupt
    log.Println("Shutting down...")

    // Cleanly close the connection
    err = c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
    if err != nil {
        log.Println("write close:", err)
        return
    }

    <-done
}
```

#### With Authentication

```go
import "net/http"

// Add authentication header
header := http.Header{}
header.Add("Authorization", "Bearer your-jwt-token")

c, _, err := websocket.DefaultDialer.Dial(u.String(), header)
```

#### Watch Pattern

```go
// Watch all keys under app/
u := url.URL{Scheme: "ws", Host: "localhost:8888", Path: "/kv/watch/app/**"}
```

### curl (SSE)

```bash
# Watch a single key
curl -N -H "Accept: text/event-stream" \
  http://localhost:8888/kv/watch/app/config

# With authentication
curl -N -H "Accept: text/event-stream" \
  -H "Authorization: Bearer your-jwt-token" \
  http://localhost:8888/kv/watch/app/config

# Watch pattern
curl -N -H "Accept: text/event-stream" \
  http://localhost:8888/kv/watch/app/**
```

**Example Output**:
```
event: kv-change
data: {"type":"set","key":"app/config","value":"production","timestamp":1704723045}

event: kv-change
data: {"type":"set","key":"app/config","value":"debug","old_value":"production","timestamp":1704723072}

event: kv-change
data: {"type":"delete","key":"app/config","old_value":"debug","timestamp":1704723105}
```

## Event Format

All watch events follow this JSON structure:

```json
{
  "type": "set",
  "key": "app/config/database",
  "value": "postgres://localhost:5432/mydb",
  "old_value": "postgres://localhost:5432/olddb",
  "timestamp": 1704723045
}
```

### Fields

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Event type: `"set"` or `"delete"` |
| `key` | string | The key that changed |
| `value` | string | New value (only for `"set"` events) |
| `old_value` | string | Previous value (optional) |
| `timestamp` | int64 | Unix timestamp of the change |

### Event Types

#### `set` Event
Triggered when a key is created or updated.

```json
{
  "type": "set",
  "key": "app/config",
  "value": "new-value",
  "timestamp": 1704723045
}
```

With previous value (update):
```json
{
  "type": "set",
  "key": "app/config",
  "value": "new-value",
  "old_value": "old-value",
  "timestamp": 1704723045
}
```

#### `delete` Event
Triggered when a key is deleted.

```json
{
  "type": "delete",
  "key": "app/config",
  "old_value": "previous-value",
  "timestamp": 1704723105
}
```

## Configuration

Configure watch behavior via environment variables:

```bash
# Enable/disable watch feature (default: true)
KONSUL_WATCH_ENABLED=true

# Event buffer size per watcher (default: 100)
KONSUL_WATCH_BUFFER_SIZE=100

# Max watchers per client (default: 100)
KONSUL_WATCH_MAX_PER_CLIENT=100
```

## Best Practices

### 1. Use Specific Patterns

Watch the most specific pattern needed to reduce unnecessary events:

✅ **Good**: `app/config/database` or `app/config/*`
❌ **Bad**: `**` (watches everything)

### 2. Handle Reconnections

Connections can drop. Always implement reconnection logic:

```javascript
function connectWatch(key) {
  const ws = new WebSocket(`ws://localhost:8888/kv/watch/${key}`);

  ws.addEventListener('close', () => {
    console.log('Connection closed, reconnecting in 5s...');
    setTimeout(() => connectWatch(key), 5000);
  });

  return ws;
}
```

### 3. Debounce Rapid Changes

If you're updating values rapidly, debounce your handlers:

```javascript
let debounceTimer;
ws.addEventListener('message', (event) => {
  clearTimeout(debounceTimer);
  debounceTimer = setTimeout(() => {
    handleWatchEvent(JSON.parse(event.data));
  }, 300);
});
```

### 4. Limit Watchers Per Client

Don't create too many watchers. The default limit is 100 per client.

### 5. Close Connections When Done

Always close connections you're not using:

```javascript
// WebSocket
ws.close();

// SSE
eventSource.close();
```

### 6. Use Buffer Size Appropriately

If events are being dropped (check `konsul_watch_events_dropped_total` metric), increase the buffer size:

```bash
KONSUL_WATCH_BUFFER_SIZE=500
```

## Troubleshooting

### Events Not Received

1. **Check ACL permissions**: Ensure you have `read` permission for the key pattern
2. **Verify authentication**: Include valid JWT token in Authorization header
3. **Check buffer**: Events may be dropped if buffer is full

### Connection Drops

1. **Network issues**: Implement reconnection logic
2. **Idle timeout**: Send periodic pings (WebSocket automatically handles this)
3. **Server restart**: Handle graceful disconnection

### Too Many Watchers

```
Error: too many watchers for this client
```

**Solution**: Reduce number of watchers or increase `KONSUL_WATCH_MAX_PER_CLIENT`

### Permission Denied

```
Error: insufficient permissions to watch this key pattern
```

**Solution**: Update ACL policy to grant `read` permission:

```json
{
  "kv": [
    {"path": "app/config/*", "capabilities": ["read"]}
  ]
}
```

## Metrics

Monitor watch system health with Prometheus metrics:

```promql
# Active watchers by transport
konsul_watchers_active{transport="websocket"}
konsul_watchers_active{transport="sse"}

# Events sent
rate(konsul_watch_events_total[5m])

# Dropped events (indicates buffer issues)
rate(konsul_watch_events_dropped_total[5m])

# Connection lifecycle
rate(konsul_watch_connections_total{status="opened"}[5m])
rate(konsul_watch_connections_total{status="closed"}[5m])

# ACL denials
rate(konsul_watch_acl_denials_total[5m])
```

## See Also

- [ACL Guide](acl-guide.md) - Configure access control
- [Architecture Decision Record: KV Watch](adr/0011-kv-watch-subscribe.md) - Design rationale
- [Metrics Documentation](METRICS.md) - Monitor watch performance
