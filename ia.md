# Browser Information Architecture: Reading Workspace

## Purpose

This document defines the major regions of the browser reading workspace and what information belongs in each one.

The governing rule is simple:

- the active passage is the primary focal object

Everything else exists to support reading that passage, not to compete with it.

## IA Principles

- one dominant object per screen state
- passage first
- supporting surfaces appear near the reading act
- secondary information stays collapsed until invited
- no more than one major secondary surface expanded at a time
- the user should always know what opinion, section, and passage they are in

## Workspace Regions

The MVP reading workspace has five regions:

1. top identity bar
2. passage stage
3. local action strip
4. secondary detail surface
5. continuity footer

## 1. Top Identity Bar

Purpose:

- establish orientation with minimal overhead

Contains:

- case name
- docket number or short opinion identifier
- current section title
- lightweight progress signal

Should not contain:

- full metadata dump
- citation detail
- question history
- dense navigation trees

Behavior:

- always visible on desktop
- compact and sticky if needed
- visually subordinate to the passage

The identity bar answers:

- what am I reading?
- where am I in the opinion?

## 2. Passage Stage

Purpose:

- present the active passage as the main reading object

Contains:

- active passage text
- inline selection affordance
- inline citation markers or citation highlights
- minimal section context if needed for comprehension

Should not contain:

- full answer explanation by default
- large metadata blocks
- stacked historical passages
- open side content by default

Behavior:

- centered in the viewport
- fully visible at once
- visually dominant
- optimized for rereading a very small amount of text

The passage stage answers:

- what exactly should I read right now?

## 3. Local Action Strip

Purpose:

- expose the next relevant actions without forcing a mode switch

Contains:

- continue / next passage action
- reveal citations action
- ask-question action
- optional back-to-current-passage action when returning from a branch

Should not contain:

- broad application navigation
- multiple competing CTA groups
- configuration controls

Behavior:

- lives immediately below or adjacent to the passage stage
- actions are few and legible
- labels should be direct and quiet

The action strip answers:

- what can I do from this passage right now?

## 4. Secondary Detail Surface

Purpose:

- temporarily reveal supporting context without replacing the reading workspace

This is one surface with multiple modes, not several panels open at once.

Modes:

- citation detail
- question composer
- guessed answer
- question history, if exposed at all in MVP

Rules:

- only one major secondary mode open at a time
- opening this surface should not make the active passage disappear completely
- the user should be able to close it and return to the passage instantly

### Citation Detail Mode

Contains:

- extracted citation text
- normalized citation label when available
- short type label such as case or statute

Does not contain:

- a full legal research workspace
- unrelated citations from elsewhere in the opinion

### Question Composer Mode

Contains:

- selected quote
- question input
- confirmation of what text is anchored

Does not contain:

- large historical context blocks by default

### Guessed Answer Mode

Contains:

- selected quote
- user question
- guessed answer
- evidence
- caveats
- return-to-reading action

Does not contain:

- hidden model internals by default
- long-running branching UI

The secondary surface answers:

- what supporting context do I need right now?

## 5. Continuity Footer

Purpose:

- preserve momentum at the end of the current passage interaction

Contains:

- next-passage affordance
- concise progress status
- optional reminder of open question state if relevant

Should not contain:

- large navigation trees
- unrelated settings

Behavior:

- visible when the user reaches the lower interaction boundary of the current passage state
- should reinforce continuation, not distract from reading

The continuity footer answers:

- how do I keep going?

## Information Placement By Type

### Active Passage

Lives in:

- passage stage

Priority:

- highest

### Opinion Identity

Lives in:

- top identity bar

Includes:

- case name
- short identifier
- section label

Priority:

- low visual weight, high orientation value

### Progress

Lives in:

- top identity bar
- continuity footer

The top bar gives ambient position.
The footer reinforces forward motion.

### Citation Detail

Lives in:

- secondary detail surface

The passage may show markers, but the explanation of a citation belongs in the secondary surface.

### Question Context

Lives in:

- secondary detail surface

The anchor quote should remain visible whenever question or answer UI is open.

### Guessed Answer

Lives in:

- secondary detail surface

It should never replace the reading workspace entirely in MVP.

### Full Metadata

Lives in:

- deferred detail view, not the default workspace

The MVP does not need a metadata-heavy reading screen.

## Desktop IA Shape

- top identity bar spans the workspace
- passage stage occupies the dominant center column
- local action strip stays attached to the passage stage
- secondary detail surface opens to the side or as a controlled overlay
- continuity footer remains near the reading flow, not detached in app chrome

Desktop should feel like one reading room with one temporary reference desk.

## Mobile IA Shape

- top identity remains compact
- passage stage takes the main column
- local actions stack directly with the passage
- secondary detail surface opens as a sheet or drawer
- continuity footer compresses into the bottom flow

Mobile should feel like reading a page and briefly sliding out context, not opening a second app layer.

## Dominance Rules

These are the review rules that keep the passage primary:

- the active passage must be the largest and most visually central object
- supporting surfaces must not outsize the passage by default
- metadata must not occupy more visual weight than the passage
- opening citation or answer detail must preserve clear passage orientation
- the user must always have an obvious path back to the active passage

## Workspace States

### Base Reading State

Visible:

- top identity bar
- passage stage
- local action strip
- continuity footer

Hidden:

- secondary detail surface

### Citation Open State

Visible:

- top identity bar
- passage stage
- local action strip
- citation mode in secondary detail surface

### Question Compose State

Visible:

- top identity bar
- passage stage
- anchored quote
- question composer in secondary detail surface

### Answer Review State

Visible:

- top identity bar
- passage stage
- guessed answer in secondary detail surface
- return-to-reading path

## IA Review Checklist

- Can a new user identify the active passage within one second?
- Is opinion identity present without crowding the reading surface?
- Do citation details live outside the passage until requested?
- Does question UI preserve the selected quote and anchor context?
- Is there only one major secondary surface open at a time?
- Does the workspace still feel like reading when answer UI is visible?
- On mobile, can the user close a secondary surface and immediately recover the passage?

## Out Of Scope

- full outline tree of the whole opinion
- multiple simultaneous side panels
- research tabs
- general-purpose notes dashboard
- persistent multi-document navigation inside the reading workspace
