#!/usr/bin/env bash
set -euo pipefail

image="${1:-ghcr.io/devaryakjha/loadwright:latest}"
docker_config="$(mktemp -d)"

cleanup() {
  rm -rf "$docker_config"
}
trap cleanup EXIT

echo "verifying anonymous pull for $image"
DOCKER_CONFIG="$docker_config" docker pull "$image"
