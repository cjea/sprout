package mvp

import (
	"errors"
	"testing"
)

func TestChunkAndScreenPolicyDefaults(t *testing.T) {
	chunk := DefaultChunkPolicy()
	if err := chunk.Validate(); err != nil {
		t.Fatalf("default chunk policy: %v", err)
	}
	if chunk.TargetSentences != 1 || chunk.MaxSentences != 3 {
		t.Fatalf("unexpected chunk policy defaults: %+v", chunk)
	}

	screen := DefaultScreenPolicy()
	if err := screen.Validate(); err != nil {
		t.Fatalf("default screen policy: %v", err)
	}
	if !screen.RequireFullFit {
		t.Fatalf("expected default screen policy to require full fit")
	}
}

func TestPolicyValidation(t *testing.T) {
	if !errors.Is(ChunkPolicy{TargetSentences: 4, MaxSentences: 3}.Validate(), ErrInvalidChunkPolicy) {
		t.Fatalf("expected invalid chunk policy")
	}
	if !errors.Is(ScreenPolicy{MaxRenderedLines: 0, MaxCharacters: 1}.Validate(), ErrInvalidScreenPolicy) {
		t.Fatalf("expected invalid screen policy")
	}
}

func TestNewPassage(t *testing.T) {
	opinionID, _ := NewOpinionID("24-777")
	sectionID, _ := NewSectionID("majority")
	passageID, _ := NewPassageID("p-1")
	span, _ := NewSpan(0, 4, "Roe")
	citationID, _ := NewCitationID("c-1")
	citation, _ := NewCitation(citationID, CitationKindCase, "Roe v. Wade", nil, span)

	passage, err := NewPassage(passageID, opinionID, sectionID, 0, 0, 4, 4, "One sentence.", []Citation{citation}, true)
	if err != nil {
		t.Fatalf("new passage: %v", err)
	}
	if len(passage.Citations) != 1 {
		t.Fatalf("got %d citations, want 1", len(passage.Citations))
	}
}

func TestPassageValidation(t *testing.T) {
	opinionID, _ := NewOpinionID("24-777")
	sectionID, _ := NewSectionID("majority")
	passageID, _ := NewPassageID("p-1")

	tests := []struct {
		name string
		fn   func() error
		err  error
	}{
		{
			name: "invalid sentence range",
			fn: func() error {
				_, err := NewPassage(passageID, opinionID, sectionID, 2, 1, 1, 1, "text", nil, true)
				return err
			},
			err: ErrInvalidSentenceRange,
		},
		{
			name: "invalid page range",
			fn: func() error {
				_, err := NewPassage(passageID, opinionID, sectionID, 0, 0, 2, 1, "text", nil, true)
				return err
			},
			err: ErrInvalidPageRange,
		},
		{
			name: "empty text",
			fn: func() error {
				_, err := NewPassage(passageID, opinionID, sectionID, 0, 0, 1, 1, "", nil, true)
				return err
			},
			err: ErrEmptyPassageText,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.fn(), tt.err) {
				t.Fatalf("expected error %v", tt.err)
			}
		})
	}
}

func TestPassageSatisfiesStorageRecord(t *testing.T) {
	var record PassageRecord = Passage{}
	if record == nil {
		t.Fatalf("expected passage to satisfy PassageRecord")
	}
}
