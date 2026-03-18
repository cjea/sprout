# Passage Repair Flows

This document defines the lightweight operator flows for passage repair.

The interaction should feel immediate:

- open the suspect region
- inspect surrounding passages
- classify what is wrong
- apply one structural operation
- undo if needed
- move on

There is no separate propose-and-accept phase.

## 1. Open Case

The operator opens a repair case around one focused passage.

The case should show:

- the focused passage
- the previous passage
- the next passage
- attached citations
- source-derived sentence boundaries

## 2. Inspect Context

The operator decides whether the passage is actually wrong.

Common examples:

- citation detached into the next passage
- `Pp. 3-10.` detached from the held sentence
- acronym or pin cite split
- orphan fragment

## 3. Classify Issue

The operator applies or confirms a lightweight issue class.

Examples:

- `citation_detached`
- `page_reference_detached`
- `bad_sentence_boundary`
- `hyphenation_artifact`

Classification exists to help pick the next action. It is not the action itself.

## 4. Apply Primitive Operation

The operator applies one structural repair:

- merge with next
- merge with previous
- split at sentence
- move one sentence to adjacent passage
- drop passage artifact
- restore dropped passage
- rerun a gated system transform
- `guessSentenceRepair`

The result should appear immediately in context.

## 5. Undo

If the repair is wrong, undo should be immediate.

Undo should restore the previous structural snapshot without requiring a separate review step.

## 6. Continue

Once the passage looks right, the operator moves to the next suspect case.

The system should preserve:

- snapshot history
- issue classification
- operation history
- transform provenance
