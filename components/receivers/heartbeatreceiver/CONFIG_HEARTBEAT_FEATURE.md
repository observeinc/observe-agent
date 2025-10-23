# Config Heartbeat Feature

## Overview

The config heartbeat feature sends periodic heartbeat events containing the observe-agent configuration and rendered OTEL configuration. Sensitive fields are automatically obfuscated before transmission.

## Features

- **Event Type**: "AgentConfig" heartbeat events
- **Event Body**: Two base64-encoded configuration files:
  - `observeAgentConfig`: observe-agent.yaml with sensitive fields obfuscated
  - `otelConfig`: Fully rendered OTEL configuration
- **Timing**: Configurable interval (default: 24h)
- **Security**: Pattern-based obfuscation of sensitive fields
- **Independence**: Runs on separate timer from lifecycle heartbeat

## Sensitive Field Obfuscation

### How It Works

The obfuscation system uses YAML AST parsing to accurately identify and obfuscate sensitive fields:

1. Parses YAML into Abstract Syntax Tree
2. Traverses tree matching against configured patterns
3. Obfuscates matched values (shows prefix, replaces rest with asterisks)
4. Marshals back to YAML

### Configuration

Sensitive fields are defined in `receiver.go` using dot-separated paths:

```go
var sensitiveFieldPatterns = []SensitiveFieldPattern{
    {Path: "token", PrefixLength: 8},
}
```

### Adding New Sensitive Fields

To obfuscate additional fields:

1. Open `components/receivers/heartbeatreceiver/receiver.go`
2. Find `sensitiveFieldPatterns` variable
3. Add your pattern:

```go
var sensitiveFieldPatterns = []SensitiveFieldPattern{
    {Path: "token", PrefixLength: 8},
    {Path: "database.password", PrefixLength: 4},
    {Path: "api.credentials.secret", PrefixLength: 6},
}
```

**Path Format:**
- Use dot notation for nested fields
- Example: `"database.password"` matches:
  ```yaml
  database:
    password: secretvalue  # This gets obfuscated
  ```

**PrefixLength:**
- Number of characters to show before obfuscating
- Remaining characters replaced with asterisks
- Example: `PrefixLength: 8` on "secrettoken123" → "secretto*******"

## Configuration

### Agent Config YAML

```yaml
self_monitoring:
  enabled: true
  fleet:
    enabled: true
    interval: "10m"          # Lifecycle heartbeat interval
    config_interval: "24h"   # Config heartbeat interval
```

### Configuration Fields

- `interval`: How often to send lifecycle heartbeats (default: 10m)
- `config_interval`: How often to send config heartbeats (default: 24h, minimum: 5s)

## Event Structure

Config heartbeat events have the following structure:

```json
{
  "kind": "AgentConfig",
  "body": {
    "observeAgentConfig": "base64_encoded_yaml...",
    "otelConfig": "base64_encoded_yaml..."
  },
  "observe_transform": {
    "identifiers": {
      "observe.agent.instance.id": "..."
    },
    "control": {
      "isDelete": false
    },
    "process_start_time": 1234567890,
    "valid_from": 1234567890000000000,
    "valid_to": 1234573290000000000,
    "kind": "AgentConfig"
  }
}
```

### Resource Attributes

- `observe.agent.instance.id`: Unique agent instance identifier
- `observe.agent.environment`: Environment (linux/macos/windows)
- `observe.agent.processId`: Process ID

## Error Handling

Failures are logged but do not impact the agent:

- Config errors → Log and retry on next interval
- OTEL config generation fails → Log and retry on next interval
- YAML parsing fails → Return original content (fail-safe)
- Timer errors → Log but continue running

### Security Note

The `observeAgentConfig` field contains obfuscated sensitive values. For example:
- `token: ds_abc12********************************` (shows first 8 chars)

Do not assume this data is safe to share publicly - it may contain partial sensitive information.
