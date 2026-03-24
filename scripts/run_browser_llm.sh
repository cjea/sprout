#!/usr/bin/env bash
set -euo pipefail

ADDR="${1:-:8080}"
MODEL="${SPROUT_LLM_MODEL:-gpt-4.1-mini}"
BASE_URL="${OPENAI_BASE_URL:-https://api.openai.com}"
GOCACHE_DIR="${GOCACHE:-$(pwd)/.gocache}"

if [[ -z "${OPENAI_API_KEY:-}" ]]; then
  if [[ -t 0 ]]; then
    printf "OpenAI API key required for browser LLM mode.\n" >&2
    printf "Enter OPENAI_API_KEY: " >&2
    read -r OPENAI_API_KEY
    if [[ -z "${OPENAI_API_KEY:-}" ]]; then
      printf "No API key provided. Aborting.\n" >&2
      exit 1
    fi
    export OPENAI_API_KEY
  else
    printf "OPENAI_API_KEY is required for browser LLM mode.\n" >&2
    printf "Set OPENAI_API_KEY in your environment or rerun interactively to be prompted.\n" >&2
    exit 1
  fi
fi

export OPENAI_BASE_URL="$BASE_URL"

printf "Starting sprout-web in LLM mode on %s with model %s\n" "$ADDR" "$MODEL" >&2
printf "Provider base URL: %s\n" "$OPENAI_BASE_URL" >&2

if [[ "${SPROUT_DRY_RUN:-}" == "1" ]]; then
  printf "GOCACHE=%s go run ./cmd/sprout-web --addr %s --model %s\n" "$GOCACHE_DIR" "$ADDR" "$MODEL"
  exit 0
fi

GOCACHE="$GOCACHE_DIR" go run ./cmd/sprout-web --addr "$ADDR" --model "$MODEL"
