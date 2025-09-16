# Konsul service

In development now :> [!WARNING]

## KV storage (map with mutex)

| Method | endpoint  | Description  |
| ------ | --------- | ------------ |
| PUT    | /kv/<key> | Write value  |
| GET    | /kv/<key> | Read value   |
| DELETE | /kv/<key> | Delete value |

## Service Discovery (map)

| Method | endpoint  | Description                                |
| ------ | --------- | ------------                               |
| PUT    | /register | service registration                       |
| GET    | /services/ | get all registered services in JSON       |
| GET    | /services/<name> | get service with given name in JSON |
| DELETE | /deregister/<name> | deregister service                |
| PUT    | /heartbeat/<name> | update service TTL                |

#### Example:
```
PUT /register
{
  "name": "auth-service",
  "address": "10.0.0.1",
  "port": 8080
}
```

### Health Check TTL ✅

- Registration sets 30 second default TTL for service
- TTL updated through `/heartbeat/<name>` endpoint
- Background process runs every 60 seconds removing expired services
- Services automatically expire if no heartbeat received within TTL
