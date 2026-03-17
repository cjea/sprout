#!/usr/bin/env bash
set -euo pipefail

ADDR="${1:-:8080}"

GOCACHE="${GOCACHE:-$(pwd)/.gocache}" go run ./cmd/sprout-web --addr "$ADDR"
