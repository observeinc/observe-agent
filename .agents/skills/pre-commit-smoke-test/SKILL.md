---
name: pre-commit-smoke-test
description: >-
  Run a local end-to-end smoke test of the observe-agent to verify it collects
  and exports telemetry correctly. Use when the user wants to verify their branch
  changes work, run a smoke test, test the agent locally, or validate the
  observe-agent pipeline before committing or merging.
---

# Pre-Commit Smoke Test

End-to-end verification that the observe-agent binary on the current branch can
start, collect log telemetry, and deliver it to the Observe backend.

## Prerequisites

- The observe-agent config at `/etc/observe-agent/observe-agent.yaml` must
  exist with valid `token` and `observe_url` fields.
- The config must have `host_monitoring.logs.enabled: true` with an include glob
  that covers `/var/log/**/*.log`.
- If no config file exists for you to refer to, inform the user, and halt execution immediately.
- The Observe MCP server (`user-observe`) must be connected for backend
  verification.

## Step 1: Build or locate the binary

Check if a freshly built `observe-agent` binary exists in the project root:

```bash
ls -la observe-agent
```

If the binary is missing or stale (older than the latest source change), build
it:

```bash
make build
```

## Step 2: Stop any running agent

Ensure no existing agent is holding the health-check port or file-storage locks:

```bash
sudo systemctl stop observe-agent 2>/dev/null || true
```

## Step 3: Start the agent

Launch the agent in the background with the local config. Set `OTEL_LOG_LEVEL`
since the config references it via `${env:OTEL_LOG_LEVEL}`:

```bash
sudo OTEL_LOG_LEVEL=info ./observe-agent \
  --observe-config /etc/observe-agent/observe-agent.yaml start
```

Run this as a background shell command (`block_until_ms: 0`) with
`required_permissions: ["all"]` so the agent can read `/var/log/` and bind
privileged ports.

## Step 4: Confirm startup

Wait ~10 seconds, then verify:

1. **Search the agent output** for `Everything is ready. Begin running and
processing data.` to confirm successful startup.
2. **Hit the health endpoint**:

```bash
curl -s http://localhost:13133/status
```

Expected response: `{"status":"Server available", ...}`. If startup failed,
check the agent output for errors and report them to the user.

## Step 5: Generate smoke telemetry

Write distinctly identifiable log lines into a path the filelog receiver will
pick up. Use a unique marker with a Unix timestamp so the data can be queried
unambiguously:

```bash
SMOKE_MARKER="OBSERVE_SMOKE_TEST_$(date +%s)"
sudo mkdir -p /var/log/smoke-test
for i in $(seq 1 10); do
  echo "{\"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%S.%3NZ)\", \
\"level\": \"INFO\", \
\"message\": \"${SMOKE_MARKER} line ${i} - smoke test\", \
\"service\": \"smoke-test\", \
\"line_number\": ${i}}" \
  | sudo tee -a /var/log/smoke-test/smoke-test.log > /dev/null
  sleep 0.5
done
```

After writing, confirm the agent detected the file by searching its output for:
`Started watching file.*smoke-test.log`

## Step 6: Wait for export

Wait **30-45 seconds** for the agent to poll the file, batch the logs, and
export them to the Observe backend. During this time, verify the agent is still
healthy:

```bash
curl -s http://localhost:13133/status
```

Also check the agent output for any `Exporting failed` errors. A few
`Failed to open file` errors for stale checkpoint paths (e.g. old k8s pod logs)
are harmless and expected on non-k8s hosts.

## Step 7: Query the Observe backend

Use the Observe MCP `generate-query-card` tool to verify the smoke test logs
reached the backend:

```
Tool: generate-query-card
Prompt: "Show me logs from the last 10 minutes that contain
  'OBSERVE_SMOKE_TEST' in the log body from host <hostname>"
```

Replace `<hostname>` with the host's FQDN (visible in the agent's
`resourcedetection` output, e.g. `ip-0-1-2-3.us-east-2.compute.internal`).

### Interpret results

| Result                                                            | Verdict                                         |
| ----------------------------------------------------------------- | ----------------------------------------------- |
| All 10 log lines present with correct marker, host, and file path | **PASS**                                        |
| Partial results (< 10 lines)                                      | **WARN** — may need more wait time, retry query |
| Zero results after 60+ seconds                                    | **FAIL** — check agent output for export errors |

When logs are confirmed, report to the user:

- Number of log lines found (expected: 10)
- Source file path (`/var/log/smoke-test/smoke-test.log`)
- Resource attributes present (cloud metadata, host info, deployment env)
- Ingestion latency (time between log write and backend timestamp)
- Link to the Observe worksheet (included in the MCP response)

## Step 8: Clean up

After verification, stop the agent:

```bash
sudo kill $(pgrep -f 'observe-agent.*start') 2>/dev/null || true
```

Optionally remove the smoke test log:

```bash
sudo rm -rf /var/log/smoke-test
```

## Failure Troubleshooting

| Symptom                             | Likely Cause                           | Action                                                                  |
| ----------------------------------- | -------------------------------------- | ----------------------------------------------------------------------- |
| Agent fails to start                | Port 13133 or 4317/4318 already in use | `sudo systemctl stop observe-agent` or kill conflicting process         |
| `Everything is ready` never appears | Config validation error                | Check agent output for `error` lines near startup                       |
| File not watched                    | Glob mismatch                          | Verify config `host_monitoring.logs.include` covers `/var/log/**/*.log` |
| Logs not in backend                 | Export failure or auth issue           | Check for `401` or `Exporting failed` in agent output                   |
| Partial logs in backend             | Batch not yet flushed                  | Wait longer and re-query                                                |
