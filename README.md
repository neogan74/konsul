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

#### Example:
```
PUT /registration
{
  "name": "auth-service",
  "address": "10.0.0.1",
  "port": 8080
}
```

### Health Check TTL

- Registration set 30 sec defatult TTL for service.
- Evety 60 seconds TTL updated thoug /heartbeat
- Backgroud process is removing expired services.
