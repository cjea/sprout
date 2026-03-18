# Responsive Layout Spec: Passage-First Reading

## Purpose

This document defines the responsive layout behavior for the browser reader so the active passage remains fully visible and readable across desktop and mobile.

The governing rule is:

- the active passage must remain the visual and spatial center of the experience

## Layout Principles

- preserve one-glance readability of the active passage
- avoid multi-panel clutter on smaller screens
- let secondary surfaces expand only when invited
- keep interaction close to the passage
- prefer vertical clarity over feature density

## Breakpoints

The MVP should use three layout bands:

1. mobile
2. tablet / narrow desktop
3. desktop

Suggested breakpoints:

- mobile: `< 768px`
- tablet / narrow desktop: `768px - 1199px`
- desktop: `1200px+`

These are behavioral breakpoints, not just CSS thresholds. The surface rules change with them.

## Shared Constraints

Across all breakpoints:

- the active passage must fit fully on screen without internal scrolling
- only one major secondary surface may be open at once
- citation detail, question compose, and answer review all count as secondary surfaces
- progress and identity must remain visible without dominating the workspace
- scroll should belong to the page, not to nested passage containers

## Passage Column

The passage column is the main reading column.

Rules:

- never stretch passage text too wide
- keep comfortable line length
- keep enough surrounding whitespace that the passage feels isolated and intentional
- preserve a stable text block so advancing to the next passage does not reflow the entire interface dramatically

Suggested target:

- ideal reading width around `40rem - 52rem`
- wider screens should add margin, not paragraph width

## Vertical Rhythm

The vertical rhythm should feel editorial, not application-dense.

Rules:

- generous top breathing room above the passage
- compact but clear identity bar
- passage stage gets the most vertical space
- action strip sits close enough to feel attached
- footer reinforces continuity without looking like app chrome

The user should visually read the screen in this order:

1. identity
2. passage
3. local actions
4. continuation

## Desktop Layout

### Shape

- centered reading column
- optional right-side secondary surface
- minimal header above
- footer attached to the reading flow

### Desktop Width Rules

- passage column remains fixed within the target reading width
- secondary surface may open beside the passage, not above it
- the passage must remain fully legible when the secondary surface is open
- if the viewport is too narrow to preserve both comfortably, collapse to the tablet behavior

### Desktop Secondary Surface

When closed:

- the passage column appears centered

When open:

- a secondary panel opens to the side
- it should feel like a temporary reference surface
- it must not visually dominate the passage

### Desktop Scrolling

- page-level scroll only
- no independent scroll region inside the passage
- secondary surface may scroll internally only if necessary, but the default answer/citation payload should fit comfortably

## Tablet / Narrow Desktop Layout

### Shape

- primary reading column remains dominant
- secondary surface uses a narrower side panel or controlled overlay depending on available width

### Rules

- prioritize passage readability over keeping the side panel permanently visible
- if side-by-side makes the passage cramped, the secondary surface should overlay or dock below the passage instead
- the user should still feel in one workspace, not in a modal stack

## Mobile Layout

### Shape

- single-column layout
- compact identity bar
- passage stage first
- action strip immediately follows the passage
- secondary surface opens as a bottom sheet, drawer, or full-height sheet with clear return affordance

### Mobile Rules

- passage width should feel comfortable and uncluttered
- metadata must compress aggressively
- no persistent side rail
- citation and answer surfaces should slide over the reading flow rather than shrink the passage into a cramped strip

### Mobile Secondary Surface

When open:

- selected quote or citation anchor stays visible near the top
- close / return control is always obvious
- the user must be able to dismiss the surface and land exactly back at the passage

### Mobile Scrolling

- page scroll remains the default in reading mode
- secondary surfaces may scroll internally because mobile height is limited
- the top of the secondary surface should keep context visible before deep content begins

## Panel Behavior

### Citation Detail

- desktop: side panel or contained overlay
- tablet: side panel if comfortable, otherwise overlay
- mobile: bottom sheet or full-height sheet

### Question Compose

- should feel lighter than a full modal
- anchored quote remains visible
- the input field should appear without displacing the passage completely on larger screens

### Answer Review

- can use more height than citation detail
- must still preserve a clear return path
- should not feel like route navigation on its own

## Spacing Rules

- passage text gets the largest spacing budget
- metadata gets the smallest spacing budget
- action strip spacing should suggest immediacy
- secondary content spacing should support scanning, not sprawl

## Overflow Rules

### Passage Overflow

If a passage does not fit cleanly:

- the problem is upstream in chunking
- do not solve it with a tiny scroll box
- surface a layout error or repair state if needed during development

### Secondary Surface Overflow

If answer or citation content exceeds available height:

- allow secondary-surface scrolling
- preserve top anchor information
- keep return affordance pinned or easy to reach

## Responsive Review Checklist

- Does the active passage stay fully visible on desktop, tablet, and mobile?
- Does the passage remain the largest and most central object?
- Do wider screens add breathing room rather than text sprawl?
- On mobile, do secondary surfaces open without destroying passage continuity?
- Is there at most one major secondary surface open at a time?
- Are scroll boundaries obvious and minimal?
- Does the layout still feel editorial rather than app-dense at every breakpoint?

## Out Of Scope

- power-user multi-column layouts
- configurable pane resizing
- side-by-side comparison of multiple passages
- landscape-phone specialized layout beyond normal responsive behavior
