# Supreme Court Fixtures

This directory contains real-data fixtures for the Supreme Court opinion reader MVP.

Canonical MVP fixture:

- `24-777_9ol1.pdf`
  Source: `https://www.supremecourt.gov/opinions/25pdf/24-777_9ol1.pdf`
  Case: `Urias-Orellana v. Bondi`

Supporting files:

- `24-777_9ol1.expect.json`
  Checked-in expectations for metadata, section markers, and citation-bearing passages.
- `24-777_9ol1.excerpts.txt`
  Human-readable excerpts derived from the real opinion and used to lock the expectations.
- `24-777_9ol1.quality.json`
  Passage-quality corpus covering sentence-boundary edge cases, quote/citation cohesion, page-header artifacts, and line-break hyphenation cleanup.

The PDF hash in the expectation file is the integrity check for the real source artifact. If the source PDF is ever refreshed upstream, update the hash and re-validate every expectation before changing the checked-in fixture set.
