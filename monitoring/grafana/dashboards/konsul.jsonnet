local grafana = import '../vendor/grafonnet/grafonnet/grafana.libsonnet';
local dashboard = grafana.dashboard;
local row = grafana.row;
local prometheus = grafana.prometheus;
local template = grafana.template;
local graphPanel = grafana.graphPanel;
local singlestat = grafana.singlestat;
local gaugePanel = grafana.gaugePanel;
local tablePanel = grafana.tablePanel;

dashboard.new(
  'Konsul Dashboard',
  description='Comprehensive monitoring dashboard for Konsul service discovery and KV store',
  uid='konsul-main',
  time_from='now-1h',
  editable=true,
  tags=['konsul', 'service-discovery', 'kv-store'],
  refresh='30s',
  schemaVersion=16,
)
.addTemplate(
  template.datasource(
    'datasource',
    'prometheus',
    'Prometheus',
  )
)
.addTemplate(
  template.new(
    'instance',
    '$datasource',
    'label_values(konsul_build_info, instance)',
    label='Instance',
    refresh='time',
    multi=false,
    includeAll=false,
  )
)

// Overview Row
.addRow(
  row.new(title='Overview', collapse=false)
)
.addPanel(
  singlestat.new(
    'Uptime',
    datasource='$datasource',
    format='s',
    sparklineShow=true,
  )
  .addTarget(
    prometheus.target(
      'time() - process_start_time_seconds{instance="$instance"}',
      legendFormat='Uptime',
    )
  ), gridPos={x: 0, y: 0, w: 6, h: 4}
)
.addPanel(
  singlestat.new(
    'Total Services',
    datasource='$datasource',
    format='none',
    colorBackground=true,
    thresholds='10,50',
    colors=['#299c46', '#e0b400', '#d44a3a'],
  )
  .addTarget(
    prometheus.target(
      'konsul_registered_services_total{instance="$instance"}',
      legendFormat='Services',
    )
  ), gridPos={x: 6, y: 0, w: 6, h: 4}
)
.addPanel(
  singlestat.new(
    'KV Store Keys',
    datasource='$datasource',
    format='none',
    colorBackground=true,
    thresholds='100,1000',
    colors=['#299c46', '#e0b400', '#d44a3a'],
  )
  .addTarget(
    prometheus.target(
      'konsul_kv_store_size{instance="$instance"}',
      legendFormat='Keys',
    )
  ), gridPos={x: 12, y: 0, w: 6, h: 4}
)
.addPanel(
  singlestat.new(
    'Request Rate',
    datasource='$datasource',
    format='reqps',
    sparklineShow=true,
  )
  .addTarget(
    prometheus.target(
      'rate(konsul_http_requests_total{instance="$instance"}[5m])',
      legendFormat='req/s',
    )
  ), gridPos={x: 18, y: 0, w: 6, h: 4}
)

// HTTP Metrics Row
.addRow(
  row.new(title='HTTP Metrics', collapse=false)
)
.addPanel(
  graphPanel.new(
    'HTTP Request Rate by Status',
    datasource='$datasource',
    format='reqps',
    legend_show=true,
    legend_alignAsTable=true,
    legend_values=true,
    legend_current=true,
    legend_avg=true,
  )
  .addTarget(
    prometheus.target(
      'rate(konsul_http_requests_total{instance="$instance"}[5m])',
      legendFormat='{{method}} {{path}} [{{status}}]',
    )
  ), gridPos={x: 0, y: 4, w: 12, h: 8}
)
.addPanel(
  graphPanel.new(
    'HTTP Request Duration (p50, p95, p99)',
    datasource='$datasource',
    format='s',
    legend_show=true,
    legend_alignAsTable=true,
    legend_values=true,
    legend_current=true,
  )
  .addTarget(
    prometheus.target(
      'histogram_quantile(0.50, rate(konsul_http_request_duration_seconds_bucket{instance="$instance"}[5m]))',
      legendFormat='p50',
    )
  )
  .addTarget(
    prometheus.target(
      'histogram_quantile(0.95, rate(konsul_http_request_duration_seconds_bucket{instance="$instance"}[5m]))',
      legendFormat='p95',
    )
  )
  .addTarget(
    prometheus.target(
      'histogram_quantile(0.99, rate(konsul_http_request_duration_seconds_bucket{instance="$instance"}[5m]))',
      legendFormat='p99',
    )
  ), gridPos={x: 12, y: 4, w: 12, h: 8}
)
.addPanel(
  graphPanel.new(
    'In-Flight Requests',
    datasource='$datasource',
    format='short',
    legend_show=true,
  )
  .addTarget(
    prometheus.target(
      'konsul_http_requests_in_flight{instance="$instance"}',
      legendFormat='In-flight',
    )
  ), gridPos={x: 0, y: 12, w: 12, h: 6}
)

