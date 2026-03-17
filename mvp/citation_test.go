package mvp

import (
	"errors"
	"testing"
)

func TestParseCitationKind(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  CitationKind
		err   error
	}{
		{name: "case", input: "case", want: CitationKindCase},
		{name: "statute", input: " Statute ", want: CitationKindStatute},
		{name: "unknown", input: "treatise", err: ErrUnknownCitationKind},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCitationKind(tt.input)
			if !errors.Is(err, tt.err) {
				t.Fatalf("expected error %v, got %v", tt.err, err)
			}
			if tt.err == nil && got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewSpanAndCitation(t *testing.T) {
	span, err := NewSpan(3, 10, "5 U. S. C. §706")
	if err != nil {
		t.Fatalf("new span: %v", err)
	}
	citationID, _ := NewCitationID("c-1")
	normalized := "5 U.S.C. § 706"

	citation, err := NewCitation(citationID, CitationKindStatute, " 5 U. S. C. §706 ", &normalized, span)
	if err != nil {
		t.Fatalf("new citation: %v", err)
	}
	if citation.RawText != "5 U. S. C. §706" {
		t.Fatalf("got raw text %q", citation.RawText)
	}
	if citation.Normalized == nil || *citation.Normalized != normalized {
		t.Fatalf("expected normalized citation text")
	}
}

func TestCitationValidation(t *testing.T) {
	span, _ := NewSpan(1, 2, "cite")
	citationID, _ := NewCitationID("c-1")
	blank := "   "

	tests := []struct {
		name string
		fn   func() error
		err  error
	}{
		{
			name: "invalid span",
			fn: func() error {
				_, err := NewSpan(5, 1, "cite")
				return err
			},
			err: ErrInvalidSpanOffsets,
		},
		{
			name: "empty raw text",
			fn: func() error {
				_, err := NewCitation(citationID, CitationKindCase, "", nil, span)
				return err
			},
			err: ErrEmptyRawCitation,
		},
		{
			name: "empty normalized text",
			fn: func() error {
				_, err := NewCitation(citationID, CitationKindCase, "Roe v. Wade", &blank, span)
				return err
			},
			err: ErrEmptyNormalizedText,
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
