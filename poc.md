# POC: Supreme Court Opinion Reader

## Purpose

This POC narrows the product to a single high-value use case: helping users read United States Supreme Court opinions in a way that preserves context, supports curiosity, and encourages firsthand understanding of the Court’s work.

Supreme Court opinions are strong candidates for the product because they are:

- dense but finite
- highly structured
- full of unfamiliar legal terms
- rich with citations to precedent and statutes
- important enough that readers benefit from direct engagement rather than summaries alone

The narrow scope also gives the product a chance to include domain-specific logic early, instead of pretending all documents are the same.

## POC User

The initial user is a civically engaged reader who wants to understand a Supreme Court opinion directly, but needs help with legal jargon, cited authorities, and the overall structure of the opinion.

## MVP Proof Target

The MVP will prove the concept by working on this specific Supreme Court opinion PDF:

https://www.supremecourt.gov/opinions/25pdf/24-777_9ol1.pdf

The first version does not need to generalize broadly. It needs to work convincingly on this one opinion end to end.

## Core POC Experience

1. User imports a Supreme Court opinion PDF.
2. System parses the opinion into logical sections such as syllabus, majority opinion, concurrence, and dissent where available.
3. System breaks each section into extremely small reading chunks.
4. User reads chunk by chunk.
5. At any point, user can ask a question anchored to a specific passage.
6. System provides a contextual explanation draft and preserves the user’s exact place in the opinion.
7. System extracts cited cases and statutes from the current chunk and shows them as related materials.

This is enough to test the core product claim: the reader can follow curiosity without losing the thread of the opinion.

## In Scope

- import one Supreme Court opinion PDF at a time
- extract core metadata such as case name, term, docket number, date, and opinion sections when possible
- chunk the opinion using legal-document structure rather than generic paragraph splitting
- keep chunks bite-sized: 3 sentences maximum, skewing toward 1 sentence
- ensure the full active chunk fits on screen at once without scrolling
- support anchored questions on a chunk or text span
- generate explanation drafts grounded in the selected passage and nearby context
- detect cited cases and U.S. Code references in the current chunk
- show a lightweight related-materials panel for cited authorities
- preserve reading progress and return-to-origin navigation

## Why This Scope Is Good

- The corpus is prestigious, bounded, and publicly available.
- Opinions have recognizable internal structure that can guide chunking.
- Citation density creates obvious opportunities for curiosity branches.
- Domain-specific enrichment is tractable: case citations and statutory citations have recognizable patterns.
- A successful experience here can later generalize to appellate cases, agency opinions, and legal scholarship.

## Product-Specific Logic Worth Building Early

### Opinion-aware parsing

The parser should try to identify:

- syllabus
- opinion announcement metadata
- majority opinion
- concurrences
- dissents
- authoring justice when available

This is better than treating the PDF as flat text because readers often want to navigate by opinion type, not just by page.

### Citation extraction

The system should detect and label:

- case citations
- U.S. Code citations
- constitutional references
- references to prior sections of the same opinion

Even a rules-based first pass is useful here.

### Legal glossary support

The system should maintain a small legal-term lookup layer for recurring procedural and doctrinal terms. This does not need to be comprehensive. The point is to reduce friction on terms that repeatedly block reading.

## Hairiest Parts for the POC

### PDF parsing quality

Supreme Court PDFs are structured, but PDF extraction can still be messy. The POC will rise or fall on whether section boundaries, citations, and paragraph flow remain reliable enough to support chunking and anchoring.

### Section-aware chunking

A syllabus should not be chunked the same way as a long dissent. The chunker should respect legal structure and avoid splitting passages in ways that destroy the argument.

For this POC, chunk size should be intentionally extreme:

- default toward 1 sentence
- never exceed 3 sentences
- preserve enough surrounding structure that the sentence still makes sense
- optimize for a reading view where the entire chunk is visible at once

### Citation handling

Citations are central to the reading experience, but matching them cleanly to useful related material is nontrivial. The first version can stop at extraction and display, without fully resolving every citation.

### Explanation grounding

Legal explanations are especially prone to sounding authoritative while being wrong. Any generated explanation should stay tightly tied to the selected passage and surrounding text.

## Smallest Credible Build

- one imported Supreme Court PDF
- parsed into major sections
- chunk list for the opinion using ultra-small chunks
- reading view for one chunk at a time
- ask-a-question flow anchored to a passage
- generated explanation draft with source context
- extracted citations shown beside the chunk
- persistent progress and return-to-origin behavior

If this works well, the product has demonstrated a real wedge.

## Success Criteria

- users can read a long opinion in several short sessions without losing their place
- each chunk is small enough to be read in one glance and fit fully on screen
- users can ask legal-context questions without leaving the opinion workflow
- cited authorities feel useful as reading aids, even if not fully resolved
- the structure of the opinion feels clearer than reading the raw PDF alone

## Next Step After the POC

If the POC succeeds, the natural next expansion is a broader Supreme Court corpus with cross-opinion linking, reusable concept branches, and better citation resolution for precedent and statutes.
