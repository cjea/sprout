# AGENTS.md

This repo is a Supreme Court opinion reader MVP. The current product target is one real opinion:

- `https://www.supremecourt.gov/opinions/25pdf/24-777_9ol1.pdf`

Everything should be built and judged against that concrete use case first.

## Product Scope

- The only document type is United States Supreme Court opinions.
- The current MVP only needs to work convincingly on the checked-in `24-777_9ol1` opinion.
- Domain-specific behavior is a feature, not a bug.
- Favor product logic for sections, citations, statutory references, and legal reading flow over generic document abstractions.

## Core Domain Language

Use the established domain names:

- `Opinion`
- `Section`
- `Passage`
- `Anchor`
- `Question`
- `AnswerDraft`
- `Citation`
- `Queue`
- `ReadingState`

Avoid vague abstractions like `Authority`, `Entity`, `Node`, `Payload`, or `Document` unless the job is truly generic.

## Passage Policy

- Passages are extremely small.
- Hard cap: `3` sentences.
- Default bias: `1` sentence.
- The full active passage should fit on screen at once.
- Chunking should preserve legal meaning, not just punctuation boundaries.

This means sentence detection is part of product quality, not just parser plumbing.

## Sentence Detection Rules

The heuristic segmenter is the baseline. It must be tested against real legal text.

Known hard cases that must be treated carefully:

- `Pp.`
- `5-13.` or `5–13.`
- `Id.`
- `, at 484.`
- trailing authority cites like `8 U. S. C. §1158(b)(1)(A).`

Trailing citations should stay attached to the sentence they support.

## Guessing And LLM Boundaries

Any probabilistic or model-backed function should be prefixed with `guess`.

Examples:

- `guessSections`
- `guessAnswer`

Current rule for sentence detection:

- heuristic segmentation runs first
- any LLM use should be optional
- LLM work should be a second-pass repair step, not the primary segmenter
- the system must allow LLM repair to be turned on or off
- default-safe behavior remains heuristic-only

If an LLM repair path is added, it should only repair suspicious fragments or bad boundaries from the heuristic pass, not re-segment the whole document blindly.

## Real Data First

Do not rely on toy examples alone.

Required real fixture:

- [`fixtures/scotus/24-777_9ol1.pdf`](./fixtures/scotus/24-777_9ol1.pdf)

When changing parsing, chunking, citation extraction, question flow, or persistence, add or update tests against the real fixture so heuristics stay directionally correct and surprises show up before production.

## Go Guidance

Follow [`golang.md`](./golang.md).

Especially important:

- name concepts, do not comment them
- keep types close to the domain
- keep interfaces narrow
- make guesses explicit
- validate at construction time
- prefer small named helpers over dense control flow
- write table-driven tests where behavior varies by case

## Browser UX Direction

Browser work should start with UX specification before implementation.

The UX spec should define:

- overall feel
- visual language
- passage length on screen
- reading rhythm
- citation reveal behavior
- question and answer flow
- return-to-reading behavior
- desktop and mobile constraints

The passage is the focal object. Side panels and metadata should not overpower the reading surface.

## Persistence

- SQLite is the default local persistence layer.
- Design with the possibility of browser-compatible SQLite in mind.
- Keep persistence seams swappable so memory-backed and SQLite-backed flows can both be tested.

## Bones

Use `bn tldr` for full bones instructions.

Track substantial work in Bones.

When creating Bones:

- make tasks concrete and testable
- mention tests explicitly
- encode real dependencies
- prefer narrow tasks over broad umbrellas

When a product or technical rule becomes stable, update Bones or this file so the rule is not lost.
