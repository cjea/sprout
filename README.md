# Sprout MVP

This repo currently exposes a CLI-first MVP for Supreme Court opinion ingestion and reading.

What works today:

- boot a SQLite database and apply migrations
- ingest the target Supreme Court opinion PDF
- parse sections and chunk them into bite-sized passages
- inspect stored opinions, passages, and progress
- resume the current passage for a user
- ask a span-anchored question and get a heuristic `guessAnswer`

The current target opinion is:

- `https://www.supremecourt.gov/opinions/25pdf/24-777_9ol1.pdf`

The checked-in local fixture for that opinion is:

- [`fixtures/scotus/24-777_9ol1.pdf`](./fixtures/scotus/24-777_9ol1.pdf)

## Prerequisites

- Go `1.25`
- a working C toolchain for `github.com/mattn/go-sqlite3`

## Test

```sh
mkdir -p .gocache
GOCACHE=$(pwd)/.gocache go test ./...
```

## CLI

Show all commands:

```sh
GOCACHE=$(pwd)/.gocache go run ./cmd/sprout help
```

Initialize the database:

```sh
GOCACHE=$(pwd)/.gocache go run ./cmd/sprout init-db --db var/sprout.db
```

Ingest the checked-in Supreme Court fixture:

```sh
GOCACHE=$(pwd)/.gocache go run ./cmd/sprout ingest-file \
  --db var/sprout.db \
  --user demo \
  --file fixtures/scotus/24-777_9ol1.pdf \
  --url https://www.supremecourt.gov/opinions/25pdf/24-777_9ol1.pdf
```

Ingest directly from the remote URL instead:

```sh
GOCACHE=$(pwd)/.gocache go run ./cmd/sprout ingest-url \
  --db var/sprout.db \
  --user demo \
  --url https://www.supremecourt.gov/opinions/25pdf/24-777_9ol1.pdf
```

Show the stored opinion and sections:

```sh
GOCACHE=$(pwd)/.gocache go run ./cmd/sprout show-opinion \
  --db var/sprout.db \
  --opinion-id 24-777_9ol1
```

List the generated passages:

```sh
GOCACHE=$(pwd)/.gocache go run ./cmd/sprout list-passages \
  --db var/sprout.db \
  --opinion-id 24-777_9ol1
```

Resume reading the current passage:

```sh
GOCACHE=$(pwd)/.gocache go run ./cmd/sprout read \
  --db var/sprout.db \
  --user demo \
  --opinion-id 24-777_9ol1
```

Inspect stored progress:

```sh
GOCACHE=$(pwd)/.gocache go run ./cmd/sprout show-progress \
  --db var/sprout.db \
  --user demo \
  --opinion-id 24-777_9ol1
```

Mark one passage complete:

```sh
GOCACHE=$(pwd)/.gocache go run ./cmd/sprout complete-passage \
  --db var/sprout.db \
  --user demo \
  --passage-id <passage-id>
```

Ask a question against a selected span:

```sh
GOCACHE=$(pwd)/.gocache go run ./cmd/sprout ask \
  --db var/sprout.db \
  --user demo \
  --passage-id <passage-id> \
  --start 0 \
  --end 32 \
  --question "What does this sentence mean?"
```

`ask` saves the question first, then builds context from the stored opinion, passage, citations, and open questions, and finally runs the heuristic `guessAnswer`.

## Scripts

Thin wrappers are available under [`scripts/`](.scripts):

- [`scripts/init_db.sh`](./scripts/init_db.sh)
- [`scripts/ingest_fixture.sh`](./scripts/ingest_fixture.sh)
- [`scripts/show_opinion.sh`](./scripts/show_opinion.sh)
- [`scripts/list_passages.sh`](./scripts/list_passages.sh)
- [`scripts/read_current.sh`](./scripts/read_current.sh)
- [`scripts/ask.sh`](./scripts/ask.sh)

Example:

```sh
scripts/init_db.sh
scripts/ingest_fixture.sh
scripts/show_opinion.sh
scripts/list_passages.sh
scripts/read_current.sh
```

## Current Limits

- There is no browser UI yet. "Read" means CLI inspection of the current passage.
- `ingest-url` depends on network access. `ingest-file` is the reproducible local path.
- The parsing and `guess...` functions are fixture-driven heuristics, not production-grade legal analysis.
