global
    log /dev/log local0
    log /dev/log local1 notice
    chroot /var/lib/haproxy
    stats socket /run/haproxy/admin.sock mode 660 level admin
    stats timeout 30s
    user haproxy
    group haproxy
    daemon

defaults
    log     global
    mode    http
    option  httplog
    option  dontlognull
    timeout connect 5000
    timeout client  50000
    timeout server  50000

# Web Service Backend
backend web_backend
    balance roundrobin
    {{- range service "web" }}
    server {{ .Name }}-{{ .Address }} {{ .Address }}:{{ .Port }} check
    {{- end }}

# API Service Backend
backend api_backend
    balance roundrobin
    {{- range service "api" }}
    server {{ .Name }}-{{ .Address }} {{ .Address }}:{{ .Port }} check inter 2000 rise 2 fall 3
    {{- end }}

# Frontend
frontend http_front
    bind *:80
    stats uri /haproxy?stats
    default_backend web_backend

    # Route /api to API backend
    acl url_api path_beg /api
    use_backend api_backend if url_api
