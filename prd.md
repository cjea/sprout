# Technical PRD: Deep Reading That Survives Curiosity

## 1. Purpose

This document translates the product vision in `vision.md` into a high-level technical PRD. It is intentionally architecture-first and research-aware. The goal is not to fully specify implementation details, but to define the major product systems, interfaces between them, and the hardest unresolved problems.

Core principle:

> Deep reading should be able to survive curiosity.

The product should help users:

- import long-form documents
- break them into manageable reading units
- preserve exact reading position and context
- turn questions into structured exploration branches
- connect related ideas across documents
- convert difficult or important ideas into reviewable memory
- maintain steady progress without losing optionality

## 2. Product Summary

The system is a reading and learning platform for dense documents such as court opinions, research papers, and technical essays.

Users import a document, which is normalized and chunked into incremental reading units. While reading, users can ask questions at any point. Each question creates a branch anchored to a precise span in the source material. Branches may contain answers, notes, referenced documents, follow-up questions, and flashcards. Reading progress and exploration history are preserved in a navigable graph. The system also schedules both reading work and memory review work.

## 3. Goals

### Primary goals

- Let users make progress through large documents in short sessions.
- Preserve context when users digress into questions or related material.
- Represent exploration as structured, navigable branches rather than disconnected notes.
- Create a reusable substrate for incremental reading, annotation, question tracking, flashcard generation, and spaced repetition.
- Support cross-document linking so repeated concepts converge instead of fragmenting.

### Secondary goals

- Enable AI-assisted workflows without making the product dependent on a single model or provider.
- Make the system composable so each major capability can evolve independently.
- Capture enough provenance that generated outputs can always be traced back to source passages and user actions.

## 4. Design Principles

- Composable over monolithic: document storage, chunking, branching, cards, and scheduling should be separate systems with clear boundaries.
- Anchored context: every question, note, card, and review item should point back to a document, chunk, or text span when possible.
- User progress is first-class: the queue and return path matter as much as the content itself.
- Human-in-the-loop: generated chunks, cards, and links should be inspectable and correctable.
- Provenance by default: derived artifacts should preserve source references, generation metadata, and revision history.

## 5. Users and Jobs To Be Done

### Primary users

- readers working through dense primary sources
- students and self-learners building durable understanding
- researchers synthesizing many related documents

### Core jobs

- “Help me read a hard document in small steps.”
- “Let me ask a question without losing my place.”
- “Show me how this question relates to other things I’ve read.”
- “Turn what confused me into something I can remember.”
- “Tell me what I should read or review next.”

## 6. Core User Workflows

### 6.1 Import and prepare a document

1. User imports a document or pasted text.
2. System stores canonical source content and metadata.
3. System extracts structure and creates addressable text spans.
4. System produces reading chunks.
5. Chunks are added to the reading queue.

### 6.2 Read incrementally

1. User opens the next chunk from a document queue or global queue.
2. System restores prior reading state, annotations, and branch context.
3. User marks chunk complete, defers it, or creates new follow-up work.

### 6.3 Ask a question in context

1. User selects a span or cursor location and asks a question.
2. System creates a branch anchored to that location.
3. Branch can contain notes, generated explanations, linked references, and follow-up questions.
4. User returns to the original chunk at the exact anchor point.

### 6.4 Create flashcards from a reading branch

1. User or system identifies a concept worth reviewing.
2. Flashcard draft is generated from anchored source material and branch context.
3. User accepts, edits, or rejects the card.
4. Accepted card enters the review scheduler.

### 6.5 Discover cross-document overlap

1. System detects that two branches, chunks, or cards concern related concepts.
2. User sees suggested links with supporting evidence.
3. User accepts, rejects, or ignores the connection.

## 7. Functional Requirements by Composable System

### 7.1 Document ingestion and storage

Responsibilities:

- accept documents from supported sources
- store canonical source content and metadata
- preserve source format where useful, but normalize into an internal text model
- maintain stable identifiers for document, section, chunk, and text span

