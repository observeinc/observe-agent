---
name: release-agent
description: >-
  Release a new version of the Observe Agent. Use when the user asks to release,
  tag, publish, or cut a new version of the observe-agent, or mentions the
  release process.
---

# Release Observe Agent

## Step 1: Review changelog and determine version bump

Run the [Generate Release Changelog](https://github.com/observeinc/observe-agent/actions/workflows/generate-changelog.yaml) action if not already done. Then determine the latest released version and review changes:

```bash
git tag --sort=-v:refname | head -10
git log <last-tag>..HEAD --oneline
git log <last-tag>..HEAD --oneline --grep="!"  # breaking changes
```

Present the changes to the user and recommend a version bump:

- **Major**: Breaking changes — removed metrics/logs/traces, removed components,
  removed CLI commands/flags, removed OS/arch support, major OTel Collector upgrade
- **Minor**: Non-breaking features — new metrics/logs/traces, new components,
  new CLI commands/flags, OTel minor/patch upgrades, sampling/filtering changes
- **Patch**: Security fixes, smaller bugfixes

Confirm the new version number with the user before proceeding.

## Step 2: Check for CVEs

First, do a quick local check by building the Docker image and running Docker Scout:

```bash
goreleaser build --snapshot --id=linux_build --skip=validate --single-target
cp dist/linux_amd64/linux_build_linux_amd64_v1/observe-agent .
docker build --build-arg TARGETPLATFORM=. -f packaging/docker/Dockerfile -t observeinc/observe-agent:test .
docker scout cves --only-fixed --only-severity medium,critical,high observeinc/observe-agent:test
```

If the local scout check is clean, proceed to the next step. If CVEs are found (or to get full recommendations), also run the [Docker Image Vulnerability Check](https://github.com/observeinc/observe-agent/actions/workflows/vuln-check-full.yaml) against `main`. If critical or high vulnerabilities are found:

1. Identify the import chain for the vulnerable package:

```bash
go mod why -m $PACKAGE_NAME
```

2. If the package comes from `otel-collector-contrib`, check if the latest
   upstream version resolves the CVE. If so, bump versions in
   `builder-config.yaml` — note that all OTel versions (base collector + components)
   usually need to be bumped together.

3. If the package is imported directly, run:

```bash
go mod tidy
```

Repeat until no critical/high CVEs remain.

## Step 3: Test with a nightly release

Trigger the [Release Nightly Version](https://github.com/observeinc/observe-agent/actions/workflows/release-nightly.yaml) action. This verifies:
- The goreleaser build process succeeds
- The built Docker image works correctly

The nightly image is automatically deployed to the eng cluster. You can also
test locally using the [agent dev-tools repo](https://github.com/observeinc/observe-agent-dev-tools/blob/main/snippets.md#agent-helm):

```bash
./start_k8s_demo.sh -f helm/values/nightly-image.yaml
```

Verify via the **Observe MCP** that all expected data from the **eng cluster** is flowing:
- **k8s explorer**: OTel demo pods should appear as running pods
- **APM / traces**: OTel demo services should appear with traces and logs
  (if APM transform is running slowly, check traces directly)

Do not proceed to tagging until nightly testing passes.

## Step 4: Create and push the tag

Ensure you're on `main` with a clean working tree, then:

```bash
git tag v<new-version>
git push origin tag v<new-version>
```

## Step 5: Monitor the Release Version action

The tag push triggers the [Release Version](https://github.com/observeinc/observe-agent/actions/workflows/release.yaml) workflow. Monitor it:

```bash
gh run list --repo observeinc/observe-agent --workflow release.yaml --limit 1
```

Past successful releases take ~30-37 minutes. Watch until completion:

```bash
gh run watch <run-id> --repo observeinc/observe-agent
```

On failure, fetch logs with:

```bash
gh run view <run-id> --repo observeinc/observe-agent --log-failed | tail -100
```

## Step 6: Draft release announcement

Once the action succeeds, draft a message for the **#observe-agent** slack channel
with this structure:

```
Observe Agent v<version> Released :rocket:

We've just released v<version>! Here's what's new:

**Features**
- <list of feat: commits with PR numbers>

**Fixes**
- <list of fix: commits with PR numbers>

**Infrastructure** (if applicable)
- <list of chore: commits summarized>
```

Include a link to the GitHub release page.