// KV Store Metrics Row
.addRow(
  row.new(title='KV Store Metrics', collapse=false)
)
.addPanel(
  graphPanel.new(
    'KV Operations Rate',
    datasource='$datasource',
    format='ops',
    legend_show=true,
    legend_alignAsTable=true,
    legend_values=true,
    legend_current=true,
  )
  .addTarget(
    prometheus.target(
      'rate(konsul_kv_operations_total{instance="$instance"}[5m])',
      legendFormat='{{operation}} [{{status}}]',
    )
  ), gridPos={x: 0, y: 18, w: 12, h: 8}
)
.addPanel(
  graphPanel.new(
    'KV Store Size Over Time',
    datasource='$datasource',
    format='short',
    legend_show=true,
  )
  .addTarget(
    prometheus.target(
      'konsul_kv_store_size{instance="$instance"}',
      legendFormat='Keys',
    )
  ), gridPos={x: 12, y: 18, w: 12, h: 8}
)

// Service Discovery Metrics Row
.addRow(
  row.new(title='Service Discovery Metrics', collapse=false)
)
.addPanel(
  graphPanel.new(
    'Service Operations Rate',
    datasource='$datasource',
    format='ops',
    legend_show=true,
    legend_alignAsTable=true,
    legend_values=true,
    legend_current=true,
  )
  .addTarget(
    prometheus.target(
      'rate(konsul_service_operations_total{instance="$instance"}[5m])',
      legendFormat='{{operation}} [{{status}}]',
    )
  ), gridPos={x: 0, y: 26, w: 12, h: 8}
)
.addPanel(
  graphPanel.new(
    'Registered Services',
    datasource='$datasource',
    format='short',
    legend_show=true,
  )
  .addTarget(
    prometheus.target(
      'konsul_registered_services_total{instance="$instance"}',
      legendFormat='Total Services',
    )
  ), gridPos={x: 12, y: 26, w: 6, h: 8}
)
.addPanel(
  graphPanel.new(
    'Service Heartbeats',
    datasource='$datasource',
    format='ops',
    legend_show=true,
  )
  .addTarget(
    prometheus.target(
      'rate(konsul_service_heartbeats_total{instance="$instance"}[5m])',
      legendFormat='{{service}} [{{status}}]',
    )
  ), gridPos={x: 18, y: 26, w: 6, h: 8}
)
.addPanel(
  graphPanel.new(
    'Expired Services Cleanup',
    datasource='$datasource',
    format='short',
    legend_show=true,
  )
  .addTarget(
    prometheus.target(
      'increase(konsul_expired_services_total{instance="$instance"}[5m])',
      legendFormat='Expired',
    )
  ), gridPos={x: 0, y: 34, w: 12, h: 6}
)

// Rate Limiting Metrics Row
.addRow(
  row.new(title='Rate Limiting', collapse=false)
)
.addPanel(
  graphPanel.new(
    'Rate Limit Checks',
    datasource='$datasource',
    format='reqps',
    legend_show=true,
    legend_alignAsTable=true,
  )
  .addTarget(
    prometheus.target(
      'rate(konsul_rate_limit_requests_total{instance="$instance"}[5m])',
      legendFormat='{{limiter_type}} [{{status}}]',
    )
  ), gridPos={x: 0, y: 40, w: 12, h: 8}
)
.addPanel(
  graphPanel.new(
    'Rate Limit Violations',
    datasource='$datasource',
    format='short',
    legend_show=true,
  )
  .addTarget(
    prometheus.target(
      'increase(konsul_rate_limit_exceeded_total{instance="$instance"}[5m])',
      legendFormat='{{limiter_type}}',
    )
  ), gridPos={x: 12, y: 40, w: 12, h: 8}
)
.addPanel(
  graphPanel.new(
    'Active Rate Limited Clients',
    datasource='$datasource',
    format='short',
    legend_show=true,
  )
  .addTarget(
    prometheus.target(
      'konsul_rate_limit_active_clients{instance="$instance"}',
      legendFormat='{{limiter_type}}',
    )
  ), gridPos={x: 0, y: 48, w: 12, h: 6}
)

// System Metrics Row
.addRow(
  row.new(title='System Metrics', collapse=false)
)
.addPanel(
  graphPanel.new(
    'Memory Usage',
    datasource='$datasource',
    format='bytes',
    legend_show=true,
    legend_alignAsTable=true,
    legend_values=true,
    legend_current=true,
  )
  .addTarget(
    prometheus.target(
      'go_memstats_alloc_bytes{instance="$instance"}',
      legendFormat='Allocated',
    )
  )
  .addTarget(
    prometheus.target(
      'go_memstats_sys_bytes{instance="$instance"}',
      legendFormat='System',
    )
  ), gridPos={x: 0, y: 54, w: 12, h: 8}
)
.addPanel(
  graphPanel.new(
    'Goroutines',
    datasource='$datasource',
    format='short',
    legend_show=true,
  )
  .addTarget(
    prometheus.target(
      'go_goroutines{instance="$instance"}',
      legendFormat='Goroutines',
    )
  ), gridPos={x: 12, y: 54, w: 12, h: 8}
)
.addPanel(
  graphPanel.new(
    'GC Duration',
    datasource='$datasource',
    format='s',
    legend_show=true,
  )
  .addTarget(
    prometheus.target(
      'rate(go_gc_duration_seconds_sum{instance="$instance"}[5m]) / rate(go_gc_duration_seconds_count{instance="$instance"}[5m])',
      legendFormat='Average GC Duration',
    )
  ), gridPos={x: 0, y: 62, w: 12, h: 6}
)
