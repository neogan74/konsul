# ADR-0028: Edge Computing & IoT Strategy

**Date**: 2025-12-06

**Status**: Proposed

**Deciders**: Konsul Core Team

**Tags**: edge-computing, iot, lightweight, offline, mqtt, crdt

## Context

IoT and edge computing require service discovery and configuration management at the edge, but traditional cloud-centric architectures face challenges:

### Edge/IoT Requirements

**Constraints**:
- Limited resources (64-256MB RAM, ARM processors)
- Intermittent connectivity (offline periods)
- Low bandwidth (<1 Mbps)
- Battery-powered devices
- Thousands of devices per deployment

**Use Cases**:
- **Retail**: Store devices, POS systems, inventory scanners
- **Manufacturing**: Factory floor sensors, robotics, edge controllers
- **Smart Cities**: Traffic sensors, cameras, environmental monitoring
- **Agriculture**: Soil sensors, automated irrigation, drones
- **Healthcare**: Medical devices, patient monitors, diagnostics

## Decision

Implement **Lightweight Edge Nodes** (<10MB footprint) with **offline-first** architecture and **MQTT integration** for IoT devices.

### Architecture

```
┌──────────────────────────────────────────────────────────┐
│           Edge Computing Architecture                     │
├──────────────────────────────────────────────────────────┤
│                                                           │
│  ┌────────── Cloud Cluster ──────────┐                   │
│  │                                   │                   │
│  │  ┌───────────────────────────┐   │                   │
│  │  │ Konsul Server Cluster     │   │                   │
│  │  │ (5 nodes, Full feature)   │   │                   │
│  │  └───────────┬───────────────┘   │                   │
│  │              │ Sync              │                   │
│  └──────────────┼────────────────────┘                   │
│                 │ (Occasional)                           │
│                 ▼                                        │
│  ┌──────────────────────────────────────────┐           │
│  │   Edge Nodes (Lightweight)              │           │
│  │   ┌────────┐  ┌────────┐  ┌────────┐   │           │
│  │   │ Edge-1 │  │ Edge-2 │  │ Edge-N │   │           │
│  │   │ <10MB  │  │ <10MB  │  │ <10MB  │   │           │
│  │   │ Offline│  │ Offline│  │ Offline│   │           │
│  │   └───┬────┘  └───┬────┘  └───┬────┘   │           │
│  │       │            │            │       │           │
│  │       │ MQTT       │ HTTP       │ gRPC  │           │
│  │       ▼            ▼            ▼       │           │
│  │  ┌────────┐  ┌────────┐  ┌────────┐   │           │
│  │  │IoT     │  │Sensors │  │Edge    │   │           │
│  │  │Devices │  │        │  │Services│   │           │
│  │  └────────┘  └────────┘  └────────┘   │           │
│  └──────────────────────────────────────────┘           │
│                                                          │
└──────────────────────────────────────────────────────────┘
```

### Key Components

**1. Lightweight Edge Node**:
- Binary size <10MB (stripped build)
- Memory footprint <50MB
- Embedded storage (SQLite)
- Offline-capable
- ARM/ARM64 support

**2. Offline-First Sync**:
- Local-first operations
- Queue changes during offline
- Batch sync when connected
- Conflict resolution (CRDTs)
- Eventual consistency

**3. MQTT Bridge**:
- MQTT broker integration
- Pub/sub for IoT devices
- QoS support (0, 1, 2)
- Lightweight protocol
- Battery-efficient

**4. Device Registry**:
- Device metadata
- Firmware versions
- OTA update tracking
- Telemetry collection
- Health monitoring

### Edge Node Build

**Minimal Build Configuration**:

```bash
# Build lightweight edge binary
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build \
  -ldflags="-s -w" \
  -tags=edge,minimal \
  -o konsul-edge \
  ./cmd/konsul-edge

# Result: ~8MB binary
# Optimizations:
# - No debug symbols (-s -w)
# - Minimal dependencies
# - Embedded storage only
# - No UI components
# - Stripped-down features
```

**Edge Configuration**:

```yaml
# edge-config.yaml
mode: edge

# Resource limits
resources:
  memory_limit: 50Mi
  cpu_limit: 0.1  # 100 millicores
  storage_limit: 500Mi

# Cloud connection
cloud:
  server_address: "https://konsul-cloud.example.com:8500"
  sync_interval: 300s  # 5 minutes
  offline_queue_size: 10000
  retry_backoff: exponential

# Local storage
storage:
  type: sqlite
  path: /data/konsul.db
  max_size: 100Mi

# MQTT configuration
mqtt:
  enabled: true
  broker: tcp://localhost:1883
  client_id: konsul-edge-1
  qos: 1
  topics:
    - devices/+/telemetry
    - devices/+/events

# Services
services:
  max_local_services: 100
  cache_ttl: 3600s

# Sync strategy
sync:
  mode: delta  # Only sync changes
  compression: true
  conflict_resolution: last-write-wins  # or crdt
```

