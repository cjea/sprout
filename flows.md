# Browser Flow Spec: Reading And Questions

## Purpose

This document defines the browser interaction flows for reading, selecting text, asking a question, reviewing a guessed answer, and returning to the passage.

The governing constraint is continuity:

- the user should be able to follow curiosity without losing their place in the opinion

## Flow Principles

- the active passage remains the anchor state
- branching actions should feel temporary
- one major interaction mode at a time
- every branch has a clear return path
- failure states should preserve orientation rather than eject the user

## Primary Modes

The reading workspace has four primary interaction modes:

1. reading
2. citation detail
3. question compose
4. answer review

The user should always be able to tell which mode they are in.

## Flow: Open Opinion Into Reading

### Entry Conditions

- user has selected or imported an opinion

### Steps

1. System shows the selected opinion identity.
2. System shows loading/preparation status in plain language.
3. System prepares sections, passages, citations, and starting progress state.
4. System opens the first available passage in reading mode.

### Visible State

- top identity bar
- active passage
- local action strip
- continuity footer

### Success Outcome

- user lands in reading mode with a clear active passage

### Failure State

If opinion preparation fails:

- show a concise failure message
- preserve the opinion identity if known
- offer retry
- do not drop the user into a blank app shell

## Flow: Read Current Passage

### Entry Conditions

- user is in reading mode

### Steps

1. User sees the current passage centered and fully visible.
2. User reads the passage.
3. User either continues forward or opens a branch action.

### Interaction Rules

- the passage must be readable without scrolling inside its container
- no secondary surface is open by default
- citation affordances and question affordances are visible but quiet

### Success Outcome

- reading feels calm and uninterrupted

## Flow: Continue To Next Passage

### Entry Conditions

- user is in reading mode on a valid current passage

### Steps

1. User activates the continue action.
2. System marks the current passage complete if that is the current rule.
3. System loads the next passage in sequence.
4. System updates progress.
5. System presents the next passage in the same reading workspace.

### Interaction Rules

- advancement should feel like moving through a text, not loading a new page
- orientation elements must update immediately
- the user should not lose context of which section they are in

### Success Outcome

- the user keeps reading with minimal transition overhead

### Empty State

If there is no next passage:

- show a clear end-of-opinion state
- preserve opinion identity and completion status
- offer a way to review prior passages or reopen citations/questions if applicable

## Flow: Select Text

### Entry Conditions

- user is in reading mode

### Steps

1. User selects a span within the active passage.
2. System highlights the selected quote.
3. System reveals the ask-question action in a stronger state.
4. System preserves the rest of the workspace.

### Interaction Rules

- selection should stay local to the active passage
- selection should not immediately force open a panel
- the selected quote must remain visible when the next step opens

### Success Outcome

- the user can express curiosity without leaving the reading context

### Failure State

If selection is invalid:

- reject it quietly
- keep the user in reading mode

## Flow: Reveal Citation Detail

### Entry Conditions

- active passage contains one or more citations

### Steps

1. User activates a citation marker or citation affordance.
2. System opens citation detail mode in the secondary detail surface.
3. System shows the citation text, type, and any normalized label.
4. User closes the citation detail surface.
5. System returns to reading mode with the same passage still active.

### Interaction Rules

- only citation detail opens, not a full research pane
- the active passage remains visible
- only one citation detail context should be open at a time

### Success Outcome

- citation inspection feels like a temporary side glance

### Empty State

If a citation marker has no enriched detail:

- still show the extracted citation text
- do not imply full resolution exists

### Failure State

If citation detail cannot be loaded:

- show a compact error inside the secondary surface
- keep the passage visible
- offer close or retry

## Flow: Open Question Composer

### Entry Conditions

- user has selected a valid span

### Steps

1. User activates ask-question.
2. System opens question compose mode in the secondary detail surface.
3. System shows the selected quote and passage identity.
4. System focuses the question input.
5. User enters a question.

### Interaction Rules

- the selected quote must remain visible while composing
- the passage must remain visible enough to preserve context
- opening compose mode should not clear the selection

### Success Outcome

- user understands exactly what text the question is anchored to

### Empty State

Before the user types:

- show placeholder guidance in plain language
- do not overload the composer with instructions

## Flow: Submit Question And Review Answer

### Entry Conditions

- user is in question compose mode with a valid question

### Steps

1. User submits the question.
2. System saves the question against the selected anchor.
3. System shows a compact loading state in place.
4. System gathers local context.
5. System presents guessed answer mode in the same secondary surface.

### Visible State

- selected quote
- user question
- guessed answer
- evidence
- caveats
- return-to-reading action

### Interaction Rules

- answer review should feel like an expansion of the current branch, not a page navigation
- evidence and caveats should be visible without drowning the answer
- the selected quote anchors trust

### Success Outcome

- the user gets help while staying oriented to the exact passage

### Failure State

If answer generation fails:

- preserve the saved question if possible
- show a compact failure state in the same surface
- keep the selected quote visible
- offer retry and close

## Flow: Return To Reading

### Entry Conditions

- user is in citation detail, question compose, or answer review mode

### Steps

1. User activates close, done, or return-to-reading.
2. System closes the secondary detail surface.
3. System restores base reading mode on the same passage.
4. System preserves progress and, when relevant, selection memory or question history state.

### Interaction Rules

- return should be immediate
- the user should not be dropped at a different passage
- return should feel like closing a temporary detour

### Success Outcome

- orientation is preserved

## Flow: Resume Reading Later

### Entry Conditions

- user has an existing opinion and saved progress

### Steps

1. User reopens the app or revisits the opinion.
2. System restores the last active passage.
3. System opens directly into reading mode.
4. User can continue immediately or inspect the saved branch context if exposed.

### Success Outcome

- resume feels trustworthy and instant

### Failure State

If progress cannot be restored:

- reopen at the best known passage
- explain the fallback briefly
- avoid forcing the user to restart unless necessary

## Empty States

### No Citation In Passage

- citation affordance remains absent or inert
- do not show a dead panel

### No Question Yet

- question history remains hidden
- reading workspace should not advertise empty side structures

### No Next Passage

- show end-of-opinion completion state
- keep the user inside the opinion context

## Failure-State Rules

- failures should appear where the user initiated the action
- failures should not collapse the reading workspace
- the active passage should remain visible during recoverable failures
- recovery paths should be obvious and low-friction

## Review Criteria

- Can the user read several passages without accidental mode switching?
- Does text selection feel local and reversible?
- Can citation inspection be opened and closed without losing reading continuity?
- Does question compose preserve the selected quote clearly?
- Does answer review remain visually subordinate to the active passage?
- Is every branch action paired with an obvious return-to-reading path?
- Do failure states preserve the current passage and orientation?

## Out Of Scope

- multi-step wizard flows
- persistent split view with several open branches
- multi-question management workflow inside the reading loop
- full citation research flows
