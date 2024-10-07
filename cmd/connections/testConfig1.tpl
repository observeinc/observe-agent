receivers:
  filelog/agent-config: 
    include: [{{ filepaths }}]
    start_at: {{ start_at }}
    poll_interval: 5m
    multiline:
      line_end_pattern: ENDOFLINEPATTERN

service:
  pipelines:
    logs/agent-config:
       receivers: [filelog/agent-config]
       processors: [memory_limiter, transform/truncate, resourcedetection, resourcedetection/cloud, batch]
       exporters: [otlphttp/observe]