### Offline-First Synchronization

**CRDT-Based Conflict Resolution**:

```go
package edge

import (
    "github.com/hashicorp/go-memdb"
)

// CRDT (Conflict-Free Replicated Data Type)
type ServiceCRDT struct {
    ID            string
    Name          string
    Address       string
    VectorClock   map[string]uint64  // For causality tracking
    Tombstone     bool               // For deletions
    LastModified  time.Time
}

func (e *EdgeNode) MergeFromCloud(cloudServices []ServiceCRDT) error {
    for _, cloudSvc := range cloudServices {
        localSvc, exists := e.getLocalService(cloudSvc.ID)

        if !exists {
            // New service from cloud
            e.addService(cloudSvc)
            continue
        }

        // Conflict resolution using vector clocks
        if cloudSvc.CausesAfter(localSvc) {
            // Cloud version is newer
            e.updateService(cloudSvc)
        } else if localSvc.CausesAfter(cloudSvc) {
            // Local version is newer - queue for upload
            e.queueForSync(localSvc)
        } else {
            // Concurrent updates - use LWW (Last-Write-Wins)
            if cloudSvc.LastModified.After(localSvc.LastModified) {
                e.updateService(cloudSvc)
            }
        }
    }
    return nil
}
```

**Sync Protocol**:

```protobuf
syntax = "proto3";

package konsul.edge;

// Edge Sync Request
message EdgeSyncRequest {
    string edge_id = 1;
    int64 last_sync_index = 2;
    repeated ServiceUpdate pending_updates = 3;
    map<string, uint64> vector_clock = 4;
}

// Edge Sync Response
message EdgeSyncResponse {
    int64 current_index = 1;
    repeated ServiceUpdate cloud_updates = 2;
    repeated string conflicts = 3;  // Service IDs with conflicts
    map<string, uint64> server_vector_clock = 4;
}
```

### MQTT Integration

**Device Communication**:

```go
package edge

import (
    mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MQTTBridge struct {
    client mqtt.Client
    edge   *EdgeNode
}

func (m *MQTTBridge) Start() error {
    // Subscribe to device topics
    m.client.Subscribe("devices/+/register", 1, m.handleDeviceRegister)
    m.client.Subscribe("devices/+/telemetry", 0, m.handleTelemetry)
    m.client.Subscribe("devices/+/heartbeat", 0, m.handleHeartbeat)

    return nil
}

func (m *MQTTBridge) handleDeviceRegister(client mqtt.Client, msg mqtt.Message) {
    var device Device
    json.Unmarshal(msg.Payload(), &device)

    // Register device as service in edge node
    service := &Service{
        ID:      device.ID,
        Name:    device.Type,
        Address: device.IP,
        Tags:    []string{"iot", "mqtt"},
        Meta:    device.Metadata,
    }

    m.edge.RegisterService(service)

    // Publish acknowledgment
    ack := map[string]string{"status": "registered", "id": device.ID}
    ackJSON, _ := json.Marshal(ack)
    client.Publish(fmt.Sprintf("devices/%s/ack", device.ID), 1, false, ackJSON)
}
```

**Device Registration Flow**:

```
1. IoT Device → MQTT Publish → devices/{id}/register
2. Edge Node → Receives via MQTT subscription
3. Edge Node → Registers device locally
4. Edge Node → Queue for cloud sync (when online)
5. Edge Node → Publishes ack to devices/{id}/ack
6. IoT Device → Receives confirmation
```

### Device Registry

**Device Model**:

```go
type Device struct {
    ID           string
    Type         string  // "sensor", "camera", "controller"
    Name         string
    Location     string
    Metadata     map[string]string

    // Firmware
    FirmwareVersion string
    TargetVersion   string
    UpdateStatus    string  // "pending", "downloading", "installing", "complete"

    // Health
    LastSeen     time.Time
    BatteryLevel int
    SignalStrength int

    // Telemetry
    TelemetryInterval time.Duration
    LastTelemetry     map[string]interface{}
}
```

**OTA Update Management**:

