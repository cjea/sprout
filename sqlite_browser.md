# SQLite Browser Constraints

The MVP should keep SQLite usage compatible with both local native execution and a possible browser-hosted SQLite deployment later.

## Constraints

- Keep the schema in portable SQLite SQL.
- Avoid loadable extensions.
- Avoid virtual tables unless there is a clear browser-compatible plan.
- Keep PDF storage as plain `BLOB` data.
- Keep migrations as ordered `.sql` files rather than Go-only imperative setup.
- Keep transaction boundaries at the application layer, not in driver-specific hooks.

## Current Direction

- One primary schema file initializes the MVP domain tables.
- Domain rows use plain scalar columns and foreign keys.
- Raw PDFs are stored as `BLOB`.
- Opinion, section, passage, citation, progress, and question state use ordinary relational tables.

## Browser Implications

- The schema should work with WASM-hosted SQLite implementations because it relies on ordinary tables, foreign keys, and transactions.
- Large PDF blobs may eventually need a split storage strategy if browser quotas become a problem, but the current schema keeps that choice open.
- The `Storage` interface remains the portability seam, so browser-backed SQLite can slot in without rewriting the product flow.
