# Browser Routing And State Spec: MVP Reader

## Purpose

This document defines the route structure and state model for the browser-based MVP reader.

The governing rules are:

- the current reading location should be addressable
- URL state should be used wherever possible
- browser refresh should preserve useful context
- branching UI should remain subordinate to the active passage

## State Layers

The MVP has three layers of state:

1. durable document state
2. durable reading state
3. ephemeral UI state

## 1. Durable Document State

This identifies what opinion and passage the user is looking at.

Examples:

- opinion id
- current passage id
- current section id if needed for route readability

This state belongs in the URL.

## 2. Durable Reading State

This captures where the reader left off and should survive refresh or return.

Examples:

- last active passage
- completion state
- saved questions

This state belongs in persistence first, with URL mirroring where useful.

## 3. Ephemeral UI State

This captures temporary interface posture.

Examples:

- citation panel open
- question compose open
- answer view open
- selected span

This state should be expressed in the URL when it materially changes the visible screen and is worth restoring or sharing.

## Route Model

The MVP should use a small route surface:

1. landing / import route
2. opinion reading route

Suggested routes:

- `/`
- `/opinions/:opinionId`

Everything else should be encoded as route state, query params, or hash-like UI state on the opinion route rather than proliferating pages.

## Base Route

### `/`

Purpose:

- entry and opinion load

Contains:

- import/open action
- last-known opinion shortcut if available

Should not contain:

- active reading workspace

## Opinion Route

### `/opinions/:opinionId`

Purpose:

- main reading workspace

Required route identity:

- `opinionId`

Required query state:

- `passage`

Suggested example:

- `/opinions/24-777_9ol1?passage=24-777_9ol1-p001-s001`

The app should not open an opinion route without a resolvable active passage.

If the passage is missing:

- restore from saved progress if possible
- otherwise resolve the first readable passage
- then normalize the URL

## Secondary Surface URL State

Secondary surfaces should be encoded in the URL when open.

Suggested query keys:

- `panel=citation`
- `panel=question`
- `panel=answer`

Optional supporting query keys:

- `citation=<citationId>`
- `question=<questionId>`

Examples:

- `/opinions/24-777_9ol1?passage=24-777_9ol1-p001-s001&panel=citation&citation=cit-12`
- `/opinions/24-777_9ol1?passage=24-777_9ol1-p001-s001&panel=question`
- `/opinions/24-777_9ol1?passage=24-777_9ol1-p001-s001&panel=answer&question=q-7`

Rules:

- only one `panel` value at a time
- panel state must be removable without changing the active passage
- closing a panel should preserve the base opinion route and passage

## Selection State

Selected span state should be URL-addressable where practical because it materially changes question flow and may be worth restoring.

Suggested query keys:

- `start=<offset>`
- `end=<offset>`

Example:

- `/opinions/24-777_9ol1?passage=24-777_9ol1-p001-s001&panel=question&start=0&end=32`

Rules:

- selection offsets are only valid relative to the active passage
- invalid offsets should be rejected and stripped from the URL
- selection state should not exist without a compatible active passage

## URL Normalization Rules

- `opinionId` must resolve to a known opinion
- `passage` must resolve to a passage belonging to that opinion
- `citation` must resolve to a citation belonging to the active passage
- `question` must resolve to a question belonging to the active opinion and passage context where required
- `panel` must be one of the supported values
- invalid combinations should be normalized to the closest safe state, not crash the workspace

## Refresh Behavior

On refresh:

1. read the URL
2. validate route identity and query state
3. load opinion and passage
4. restore panel state if valid
5. restore selection state if valid
6. render the workspace

If some UI state is invalid:

- preserve the valid base route
- drop only the bad state
- keep the user reading

## Shareability Rules

Shareable URLs should support:

- sending another reader to a specific opinion
- landing on a specific passage
- optionally opening a specific citation panel

Question and answer states may be shareable later, but MVP only needs route-level support precise enough to restore them for the current user.

## Persistence vs URL

### URL Should Hold

- opinion id
- active passage id
- open panel mode
- selected span offsets when relevant
- citation id when citation detail is open
- question id when answer review is open and stable

### Persistence Should Hold

- progress
- completed passages
- saved questions
- last-resumed state

### UI Memory Should Hold

- temporary unsaved input text
- hover state
- transient animation state

## Browser Navigation Rules

Back and forward navigation should work with user expectations:

- changing passage should create meaningful navigation history only if passage-to-passage movement is treated as navigable state
- opening and closing a major panel should be navigable if it materially changes the screen
- tiny transient interactions should not spam history

Recommended MVP rule:

- keep passage changes addressable in the URL
- use replace vs push intentionally so the history stack stays usable

## Route Review Checklist

- Can the current opinion and passage be restored from the URL alone?
- Does refresh keep the user in a sensible state?
- Can invalid panel or selection state be dropped without breaking reading?
- Are panel states encoded without creating route sprawl?
- Is the URL precise enough to support shareable passage links?
- Does browser back behave like closing a branch rather than destroying orientation?

## Out Of Scope

- many top-level routes
- full multi-document route architecture
- collaborative shared annotations in URL state
- public deep-link guarantees for every ephemeral UI detail
