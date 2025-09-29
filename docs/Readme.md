

# DNS

✅ DNS Query Examples:

  # Service discovery via SRV records
  ```bash
  dig @localhost -p 8600 _web._tcp.service.consul SRV

  # Direct IP lookup via A records
  dig @localhost -p 8600 web.node.consul A
  dig @localhost -p 8600 web.service.consul A
```
  ✅ Environment Configuration:
```bash
  KONSUL_DNS_ENABLED=true    # Enable/disable DNS server
  KONSUL_DNS_HOST=""         # DNS bind host (empty = all interfaces)
  KONSUL_DNS_PORT=8600       # DNS server port
  KONSUL_DNS_DOMAIN="consul" # DNS domain suffix
```

>  The DNS interface is now a complete, production-ready feature with comprehensive testing, proper error handling, and full integration with the existing Konsul service architecture. All tests pass and the implementation follows DNS RFC standards for both SRV and A record responses.