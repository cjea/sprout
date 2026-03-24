# Passage Repair Subsystem

This document defines the admin-facing passage repair subsystem for the Supreme Court opinion reader.

The goal is not a heavyweight review queue. The goal is a light, direct way to repair passage structure when the ingestion pipeline makes mistakes.

## Product Posture

The system should feel easy to manipulate.

- direct
- reversible
- low ceremony
- source-faithful

Admins should be able to look at a bad passage, understand what is wrong, and fix it with a small structural action.

The system should not feel like:

- proposing changes for later acceptance
- editing raw text in a freeform field
- maintaining a separate editorial rewrite layer

## Hard Constraints

### Source Fidelity

Admins must never be allowed to freely edit passage text.

The source document remains canonical. Passage repair operates on source-derived structure, not arbitrary rewritten prose.

Allowed:

- merge passage with next passage
- merge passage with previous passage
- split passage at a valid source-derived sentence boundary
- drop a passage artifact
- restore a dropped passage
- reattach a citation-bearing fragment to its supporting sentence
- move a sentence from one adjacent passage to another
- rerun a gated system transform on a bounded region

Not allowed:

- typing replacement text into a passage
- rewriting quoted language
- paraphrasing a sentence
- manually changing citation text

### System Transforms

Some text normalization may still happen, but only as gated system behavior.

Examples:

- remove running headers
- repair extraction hyphenation
- rerun heuristic sentence segmentation
- `guess` a sentence repair

These are not manual text edits. They are explicit transforms over source-derived text, with clear provenance.

## Primary User

The primary user is an internal admin or operator responsible for passage quality.

They are trying to answer:

- What is wrong with this passage?
- What is the smallest structural fix?
- Did the fix preserve source fidelity?
- Can I undo it if this was wrong?

## Core Objects

- `PassageRepairCase`: one admin-visible repair situation for one opinion region
- `PassageIssue`: one classified problem inside that case
- `PassageOperation`: one primitive structural edit
- `PassageSnapshot`: a reversible view of passage state before or after an operation
- `TransformRun`: one gated system transform applied to a bounded region

## Issue Classes

The subsystem should classify passage problems in a small, useful vocabulary.

Initial issue classes:

- `page_header_artifact`
- `hyphenation_artifact`
- `bad_sentence_boundary`
- `citation_detached`
- `page_reference_detached`
- `acronym_split`
- `pin_cite_split`
- `passage_too_long`
- `passage_too_short`
- `duplicate_passage`
- `orphan_fragment`
- `unknown_issue`

These classes are not the repair actions. They help the admin understand the problem and choose a primitive operation.

## Primitive Operations

Primitive operations should be small and composable.

Initial operation set:

- `merge_with_next`
- `merge_with_previous`
- `split_at_sentence`
- `move_last_sentence_to_next`
- `move_first_sentence_to_previous`
- `drop_passage`
- `restore_passage`
- `rerun_region_cleanup`
- `guess_sentence_repair`
- `revert_last_operation`

Each operation should be defined in terms of source-derived passages and sentence boundaries.

The important design rule is that higher-level intents map down to these operations.

Examples:

- `citation_detached` often maps to `merge_with_previous`
- `page_reference_detached` often maps to `merge_with_previous`
- `passage_too_long` often maps to `split_at_sentence`
- `orphan_fragment` may map to `merge_with_previous` or `merge_with_next`

## Undo

Undo should be first-class.

The admin should feel safe trying the obvious repair because reversal is cheap.

Undo should operate on structural history, not on freeform text diffs.

Minimum undo properties:

- every operation creates a new snapshot
- the previous snapshot remains recoverable
- revert is immediate
- undo scope is local to the repaired opinion region

## Suggested Admin Flows

### 1. Inspect A Suspect Passage

The admin opens an opinion at a passage flagged by the system or discovered manually.

The view should show:

- the current passage
- adjacent passages
- the source sentence boundaries
- citations attached to each passage
- issue classifications, if any

### 2. Classify The Problem

The admin sees a lightweight classification such as:

- detached citation
- acronym split
- bad boundary
- hyphenation artifact

Classification can be system-generated first, then adjusted by the admin if needed.

### 3. Apply A Primitive Repair

The admin chooses a small operation:

- merge
- split
- move one sentence
- drop artifact
- rerun cleanup
- `guess` a repair

The result should be visible immediately in surrounding context.

### 4. Undo Or Keep Going

If the repair is wrong, undo should be immediate.

If the repair is right, the admin moves to the next problem without a separate accept step.

## System-Assisted Repair

The system can assist, but assistance stays subordinate to source fidelity and direct manipulation.

System assistance should include:

- issue detection
- issue classification
- heuristic cleanup
- optional `guess...` operations for uncertain repairs

Any LLM-backed repair operation must use `guess` in its name.

Examples:

- `guessSentenceRepair`
- `guessIssueClass`

The meaning is explicit: the model is offering a line of best fit, not ground truth.

## Interfaces To Design Later

The subsystem should eventually expose interfaces for:

- listing suspect passages in an opinion
- opening a repair case around one passage
- classifying one issue
- applying one primitive repair operation
- running one bounded system transform
- undoing the last operation
- persisting repair history

These interfaces can exist before any admin UI exists.

## Persistence Expectations

The subsystem should preserve:

- original source-derived passage state
- current repaired passage state
- operation history
- issue classifications
- transform provenance

This makes undo, audit, and future parser improvements easier.

## Non-Goals

- freeform editorial rewriting
- replacing the source document with a corrected copy
- sending the full opinion into an LLM for global repair
- turning admin repair into a multistep proposal workflow

## Implementation Shape

The likely implementation path is:

1. define the issue and operation types
2. define the structural repair interfaces
3. define snapshot and undo persistence
4. add system-generated issue detection
5. add a lightweight operator surface later

That preserves the key product principle: passage repair should feel like careful manipulation of source-derived structure, not document editing.
