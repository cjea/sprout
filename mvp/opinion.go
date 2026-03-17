package mvp

import (
	"errors"
	"strings"
)

var (
	ErrEmptyCaseName          = errors.New("case name is required")
	ErrEmptyDocketNumber      = errors.New("docket number is required")
	ErrEmptyDecisionDate      = errors.New("decision date is required")
	ErrEmptyTermLabel         = errors.New("term label is required")
	ErrEmptySectionTitle      = errors.New("section title is required")
	ErrEmptySectionText       = errors.New("section text is required")
	ErrInvalidSectionPages    = errors.New("section pages are invalid")
	ErrSectionsOutOfOrder     = errors.New("sections must be in ascending page order")
	ErrEmptyOpinionText       = errors.New("opinion full text is required")
	ErrEmptySectionKind       = errors.New("section kind is required")
	ErrUnknownSectionKind     = errors.New("unknown section kind")
	ErrSectionOpinionMismatch = errors.New("section opinion id must match opinion id")
)

type SectionKind string

const (
	SectionKindSyllabus    SectionKind = "syllabus"
	SectionKindMajority    SectionKind = "majority"
	SectionKindConcurrence SectionKind = "concurrence"
	SectionKindDissent     SectionKind = "dissent"
	SectionKindAppendix    SectionKind = "appendix"
	SectionKindUnknown     SectionKind = "unknown"
)

func ParseSectionKind(value string) (SectionKind, error) {
	switch SectionKind(strings.TrimSpace(strings.ToLower(value))) {
	case SectionKindSyllabus,
		SectionKindMajority,
		SectionKindConcurrence,
		SectionKindDissent,
		SectionKindAppendix,
		SectionKindUnknown:
		return SectionKind(strings.TrimSpace(strings.ToLower(value))), nil
	case "":
		return "", ErrEmptySectionKind
	default:
		return "", ErrUnknownSectionKind
	}
}

func (k SectionKind) String() string {
	return string(k)
}

type Meta struct {
	CaseName      string
	DocketNumber  string
	DecidedOn     string
	TermLabel     string
	PrimaryAuthor *JusticeName
}

func NewMeta(caseName, docketNumber, decidedOn, termLabel string, primaryAuthor *JusticeName) (Meta, error) {
	meta := Meta{
		CaseName:      strings.TrimSpace(caseName),
		DocketNumber:  strings.TrimSpace(docketNumber),
		DecidedOn:     strings.TrimSpace(decidedOn),
		TermLabel:     strings.TrimSpace(termLabel),
		PrimaryAuthor: primaryAuthor,
	}
	if err := meta.Validate(); err != nil {
		return Meta{}, err
	}
	return meta, nil
}

func (m Meta) Validate() error {
	switch {
	case m.CaseName == "":
		return ErrEmptyCaseName
	case m.DocketNumber == "":
		return ErrEmptyDocketNumber
	case m.DecidedOn == "":
		return ErrEmptyDecisionDate
	case m.TermLabel == "":
		return ErrEmptyTermLabel
	default:
		return nil
	}
}

type Section struct {
	OpinionID OpinionID
	SectionID SectionID
	Kind      SectionKind
	Title     string
	Author    *JusticeName
	StartPage PageNo
	EndPage   PageNo
	Text      Text
}

func NewSection(
	opinionID OpinionID,
	sectionID SectionID,
	kind SectionKind,
	title string,
	author *JusticeName,
	startPage PageNo,
	endPage PageNo,
	text Text,
) (Section, error) {
	section := Section{
		OpinionID: opinionID,
		SectionID: sectionID,
		Kind:      kind,
		Title:     strings.TrimSpace(title),
		Author:    author,
		StartPage: startPage,
		EndPage:   endPage,
		Text:      Text(strings.TrimSpace(string(text))),
	}
	if err := section.Validate(); err != nil {
		return Section{}, err
	}
	return section, nil
}

func (s Section) Validate() error {
	switch {
	case strings.TrimSpace(string(s.OpinionID)) == "":
		return ErrEmptyOpinionID
	case strings.TrimSpace(string(s.SectionID)) == "":
		return ErrEmptySectionID
	case strings.TrimSpace(s.Kind.String()) == "":
		return ErrEmptySectionKind
	case s.Title == "":
		return ErrEmptySectionTitle
	case strings.TrimSpace(string(s.Text)) == "":
		return ErrEmptySectionText
	case s.StartPage <= 0 || s.EndPage <= 0 || s.StartPage > s.EndPage:
		return ErrInvalidSectionPages
	default:
		return nil
	}
}

type Opinion struct {
	OpinionID OpinionID
	Meta      Meta
	Sections  []Section
	FullText  Text
}

func NewOpinion(opinionID OpinionID, meta Meta, sections []Section, fullText Text) (Opinion, error) {
	opinion := Opinion{
		OpinionID: opinionID,
		Meta:      meta,
		Sections:  append([]Section(nil), sections...),
		FullText:  Text(strings.TrimSpace(string(fullText))),
	}
	if err := opinion.Validate(); err != nil {
		return Opinion{}, err
	}
	return opinion, nil
}

func (o Opinion) Validate() error {
	if strings.TrimSpace(string(o.OpinionID)) == "" {
		return ErrEmptyOpinionID
	}
	if err := o.Meta.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(string(o.FullText)) == "" {
		return ErrEmptyOpinionText
	}

	lastStartPage := PageNo(0)
	for index, section := range o.Sections {
		if err := section.Validate(); err != nil {
			return err
		}
		if section.OpinionID != o.OpinionID {
			return ErrSectionOpinionMismatch
		}
		if index > 0 && section.StartPage < lastStartPage {
			return ErrSectionsOutOfOrder
		}
		lastStartPage = section.StartPage
	}

	return nil
}

func (Opinion) OpinionRecord() {}
