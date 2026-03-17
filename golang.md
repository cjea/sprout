# Guide For Writing Go

This project should use Go for clarity, explicitness, and small composable types. Prefer code that reads like a sequence of named domain decisions, not a compressed trick.

## Name Concepts, Don't Comment Them

Code that uses a concept should not also be defining it. If you need a comment to explain what an expression means, make it a named variable or function instead. The name lives at the definition; the usage site stays clean.

```go
// bad: definition and use are tangled
if !opponentAttacks.Has(pieceSquare) { // not attacked
	continue
}

// good: concept defined once, used clearly
attacked := opponentAttacks.Has(pieceSquare)
if !attacked {
	continue
}
```

The same applies to filter predicates, conditions, and any expression whose meaning is not self-evident from the surrounding code.

A good heuristic is to look at data types. When a line mixes domain values with primitives, there is a high chance that a concept is being defined.

```go
// bad: the conditional defines and uses a new concept
if mateIndex != -1 && state.CompletedSteps.Has(mateIndex) {
	state.CompleteAllSteps()
}

// good: the concept is named once, then used clearly
mateCompleted := state.CompletedMate()
if mateCompleted {
	state.CompleteAllSteps()
}
```

Even better, if the concept truly belongs to the type, move it onto the type:

```go
if state.CompletedMate() {
	state.CompleteAllSteps()
}
```

## Prefer Small Named Functions Over Clever Control Flow

If a block needs explanation, split it. Go is easier to read when the main flow is short and subordinate details are moved into named helpers.

```go
func BuildOpinion(raw RawPDF) (Opinion, error) {
	parsed, err := ParsePDF(raw)
	if err != nil {
		return Opinion{}, err
	}

	meta := ExtractMeta(parsed)
	sections, err := GuessSections(parsed)
	if err != nil {
		return Opinion{}, err
	}

	return AssembleOpinion(raw.OpinionID, meta, sections, parsed), nil
}
```

## Keep Types Close To The Domain

Name things after the product, not after generic software categories.

- Prefer `Opinion`, `Section`, `Passage`, `Question`.
- Avoid generic names like `DocumentNode`, `Entity`, `Item`, or `Payload` unless they are truly generic.
- If a type exists only for Supreme Court opinions, let the code admit that.

## Make Guesses Explicit

Any function that depends on an LLM or probabilistic inference should be prefixed with `guess`.

Examples:

- `guessSections`
- `guessAnswer`
- `guessCitationMeaning`

This keeps uncertainty visible in the code and encourages repair paths, user review, and test doubles around the guessing boundary.

## Push Validation To Construction Time

If a type has invariants, enforce them where the type is built.

```go
func NewPassage(sectionID SectionID, text string, start, end int) (Passage, error) {
	if text == "" {
		return Passage{}, errors.New("passage text is required")
	}
	if start > end {
		return Passage{}, errors.New("sentence range is invalid")
	}

	return Passage{
		SectionID: sectionID,
		Text:      text,
		Start:     start,
		End:       end,
	}, nil
}
```

Call sites should not have to remember invisible rules.

## Keep Interfaces Narrow

Define interfaces where they are consumed, not where they are implemented.

```go
type SectionGuesser interface {
	GuessSections(ParsedPDF) ([]Section, error)
}
```

Small interfaces make tests simpler and keep the code honest about what it actually needs.

## Return Concrete Types From Core Domain Code

Use concrete structs for the main domain model. Use interfaces for seams like storage, clocks, HTTP fetchers, and LLM-backed guessers.

- Domain core: concrete
- Infrastructure seams: interfaces

That keeps the product model easy to inspect and easy to test.

## Keep Error Paths Plain

Do not hide failure behind clever abstractions. In Go, explicit error returns are part of the design.

```go
passage, err := repo.LoadPassage(id)
if err != nil {
	return ReadingState{}, fmt.Errorf("load passage %s: %w", id, err)
}
```

## Make Zero Values Safe Or Impossible

If a zero value is valid, support it intentionally. If it is invalid, provide a constructor and test the failure mode.

Do not leave readers guessing whether `Passage{}` or `Question{}` is meaningful.

## Prefer Tables In Tests When Behavior Varies By Case

Go becomes much easier to maintain when behavior differences are encoded as table-driven tests.

```go
func TestPassageFit(t *testing.T) {
	tests := []struct {
		name string
		text string
		want PassageFit
	}{
		{name: "fits", text: "One short sentence.", want: FitsScreen},
		{name: "too long", text: strings.Repeat("word ", 400), want: TooLong},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FitPassage(defaultScreenPolicy(), tt.text)
			if got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}
```

## Keep Packages Boring

Prefer a package layout that helps a new reader find code quickly.

- `opinion` for core opinion types
- `passage` for passage/chunk logic
- `question` for anchors and questions
- `storage` for persistence seams
- `mvp` for end-to-end assembly

Avoid deep trees and avoid packages that exist only to sound architectural.

## Write Code That Makes The Next Step Obvious

Each function should make it easy to see what happens next.

Good Go often reads like:

1. load
2. validate
3. derive
4. persist
5. return

If a reader has to stop and decode the control flow, the code is too dense.
