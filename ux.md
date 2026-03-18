# Browser UX Spec: Supreme Court Opinion Reader MVP

## Purpose

This document defines the browser UX for the MVP Supreme Court opinion reader.

The MVP is not a generic PDF tool. It is a passage-first reading environment for one Supreme Court opinion at a time, designed to help a reader stay oriented, follow curiosity, and return to the opinion without losing the thread.

The current proof target is:

- `https://www.supremecourt.gov/opinions/25pdf/24-777_9ol1.pdf`

## Product Posture

The product should feel:

- quiet
- serious
- precise
- literary rather than dashboard-like
- more like reading a well-edited brief than using a productivity app

The browser experience should not feel:

- noisy
- gamified
- pane-heavy by default
- visually dense in the style of legal research software
- like a raw PDF viewer

## Visual Language

The visual language should communicate public-institution seriousness without looking bureaucratic.

Direction:

- warm paper base rather than stark white
- dark ink text with strong contrast
- restrained accent color for active selection, citations, and question states
- typography that feels editorial and durable, not startup-generic
- generous whitespace around the active passage
- subtle framing instead of heavy boxes and chrome

Avoid:

- bright saturated UI chrome
- card grids
- analytics-dashboard layouts
- multiple competing panels open at once
- decorative motion that interrupts reading

## Primary Object

The active passage is the focal object.

Everything else is subordinate:

- citation details
- opinion metadata
- progress
- question history
- guessed answers

The user should always know what sentence or tiny passage they are reading right now.

## Passage Envelope

Passages are intentionally tiny.

Requirements:

- default toward `1` sentence
- never exceed `3` sentences
- the full active passage must fit on screen at once
- the user should not scroll within the passage container
- the passage should be readable in one glance, then reread slowly

Target feel:

- the passage sits in the middle of the experience with room to breathe
- even dense legal text should look approachable because only a small unit is shown

## Reading Rhythm

The product rhythm is:

1. orient
2. read one small passage
3. inspect a citation or ask a question if needed
4. return to the same place
5. continue forward

The main reading loop should feel calm and continuous. Branching into curiosity should feel possible, but never like falling into a different application mode.

## Calm, Interruptive, Hidden

### Calm

These should feel calm and ambient:

- the active passage
- section context
- reading progress
- availability of citations
- ability to ask a question

### Interruptive

These should feel interruptive on purpose:

- leaving the current passage
- submitting a question
- revealing a guessed answer
- opening a citation detail that obscures the passage

Interruptive actions should still be lightweight. They should feel like temporary excursions, not hard navigational breaks.

### Hidden By Default

These should remain hidden until needed:

- full citation detail
- question history
- explanation caveats
- opinion metadata beyond the essentials
- any debug or parser provenance

## Core Screens

### 1. Ingest / Entry

Job:

- import the opinion and begin reading

MVP behavior:

- simple entry surface
- one clear primary action
- show which opinion is being loaded
- once ready, transition directly into reading

This screen should feel operational, not ceremonial.

### 2. Reading Workspace

This is the main product screen.

It should include:

- active passage
- current section label
- minimal opinion identity
- progress indicator
- citation affordances attached to the passage
- ask-question affordance
- next / continue action

The passage should dominate the viewport. Supporting information should sit near it, not compete with it.

### 3. Answer View

The answer view should feel like a temporary expansion of the reading workspace, not a separate destination.

It should show:

- the selected quote
- the user question
- the guessed answer
- supporting evidence
- caveats
- a clear return-to-reading action

The answer surface should preserve passage context so the reader remembers what prompted the question.

## Core User Flows

### Flow: Import Opinion

1. User lands on the entry surface.
2. User imports or opens the target opinion.
3. System shows loading and parsing progress in plain language.
4. System opens the first reading passage.

Success condition:

- user reaches the reading workspace with no ambiguity about what is loaded

### Flow: Read Passage

1. User sees the active passage centered and fully visible.
2. User reads it without scrolling inside the passage.
3. User can continue to the next passage.

Success condition:

- the product feels like guided reading, not document wrangling

### Flow: Reveal Citations

1. User sees that the passage contains citations.
2. User opens citation details inline or in a secondary panel.
3. System shows the citation without displacing the reading context more than necessary.
4. User closes the citation view and resumes reading.

Success condition:

- citation inspection feels like a side glance, not a context switch

### Flow: Ask A Question

1. User selects a span or invokes question mode on the current passage.
2. User enters a question.
3. System confirms the question is anchored to the selected text.
4. System shows a guessed answer with evidence and caveats.
5. User returns to reading at the same passage.

Success condition:

- the user follows curiosity without losing orientation

### Flow: Resume Reading

1. User returns to the app.
2. System restores the current opinion and passage.
3. User can continue immediately.

Success condition:

- resuming feels instant and trustworthy

## Layout Rules

### Desktop

- a single dominant reading column
- optional secondary surface for citations or answers
- metadata minimized to a slim header or rail
- no more than one secondary surface expanded at a time

### Mobile

- one-column layout
- passage first
- supporting surfaces open as drawers, sheets, or stacked views
- when a secondary surface opens, there must be a clear path back to the active passage

## Motion

Motion should be minimal and meaningful.

Use motion for:

- opening a secondary surface
- confirming passage advancement
- transitioning into and out of answer mode

Do not use motion to decorate static reading states.

## Copy Tone

The interface copy should be:

- plain
- direct
- calm
- non-anthropomorphic

Avoid:

- hype
- cheerleading
- faux certainty in answer states

## Failure States

### Parsing Failure

- explain plainly that the opinion could not be prepared for reading
- offer retry or fallback
- avoid technical noise unless explicitly requested

### Weak Answer

- admit uncertainty
- show evidence and caveats
- keep return-to-reading obvious

### Missing Citation Resolution

- show the extracted citation text
- do not pretend full resolution exists when it does not

## UX Review Checklist

- Is the active passage the most visually dominant object?
- Does the active passage fit fully on screen?
- Does the reading workspace feel calm rather than pane-heavy?
- Can the user inspect citations without losing the passage?
- Can the user ask a question and return to reading without disorientation?
- Is the guessed answer clearly subordinate to the source passage?
- Is the visual language serious, restrained, and editorial rather than app-generic?
- Does mobile preserve the passage-first reading posture?

## Out Of Scope For This MVP

- multi-document workspace
- full legal research tooling
- collaborative annotation
- complex note management
- cross-opinion graph exploration
- highly customized reading themes
