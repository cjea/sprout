#!/usr/bin/env bash
set -euo pipefail

DB_PATH="${1:-var/sprout.db}"

GOCACHE="${GOCACHE:-$(pwd)/.gocache}" go run ./cmd/sprout init-db --db "$DB_PATH"
