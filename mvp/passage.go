package mvp

import (
	"errors"
	"strings"
)

var (
	ErrInvalidChunkPolicy     = errors.New("chunk policy is invalid")
	ErrInvalidScreenPolicy    = errors.New("screen policy is invalid")
	ErrInvalidSentenceRange   = errors.New("sentence range is invalid")
	ErrInvalidPageRange       = errors.New("page range is invalid")
	ErrEmptyPassageText       = errors.New("passage text is required")
	ErrPassageOpinionMismatch = errors.New("citation must belong to the same passage span context")
)

type ChunkPolicy struct {
	TargetSentences       int
	MaxSentences          int
	PreferSingleSentence  bool
	KeepSectionBoundaries bool
	KeepCitationContext   bool
}

func DefaultChunkPolicy() ChunkPolicy {
	return ChunkPolicy{
		TargetSentences:       1,
		MaxSentences:          3,
		PreferSingleSentence:  true,
		KeepSectionBoundaries: true,
		KeepCitationContext:   true,
	}
}

func (p ChunkPolicy) Validate() error {
	switch {
	case p.TargetSentences <= 0:
		return ErrInvalidChunkPolicy
	case p.MaxSentences <= 0:
		return ErrInvalidChunkPolicy
	case p.TargetSentences > p.MaxSentences:
		return ErrInvalidChunkPolicy
	default:
		return nil
	}
}

type ScreenPolicy struct {
	MaxRenderedLines int
	MaxCharacters    int
	RequireFullFit   bool
}

func DefaultScreenPolicy() ScreenPolicy {
	return ScreenPolicy{
		MaxRenderedLines: 18,
		MaxCharacters:    900,
		RequireFullFit:   true,
	}
}

func (p ScreenPolicy) Validate() error {
	switch {
	case p.MaxRenderedLines <= 0:
		return ErrInvalidScreenPolicy
	case p.MaxCharacters <= 0:
		return ErrInvalidScreenPolicy
	default:
		return nil
	}
}

type PassageFit string

const (
	PassageFitFitsScreen  PassageFit = "fits_screen"
	PassageFitTooLong     PassageFit = "too_long"
	PassageFitNeedsRepair PassageFit = "needs_repair"
)

func (f PassageFit) String() string {
	return string(f)
}

type Passage struct {
	PassageID     PassageID
	OpinionID     OpinionID
	SectionID     SectionID
	SentenceStart SentenceNo
	SentenceEnd   SentenceNo
	PageStart     PageNo
	PageEnd       PageNo
	Text          Text
	Citations     []Citation
	FitsOnScreen  bool
}

func NewPassage(
	passageID PassageID,
	opinionID OpinionID,
	sectionID SectionID,
	sentenceStart, sentenceEnd SentenceNo,
	pageStart, pageEnd PageNo,
	text Text,
	citations []Citation,
	fitsOnScreen bool,
) (Passage, error) {
	passage := Passage{
		PassageID:     passageID,
		OpinionID:     opinionID,
		SectionID:     sectionID,
		SentenceStart: sentenceStart,
		SentenceEnd:   sentenceEnd,
		PageStart:     pageStart,
		PageEnd:       pageEnd,
		Text:          Text(strings.TrimSpace(string(text))),
		Citations:     append([]Citation(nil), citations...),
		FitsOnScreen:  fitsOnScreen,
	}
	if err := passage.Validate(); err != nil {
		return Passage{}, err
	}
	return passage, nil
}

func (p Passage) Validate() error {
	switch {
	case strings.TrimSpace(string(p.PassageID)) == "":
		return ErrEmptyPassageID
	case strings.TrimSpace(string(p.OpinionID)) == "":
		return ErrEmptyOpinionID
	case strings.TrimSpace(string(p.SectionID)) == "":
		return ErrEmptySectionID
	case p.SentenceStart < 0 || p.SentenceEnd < 0 || p.SentenceStart > p.SentenceEnd:
		return ErrInvalidSentenceRange
	case p.PageStart <= 0 || p.PageEnd <= 0 || p.PageStart > p.PageEnd:
		return ErrInvalidPageRange
	case strings.TrimSpace(string(p.Text)) == "":
		return ErrEmptyPassageText
	default:
		for _, citation := range p.Citations {
			if err := citation.Validate(); err != nil {
				return err
			}
		}
		return nil
	}
}

func (Passage) PassageRecord() {}