```go
func (e *EdgeNode) DeployFirmwareUpdate(deviceID, version string) error {
    device := e.devices.Get(deviceID)

    // Download firmware from cloud (when online)
    firmware, err := e.downloadFirmware(version)
    if err != nil {
        return err
    }

    // Store locally for offline deployment
    e.storage.SaveFirmware(version, firmware)

    // Notify device via MQTT
    updateMsg := FirmwareUpdate{
        Version:  version,
        URL:      fmt.Sprintf("http://edge-node/firmware/%s", version),
        Checksum: calculateChecksum(firmware),
    }

    e.mqtt.Publish(
        fmt.Sprintf("devices/%s/update", deviceID),
        1,
        false,
        updateMsg,
    )

    return nil
}
```

### Deployment Patterns

**Pattern 1: Factory Edge Node**

```yaml
# docker-compose-edge.yaml
version: '3.8'
services:
  konsul-edge:
    image: konsul/edge:latest
    restart: always
    volumes:
      - edge-data:/data
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      - KONSUL_MODE=edge
      - KONSUL_CLOUD_ADDRESS=https://konsul-cloud.example.com:8500
      - KONSUL_MQTT_ENABLED=true
    ports:
      - "8500:8500"
      - "8502:8502"  # Agent API

  mqtt-broker:
    image: eclipse-mosquitto:2
    ports:
      - "1883:1883"
    volumes:
      - mosquitto-data:/mosquitto/data

volumes:
  edge-data:
  mosquitto-data:
```

**Pattern 2: Raspberry Pi Edge Node**

```bash
# Install on Raspberry Pi
curl -fsSL https://get.konsul.io/edge | sh

# Configure
konsul-edge init \
  --cloud-address https://konsul-cloud.example.com:8500 \
  --mqtt-broker tcp://localhost:1883

# Start as systemd service
sudo systemctl enable konsul-edge
sudo systemctl start konsul-edge
```

### Resource Optimization

**Memory Optimization**:
- Embedded SQLite (vs BadgerDB for cloud)
- LRU cache with strict limits
- Periodic garbage collection
- Memory-mapped file I/O

**Network Optimization**:
- Delta sync (only changes)
- Compression (Snappy)
- Batch updates
- Adaptive sync interval (based on bandwidth)

**Storage Optimization**:
- Prune old telemetry data
- Compress historical data
- Configurable retention policies

### Performance Characteristics

**Resource Usage**:
- Binary: 8-10MB
- RAM: 30-50MB (idle), 50-80MB (active)
- Storage: 50-500MB (depends on cache)
- CPU: <5% (ARM Cortex-A53)

**Network**:
- Sync bandwidth: 10-100 KB/sec
- MQTT traffic: 1-10 KB/sec
- Offline queue: up to 10,000 events

**Latency**:
- Local operations: <1ms
- MQTT publish: <5ms
- Cloud sync: 100-1000ms (depends on network)

### Implementation Phases

**Phase 1: Lightweight Build (3 weeks)**
1. Minimal build configuration
2. Embedded SQLite storage
3. Resource limits
4. ARM/ARM64 cross-compilation

**Phase 2: Offline Sync (4 weeks)**
1. Offline queue implementation
2. Delta sync protocol
3. CRDT conflict resolution
4. Batch upload/download

**Phase 3: MQTT Integration (3 weeks)**
1. MQTT broker connection
2. Device registration via MQTT
3. Telemetry collection
4. Bidirectional communication

**Phase 4: Device Management (4 weeks)**
1. Device registry
2. Firmware OTA updates
3. Health monitoring
4. Alert generation

**Total**: 14 weeks (~3.5 months)

## Alternatives Considered

### Alternative 1: Full Cloud Node at Edge
- **Reason for rejection**: Too resource-intensive (>500MB RAM)

### Alternative 2: Edge-Only (No Cloud)
- **Reason for rejection**: Limits observability and central management

### Alternative 3: CoAP Instead of MQTT
- **Reason for rejection**: MQTT more widely adopted for IoT

### Alternative 4: Always-Online (No Offline Mode)
- **Reason for rejection**: Edge environments have unreliable connectivity

## Consequences

### Positive
- **IoT support** with MQTT
- **Offline operations** during network outage
- **Resource efficient** (<10MB, <50MB RAM)
- **ARM support** for edge devices
- **OTA updates** for device management
- **Eventual consistency** via CRDTs

### Negative
- **Eventual consistency** (not strong consistency)
- **Limited features** vs cloud
- **Sync complexity** (conflict resolution)
- **Additional testing** (offline scenarios)

## References

- [ARCHITECTURE_USE_CASES.md](../ARCHITECTURE_USE_CASES.md#scenario-5-edgeiot-deployment)
- [MQTT Protocol](https://mqtt.org/)
- [CRDTs](https://crdt.tech/)

---

## Revision History

| Date | Author | Changes |
|------|--------|---------|
| 2025-12-06 | Konsul Team | Initial version |