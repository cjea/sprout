#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 5 ]]; then
  echo "usage: scripts/ask.sh <db-path> <passage-id> <start> <end> <question>" >&2
  exit 1
fi

DB_PATH="$1"
PASSAGE_ID="$2"
START_OFFSET="$3"
END_OFFSET="$4"
QUESTION_TEXT="$5"
USER_ID="${USER_ID:-demo}"

GOCACHE="${GOCACHE:-$(pwd)/.gocache}" go run ./cmd/sprout ask \
  --db "$DB_PATH" \
  --user "$USER_ID" \
  --passage-id "$PASSAGE_ID" \
  --start "$START_OFFSET" \
  --end "$END_OFFSET" \
  --question "$QUESTION_TEXT"
