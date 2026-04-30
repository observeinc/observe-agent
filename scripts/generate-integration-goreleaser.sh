#!/usr/bin/env bash
set -euo pipefail

input="${1:-.goreleaser.yaml}"
output="${2:-.goreleaser.integration.generated.yaml}"

yq eval -P '
{
  "version": .version,
  "project_name": .project_name,
  "env": .env,
  "builds": [
    (.builds[] | select(.id == "linux_build") | .goarch = ["amd64"]),
    (.builds[] | select(.id == "docker_build") | .goarch = ["amd64"]),
    (.builds[] | select(.id == "windows_build") | .goarch = ["amd64"])
  ],
  "archives": [
    (.archives[] | select(.id == "windows"))
  ],
  "nfpms": [
    (.nfpms[] | select(.id == "linux") | .formats = ["deb", "rpm"])
  ],
  "dockers_v2": [
    (
      .dockers_v2[]
      | .images = ["observe-agent"]
      | .tags = ["{{ .Version }}"]
      | .platforms = ["linux/amd64"]
      | del(.sbom)
    )
  ],
  "git": .git
}
' "$input" > "$output"
