#!/usr/bin/env bash
# Exports BOT_API_VERSION for the .goreleaser.yaml release header.
set -euo pipefail

[ -n "${GITHUB_ENV:-}" ] || exit 0

ver=$(python3 -c 'import json; print(json.load(open("internal/spec/api.json"))["version"])')
if [ -z "$ver" ] || [ "$ver" = "null" ]; then
  exit 0
fi
echo "BOT_API_VERSION=${ver}" >> "$GITHUB_ENV"