Key capabilities:

- import plain text, markdown, PDFs, URLs, and later EPUBs
- parse title, author, date, source URL, citation metadata when available
- keep both raw artifact and normalized representation
- support reprocessing without losing user-created annotations or branches

Outputs:

- `Document`
- `DocumentVersion`
- `StructuredText`
- `TextSpanIndex`

### 7.2 Chunking and reading-unit generation

Responsibilities:

- split documents into logical reading chunks
- optimize for comprehension and session length, not just token count
- maintain chunk lineage when chunking logic changes

Key capabilities:

- combine structural cues such as headings, paragraphs, citations, and semantic transitions
- allow user-editable chunk boundaries
- support multiple chunking strategies per document type
- annotate chunks with estimated reading time and difficulty

Outputs:

- `Chunk`
- `ChunkBoundary`
- `ChunkingRun`

### 7.3 Reading state and queueing

Responsibilities:

- track progress across documents and within documents
- surface “next best item” for reading
- allow multiple queues: per-document, global, deferred, priority

Key capabilities:

- mark unread, in-progress, done, skipped, deferred
- restore exact reading position and open branches
- support queue policies such as FIFO, user-priority, deadline-aware, and difficulty balancing

Outputs:

- `ReadingQueue`
- `QueueItem`
- `ReadingSession`
- `ProgressState`

### 7.4 Anchors, annotations, and branches

Responsibilities:

- attach questions and notes to exact document positions
- preserve navigability between main reading path and digressions
- represent branching exploration as a first-class data model

Key capabilities:

- text-span anchoring resilient to document reprocessing
- branch creation from chunk, paragraph, or exact span
- nested follow-up questions
- return-to-origin navigation
- branch visualization data for tree or graph views

Outputs:

- `Anchor`
- `Branch`
- `BranchNode`
- `Annotation`
- `NavigationEdge`

### 7.5 Explanation and exploration engine

Responsibilities:

- help users explore questions without destroying source context
- generate draft explanations, summaries, and follow-up prompts
- attach all generated outputs to provenance-rich context

Key capabilities:

- retrieval of relevant local context from source document and prior branches
- optional model-generated explanation drafts
- answer revision and curation
- support for importing newly discovered related documents into the same exploration tree

Outputs:

- `Question`
- `AnswerDraft`
- `EvidenceReference`
- `ExplorationSession`

### 7.6 Flashcard generation and card management

Responsibilities:

- create review items from reading friction and branch outcomes
- preserve source context so cards remain understandable later
- manage card versions and card quality signals

Key capabilities:

- support multiple card types: Q/A, cloze, concept-definition, reversible pairs
- generate cards from source spans, notes, answers, or repeated confusion patterns
- let users merge, split, suspend, or rewrite cards
- track card provenance to branch and source passage

Outputs:

- `Card`
- `CardTemplate`
- `CardSourceLink`
- `CardRevision`

### 7.7 Review scheduling

Responsibilities:

- schedule flashcard reviews
- potentially schedule reading resurfacing or re-reading tasks
- balance learning throughput with user load

Key capabilities:

- daily review queue
- card ease/difficulty updates from user feedback
- suspended/buried card behavior
- separate scheduling domains for reading progress and memory review

Outputs:

- `ReviewItem`
- `ReviewSchedule`
- `ReviewEvent`

### 7.8 Cross-document linking and concept layer

Responsibilities:

- identify related ideas across documents, branches, and cards
- suggest links without requiring a perfect global ontology
- allow an emergent concept graph to form over time

Key capabilities:

- similarity-based suggestions with evidence
- user-confirmed explicit links
- optional concept entities that can aggregate related branches and cards

Outputs:

- `Concept`
- `ConceptLink`
- `SuggestedLink`

## 8. Proposed Domain Model

The system should be decomposable around stable entities rather than UI screens.

Core entities:

