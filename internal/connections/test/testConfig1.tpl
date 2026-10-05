receivers:
  file_log/agent-config:
    include: []
    start_at: beginning
    poll_interval: 5m
    multiline:
      line_end_pattern: ENDOFLINEPATTERN

service:
  pipelines:
    logs/agent-config:
       receivers: [file_log/agent-config]
       processors: [memory_limiter, transform/truncate, resourcedetection, resourcedetection/cloud, batch]
       exporters: [otlp_http/observe]
