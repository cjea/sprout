package mvp

import (
	"errors"
	"strings"
)

var (
	ErrEmptyQuestionText    = errors.New("question text is required")
	ErrInvalidQuestionState = errors.New("question status is invalid")
	ErrAnchorOpinionMissing = errors.New("anchor opinion id is required")
	ErrAnchorSectionMissing = errors.New("anchor section id is required")
	ErrAnchorPassageMissing = errors.New("anchor passage id is required")
)

type Anchor struct {
	OpinionID OpinionID
	SectionID SectionID
	PassageID PassageID
	Span      Span
}

func NewAnchor(opinionID OpinionID, sectionID SectionID, passageID PassageID, span Span) (Anchor, error) {
	anchor := Anchor{
		OpinionID: opinionID,
		SectionID: sectionID,
		PassageID: passageID,
		Span:      span,
	}
	if err := anchor.Validate(); err != nil {
		return Anchor{}, err
	}
	return anchor, nil
}

func (a Anchor) Validate() error {
	switch {
	case strings.TrimSpace(string(a.OpinionID)) == "":
		return ErrAnchorOpinionMissing
	case strings.TrimSpace(string(a.SectionID)) == "":
		return ErrAnchorSectionMissing
	case strings.TrimSpace(string(a.PassageID)) == "":
		return ErrAnchorPassageMissing
	default:
		return a.Span.Validate()
	}
}

type QuestionStatus string

const (
	QuestionStatusOpen     QuestionStatus = "open"
	QuestionStatusAnswered QuestionStatus = "answered"
	QuestionStatusDeferred QuestionStatus = "deferred"
)

func ParseQuestionStatus(value string) (QuestionStatus, error) {
	switch QuestionStatus(strings.TrimSpace(strings.ToLower(value))) {
	case QuestionStatusOpen, QuestionStatusAnswered, QuestionStatusDeferred:
		return QuestionStatus(strings.TrimSpace(strings.ToLower(value))), nil
	default:
		return "", ErrInvalidQuestionState
	}
}

type Question struct {
	QuestionID QuestionID
	UserID     UserID
	Anchor     Anchor
	Text       QuestionText
	AskedAt    Timestamp
	Status     QuestionStatus
}

func NewQuestion(
	questionID QuestionID,
	userID UserID,
	anchor Anchor,
	text QuestionText,
	askedAt Timestamp,
	status QuestionStatus,
) (Question, error) {
	question := Question{
		QuestionID: questionID,
		UserID:     userID,
		Anchor:     anchor,
		Text:       QuestionText(strings.TrimSpace(string(text))),
		AskedAt:    askedAt,
		Status:     status,
	}
	if err := question.Validate(); err != nil {
		return Question{}, err
	}
	return question, nil
}

func (q Question) Validate() error {
	switch {
	case strings.TrimSpace(string(q.QuestionID)) == "":
		return ErrEmptyQuestionID
	case strings.TrimSpace(string(q.UserID)) == "":
		return ErrEmptyUserID
	case strings.TrimSpace(string(q.Text)) == "":
		return ErrEmptyQuestionText
	default:
		if err := q.Anchor.Validate(); err != nil {
			return err
		}
		_, err := ParseQuestionStatus(q.Status.String())
		return err
	}
}

func (q QuestionStatus) String() string {
	return string(q)
}

func (Question) QuestionRecord() {}