- `Document`: logical source item
- `DocumentVersion`: immutable imported or normalized version
- `Chunk`: addressable reading unit within a document version
- `Anchor`: stable pointer to a chunk, paragraph, or text span
- `Branch`: a curiosity thread rooted at an anchor
- `BranchNode`: item within a branch, such as question, answer, note, linked document, or child branch
- `QueueItem`: schedulable reading work
- `Card`: schedulable memory item
- `ReviewEvent`: user interaction that updates card state
- `Concept`: optional cross-document aggregation node

Key relationships:

- one `Document` has many `DocumentVersion`s
- one `DocumentVersion` has many `Chunk`s
- one `Anchor` points into one `DocumentVersion`
- one `Branch` is rooted at one `Anchor`
- one `Branch` has many `BranchNode`s
- one `Card` may reference one or more `Anchor`s or `BranchNode`s
- one `Concept` may link many branches, cards, and anchors

## 9. System Architecture

This can be implemented as a modular application with a shared datastore and independently evolvable processing pipelines.

### 9.1 Logical components

- Ingestion service
- Normalization and parsing pipeline
- Chunking pipeline
- Queueing and scheduling service
- Branch and annotation service
- AI orchestration layer
- Search and retrieval layer
- Review scheduler
- UI application

### 9.2 Architectural stance

- Start with a modular monolith unless scale forces otherwise.
- Keep processing pipelines asynchronous and idempotent.
- Separate immutable source artifacts from mutable user state.
- Treat model calls as optional enrichments around a durable product core.

### 9.3 Storage shape

- relational store for core entities and state transitions
- object storage for raw files and large artifacts
- search index or vector index for retrieval and similarity tasks
- event log or audit tables for provenance-sensitive actions

## 10. Key Technical Decisions

### 10.1 Stable anchoring

The product depends on persistent anchors surviving re-imports, normalization changes, and user edits to derived artifacts.

Need:

- location model stronger than raw character offsets alone
- rebasing strategy when source versions change
- confidence scoring when anchor restoration is fuzzy

### 10.2 Chunking strategy

Chunking should optimize for human reading flow, not just arbitrary length.

Need:

- heuristic plus model-assisted chunking evaluation
- support for domain-specific chunking, especially legal and academic text
- user correction loop so bad chunking improves over time

### 10.3 Queue design

Reading queue logic must reconcile:

- document-local progress
- global “what next” prioritization
- deferred branches
- flashcard review load

This is both a product and systems problem, not just a sorting function.

### 10.4 AI boundaries

AI can help with:

- chunking assistance
- explanation drafts
- card generation
- concept-link suggestions

AI must not be the source of truth for:

- source content
- user progress state
- final accepted links or cards

## 11. Hairiest Problems and Research Tracks

This section names the hardest areas that deserve dedicated research before detailed implementation.

### 11.1 Spaced repetition model choice

Why it is hairy:

- The right review intervals materially affect product value.
- Different content types may need different scheduling behavior.
- Reading-derived cards may behave differently from manually authored cards.

Questions to answer:

- Should the initial implementation use an established scheduler such as SM-2, FSRS, or another variant?
- Should scheduling be card-type aware?
- What user feedback inputs are required: Again/Hard/Good/Easy, pass-fail, or something else?
- Should the system schedule only cards, or also reading resurfacing and branch follow-ups?

Recommendation:

- Start with a proven, inspectable algorithm rather than inventing one.
- Treat scheduler choice as a configurable module.
- Preserve review history so migration between algorithms is possible later.

### 11.2 Anchor durability

Why it is hairy:

- If branches or cards lose their connection to the original text, the product’s core promise breaks.

Questions to answer:

- How should anchors be represented: offsets, quote selectors, structural paths, or hybrids?
- How should anchors survive new document versions and OCR noise?
- What confidence threshold should trigger user repair?

### 11.3 Chunk quality

Why it is hairy:

- Poor chunking creates either cognitive overload or fragmentation.
- Different document genres have different natural boundaries.

Questions to answer:

- What objective and subjective signals define a “good” chunk?
- How should chunk size vary by user expertise and document type?
- When should the system prefer preserving original structure over equalizing reading time?

