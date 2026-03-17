package mvp

import (
	"errors"
	"strings"
)

var (
	ErrEmptyRawCitation    = errors.New("raw citation text is required")
	ErrEmptyNormalizedText = errors.New("normalized citation text is required when present")
	ErrEmptySpanQuote      = errors.New("span quote is required")
	ErrInvalidSpanOffsets  = errors.New("span offsets are invalid")
	ErrUnknownCitationKind = errors.New("unknown citation kind")
)

type CitationKind string

const (
	CitationKindCase         CitationKind = "case"
	CitationKindStatute      CitationKind = "statute"
	CitationKindConstitution CitationKind = "constitution"
	CitationKindInternal     CitationKind = "internal"
	CitationKindUnknown      CitationKind = "unknown"
)

func ParseCitationKind(value string) (CitationKind, error) {
	switch CitationKind(strings.TrimSpace(strings.ToLower(value))) {
	case CitationKindCase,
		CitationKindStatute,
		CitationKindConstitution,
		CitationKindInternal,
		CitationKindUnknown:
		return CitationKind(strings.TrimSpace(strings.ToLower(value))), nil
	default:
		return "", ErrUnknownCitationKind
	}
}

func (k CitationKind) String() string {
	return string(k)
}

type Span struct {
	StartOffset Offset
	EndOffset   Offset
	Quote       Text
}

func NewSpan(startOffset, endOffset Offset, quote Text) (Span, error) {
	span := Span{
		StartOffset: startOffset,
		EndOffset:   endOffset,
		Quote:       Text(strings.TrimSpace(string(quote))),
	}
	if err := span.Validate(); err != nil {
		return Span{}, err
	}
	return span, nil
}

func (s Span) Validate() error {
	switch {
	case s.StartOffset < 0 || s.EndOffset < 0 || s.StartOffset > s.EndOffset:
		return ErrInvalidSpanOffsets
	case strings.TrimSpace(string(s.Quote)) == "":
		return ErrEmptySpanQuote
	default:
		return nil
	}
}

type Citation struct {
	CitationID CitationID
	Kind       CitationKind
	RawText    string
	Normalized *string
	Span       Span
}

func NewCitation(citationID CitationID, kind CitationKind, rawText string, normalized *string, span Span) (Citation, error) {
	citation := Citation{
		CitationID: citationID,
		Kind:       kind,
		RawText:    strings.TrimSpace(rawText),
		Normalized: trimStringPointer(normalized),
		Span:       span,
	}
	if err := citation.Validate(); err != nil {
		return Citation{}, err
	}
	return citation, nil
}

func (c Citation) Validate() error {
	switch {
	case strings.TrimSpace(string(c.CitationID)) == "":
		return ErrEmptyCitationID
	case strings.TrimSpace(c.Kind.String()) == "":
		return ErrUnknownCitationKind
	case c.RawText == "":
		return ErrEmptyRawCitation
	case c.Normalized != nil && *c.Normalized == "":
		return ErrEmptyNormalizedText
	default:
		return c.Span.Validate()
	}
}

func trimStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
}
