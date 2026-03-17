# Sprout MVP

This repo currently exposes a CLI-first MVP for Supreme Court opinion ingestion and reading.

What works today:

- boot a SQLite database and apply migrations
- ingest the target Supreme Court opinion PDF
- parse sections and chunk them into bite-sized passages
- inspect stored opinions, passages, and progress
- resume the current passage for a user
- ask a span-anchored question and get a heuristic `guessAnswer`
- open a browser app shell that matches the current UX, IA, flow, responsive, and routing specs

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

## Browser Shell

Run the browser shell:

```sh
GOCACHE=$(pwd)/.gocache go run ./cmd/sprout-web --addr :8080
```

Or use the wrapper:

```sh
scripts/run_browser.sh
```

Then open:

```txt
http://localhost:8080
```

This browser target now boots against the real checked-in Supreme Court fixture and renders a real passage-by-passage reading shell through `/api/reader`. It shows real opinion identity, passage text, progress snapshot, and citation availability, and the `Continue` action advances through real generated passages.

Current browser capabilities:

- load the real `24-777_9ol1` opinion fixture into SQLite if needed
- render the current passage and passage metadata
- advance through real generated passages
- open citation detail without leaving the reading workspace
- select passage text, submit a question, and review a heuristic answer
- keep passage and panel state in the URL enough to support refresh and basic back/forward navigation

Useful browser URL shapes:

```txt
http://localhost:8080/?passage=<passage-id>
http://localhost:8080/?passage=<passage-id>&panel=citation
http://localhost:8080/?passage=<passage-id>&panel=question&start=0&end=32
```

## Scripts

Thin wrappers are available under [`scripts/`](.scripts):

- [`scripts/init_db.sh`](./scripts/init_db.sh)
- [`scripts/ingest_fixture.sh`](./scripts/ingest_fixture.sh)
- [`scripts/show_opinion.sh`](./scripts/show_opinion.sh)
- [`scripts/list_passages.sh`](./scripts/list_passages.sh)
- [`scripts/read_current.sh`](./scripts/read_current.sh)
- [`scripts/ask.sh`](./scripts/ask.sh)
- [`scripts/run_browser.sh`](./scripts/run_browser.sh)

Example:

```sh
scripts/init_db.sh
scripts/ingest_fixture.sh
scripts/show_opinion.sh
scripts/list_passages.sh
scripts/read_current.sh
```

## Current Limits

- The browser target is now usable for the real fixture, but it is still an MVP shell rather than a finished product.
- Citation rendering and question flow work, but they are still minimal and heuristic-driven.
- `ingest-url` depends on network access. `ingest-file` is the reproducible local path.
- The parsing and `guess...` functions are fixture-driven heuristics, not production-grade legal analysis.