### 11.4 Branch explosion and graph usability

Why it is hairy:

- Curiosity can create combinatorial growth.
- A branch graph can become unreadable long before the dataset becomes large.

Questions to answer:

- What constraints or defaults prevent exploration clutter?
- Should branch depth or visibility be managed automatically?
- What is the minimal graph model that still feels navigable?

### 11.5 Cross-document concept linking

Why it is hairy:

- False-positive links quickly erode trust.
- High precision often conflicts with broad discovery.

Questions to answer:

- When should links be suggested versus created automatically?
- What evidence should accompany a suggestion?
- Should concepts be explicit user-managed objects in v1, or inferred later?

### 11.6 AI output quality and provenance

Why it is hairy:

- Generated explanations and cards can sound plausible while being wrong.
- Users need to understand what came from the source versus the model.

Questions to answer:

- How should source quotations and evidence be surfaced in generated outputs?
- Which tasks require mandatory user confirmation?
- How should hallucination risk differ for legal, scientific, and technical texts?

## 12. MVP Scope

The MVP should prove that incremental reading plus anchored curiosity creates better progress than ordinary document reading.

### In scope

- import text, markdown, PDF, and URL content
- normalize and store documents
- generate editable chunks
- global and per-document reading queues
- anchored questions and branch creation
- branch navigation back to source
- manual notes and AI-assisted explanation drafts
- flashcard drafting from anchored context
- basic spaced-repetition scheduling for approved cards

### Out of scope

- collaborative workspaces
- automatic web research across arbitrary sources
- advanced concept graph editing
- fully automated flashcard creation without review
- algorithmically sophisticated cross-document linking beyond simple suggestions

## 13. Success Metrics

### Product metrics

- document completion rate
- average chunk completion per week
- rate of return to original reading position after asking a question
- percentage of branches that lead to resumed reading
- flashcard acceptance rate
- review completion rate

### Quality metrics

- anchor restoration success rate
- user edits per auto-generated chunk or card
- branch navigation failure rate
- false-positive rate on suggested cross-document links

## 14. Risks

- Low-quality chunking makes the reading experience worse than the source.
- Fragile anchors break trust in branching and card provenance.
- Queue complexity overwhelms users instead of simplifying choices.
- AI-generated content feels useful but is subtly inaccurate.
- The graph UI becomes novel but not actually helpful.
- Review burden can crowd out reading rather than reinforce it.

## 15. Suggested Implementation Phases

### Phase 1: Durable reading substrate

- ingestion
- normalization
- stable anchors
- chunk generation
- reading queues
- progress tracking

### Phase 2: Curiosity substrate

- anchored questions
- branch model
- branch navigation
- notes and explanation drafts

### Phase 3: Memory substrate

- flashcard generation
- approval flow
- review scheduler
- review history

### Phase 4: Knowledge network

- related-branch suggestions
- cross-document link proposals
- early concept layer

## 16. Open Questions for Breakdown

- What document types matter most in v1: legal, academic, technical, or mixed?
- Is the initial queue model optimized for “always tell me what to do next” or for manual control?
- Should branches be represented as strict trees, DAGs, or graph-backed data with tree-like UI defaults?
- Does the MVP require OCR and citation-aware parsing, or can imported text quality be assumed?
- Should flashcards be user-created with AI assistance, or AI-drafted by default and then edited?
- Does “cross-document discovery” in v1 mean explicit links only, or ranked suggestions?
- Should review scheduling cover only flashcards, or also resurfacing unfinished chunks and branches?

## 17. Summary

The product should be built as a set of composable systems centered on anchored reading state:

- import and preserve documents
- transform documents into manageable chunks
- let questions branch from exact source locations
- turn branches into notes, answers, and cards
- schedule both reading and review work
- gradually connect repeated ideas across documents

The most important technical truth is that this is not primarily an AI product. It is a state, provenance, queueing, and navigation product with AI-assisted generation layered on top. The architecture should reflect that from the beginning.
