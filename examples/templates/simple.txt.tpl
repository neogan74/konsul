Hello from {{ kv "app/name" }}!

Registered Services:
{{- range services }}
- {{ .Name }} at {{ .Address }}:{{ .Port }}
{{- end }}

Current User: {{ env "USER" }}
Hostname: {{ file "/etc/hostname" }}
