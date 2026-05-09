#!/usr/bin/env bash
# Capture the live Telegram Bot API HTML to testdata/html/snapshot_<date>.html
# and update the latest.html symlink. Used by `make snapshot`.
set -euo pipefail

DATE=$(date +%Y-%m-%d)
DEST="testdata/html/snapshot_${DATE}.html"

curl -fsSL --user-agent "go-telegram codegen scraper" \
  https://core.telegram.org/bots/api > "$DEST"

ln -sf "snapshot_${DATE}.html" "testdata/html/latest.html"
echo "captured: $DEST"
