#!/usr/bin/env bash
set -euo pipefail

DB_PATH="${1:-var/sprout.db}"
USER_ID="${USER_ID:-demo}"
OPINION_ID="${2:-24-777_9ol1}"

GOCACHE="${GOCACHE:-$(pwd)/.gocache}" go run ./cmd/sprout read \
  --db "$DB_PATH" \
  --user "$USER_ID" \
  --opinion-id "$OPINION_ID"
