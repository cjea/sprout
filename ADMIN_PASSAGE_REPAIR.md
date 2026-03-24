# Admin Passage Repair

This is the current operator guide for passage repair.

It is intentionally simple.

Today, the passage-repair subsystem exists as Go interfaces plus a CLI-first admin surface.

The current admin workflow is:

1. inspect passages from the existing reader or CLI
2. open a repair case around one passage
3. classify the issue
4. apply a primitive repair operation
5. inspect history or undo if needed

## Ground Rules

- Do not rewrite passage text manually.
- Treat the source document as canonical.
- Repairs should be structural: merge, split, move a sentence, drop, restore, undo.
- System transforms are gated.
- Any LLM-backed repair must use a `guess...` name.

## Common Problems

Look for:

- a citation detached into the next passage
- `Pp. 3-10.` detached from the sentence it belongs to
- `Id., at 484.` detached into its own fragment
- an orphan fragment that should merge with an adjacent passage
- a passage that should split at a sentence boundary
- extraction artifacts like page headers or bad hyphens

## Primitive Repairs

Use these concepts when deciding what should happen:

- `merge_with_next`
- `merge_with_previous`
- `split_at_sentence`
- `move_last_sentence_to_next`
- `move_first_sentence_to_previous`
- `drop_passage`
- `restore_passage`
- `revert_last_operation`

These are the only kinds of admin repairs the current subsystem is meant to support.

## Current Code Surface

The current implementation lives here:

- [passage_repair_policy.go](/Users/babe/code/sprout/mvp/passage_repair_policy.go)
- [passage_repair.go](/Users/babe/code/sprout/mvp/passage_repair.go)
- [passage_repair_store.go](/Users/babe/code/sprout/mvp/passage_repair_store.go)
- [passage_issue.go](/Users/babe/code/sprout/mvp/passage_issue.go)
- [passage_repair_flow.go](/Users/babe/code/sprout/mvp/passage_repair_flow.go)

Design context lives here:

- [passage_repair.md](/Users/babe/code/sprout/passage_repair.md)
- [passage_repair_flows.md](/Users/babe/code/sprout/passage_repair_flows.md)

## Admin Commands

Open a repair case around one passage:

```sh
GOCACHE=$(pwd)/.gocache go run ./cmd/sprout repair-case \
  --db var/sprout.db \
  --opinion-id 24-777_9ol1 \
  --passage-id <passage-id>
```

Classify passage issues for one passage:

```sh
GOCACHE=$(pwd)/.gocache go run ./cmd/sprout repair-issues \
  --db var/sprout.db \
  --opinion-id 24-777_9ol1 \
  --passage-id <passage-id>
```

Apply one structural repair:

```sh
GOCACHE=$(pwd)/.gocache go run ./cmd/sprout repair-apply \
  --db var/sprout.db \
  --opinion-id 24-777_9ol1 \
  --passage-id <passage-id> \
  --op merge_with_next
```

Inspect repair history:

```sh
GOCACHE=$(pwd)/.gocache go run ./cmd/sprout repair-history \
  --db var/sprout.db \
  --opinion-id 24-777_9ol1
```

Undo the last repair:

```sh
GOCACHE=$(pwd)/.gocache go run ./cmd/sprout repair-undo \
  --db var/sprout.db \
  --opinion-id 24-777_9ol1
```

## How To Exercise The Subsystem Today

### 1. Run the full repair-focused test slice

```sh
mkdir -p .gocache
GOCACHE=$(pwd)/.gocache go test ./mvp -run 'TestOpenPassageRepairCase|TestClassifyPassageRepairCase|TestApplyAndUndoPassageRepairCaseOperation|TestApplyAdminPassageOperation.*|TestPassageRepairSession.*|TestMemoryPassageRepairStore.*|TestParsePassageIssueKind|TestClassifyPassageIssues.*' -v
```

This verifies:

- source-fidelity rules
- issue classification
- merge, split, move, drop, restore
- undo
- snapshot persistence
- lightweight repair flow state

### 2. Inspect the fixture-backed passage output

Initialize and ingest the real fixture:

```sh
scripts/init_db.sh
scripts/ingest_fixture.sh
```

List passages:

```sh
scripts/list_passages.sh
```

Read the current passage:

```sh
scripts/read_current.sh
```

Use this to identify a passage you want to repair conceptually.

### 3. Open a repair case and classify the problem

Use:

```sh
GOCACHE=$(pwd)/.gocache go run ./cmd/sprout repair-case \
  --db var/sprout.db \
  --opinion-id 24-777_9ol1 \
  --passage-id <passage-id>
```

Then:

```sh
GOCACHE=$(pwd)/.gocache go run ./cmd/sprout repair-issues \
  --db var/sprout.db \
  --opinion-id 24-777_9ol1 \
  --passage-id <passage-id>
```

### 4. Map the problem to a primitive operation

Examples:

- detached citation: `merge_with_previous`
- detached `Pp. 3-10.`: `merge_with_previous`
- passage too long: `split_at_sentence`
- orphan leading sentence: `move_first_sentence_to_previous`
- orphan trailing sentence: `move_last_sentence_to_next`
- pure extraction artifact: drop the artifact passage or rerun a gated system transform

### 5. Apply the repair and verify it

Apply the smallest structural operation that fits the problem:

```sh
GOCACHE=$(pwd)/.gocache go run ./cmd/sprout repair-apply \
  --db var/sprout.db \
  --opinion-id 24-777_9ol1 \
  --passage-id <passage-id> \
  --op merge_with_next
```

Then inspect history or undo:

```sh
GOCACHE=$(pwd)/.gocache go run ./cmd/sprout repair-history \
  --db var/sprout.db \
  --opinion-id 24-777_9ol1

GOCACHE=$(pwd)/.gocache go run ./cmd/sprout repair-undo \
  --db var/sprout.db \
  --opinion-id 24-777_9ol1
```

### 6. Verify the expected behavior in tests

The clearest current references are:

- [passage_repair_test.go](/Users/babe/code/sprout/mvp/passage_repair_test.go)
- [passage_issue_test.go](/Users/babe/code/sprout/mvp/passage_issue_test.go)
- [passage_repair_flow_test.go](/Users/babe/code/sprout/mvp/passage_repair_flow_test.go)
- [passage_repair_store_test.go](/Users/babe/code/sprout/mvp/passage_repair_store_test.go)

Those tests are the current executable manual for repair behavior.

## What You Cannot Do Yet

- persist admin repair sessions into SQLite
- drive repairs from the browser UI
- manually edit text

Those are future layers on top of the repair subsystem that now exists.

## Safe Operating Rule

If you are unsure what to do, prefer the smallest reversible structural repair and verify it.

If the only plausible fix requires rewriting text by hand, stop. That is outside the allowed admin model.
