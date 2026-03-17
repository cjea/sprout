#!/usr/bin/env bash
set -euo pipefail

DB_PATH="${1:-var/sprout.db}"
OPINION_ID="${2:-24-777_9ol1}"

GOCACHE="${GOCACHE:-$(pwd)/.gocache}" go run ./cmd/sprout show-opinion --db "$DB_PATH" --opinion-id "$OPINION_ID"
