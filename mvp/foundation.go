package mvp

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrEmptyURL           = errors.New("url is required")
	ErrEmptyOpinionID     = errors.New("opinion id is required")
	ErrEmptyUserID        = errors.New("user id is required")
	ErrEmptySectionID     = errors.New("section id is required")
	ErrEmptyPassageID     = errors.New("passage id is required")
	ErrEmptyQuestionID    = errors.New("question id is required")
	ErrEmptyCitationID    = errors.New("citation id is required")
	ErrEmptyModelName     = errors.New("model name is required")
	ErrInvalidModelTokens = errors.New("max context tokens must be positive")
)

type URL string

type UserID string
type OpinionID string
type SectionID string
type PassageID string
type QuestionID string
type CitationID string
type JusticeName string
type QuestionText string
type AnswerText string
type Text string
type PageNo int
type Offset int
type SentenceNo int

type Timestamp = time.Time
type PDFBytes = []byte

func NewOpinionID(value string) (OpinionID, error) {
	return newID[OpinionID](value, ErrEmptyOpinionID)
}

func NewUserID(value string) (UserID, error) {
	return newID[UserID](value, ErrEmptyUserID)
}

func NewSectionID(value string) (SectionID, error) {
	return newID[SectionID](value, ErrEmptySectionID)
}

func NewPassageID(value string) (PassageID, error) {
	return newID[PassageID](value, ErrEmptyPassageID)
}

func NewQuestionID(value string) (QuestionID, error) {
	return newID[QuestionID](value, ErrEmptyQuestionID)
}

func NewCitationID(value string) (CitationID, error) {
	return newID[CitationID](value, ErrEmptyCitationID)
}

func newID[T ~string](value string, errEmpty error) (T, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		var zero T
		return zero, errEmpty
	}
	return T(trimmed), nil
}

type UserInput struct {
	URL URL
}

func NewUserInput(url URL) (UserInput, error) {
	input := UserInput{URL: url}
	if err := input.Validate(); err != nil {
		return UserInput{}, err
	}
	return input, nil
}

func (u UserInput) Validate() error {
	if strings.TrimSpace(string(u.URL)) == "" {
		return ErrEmptyURL
	}
	return nil
}

type Clock interface {
	Now() Timestamp
}

type FixedClock struct {
	Time Timestamp
}

func (c FixedClock) Now() Timestamp {
	return c.Time
}

type SystemClock struct{}

func (SystemClock) Now() Timestamp {
	return time.Now().UTC()
}

type Model struct {
	Name             string
	MaxContextTokens int
}

func NewModel(name string, maxContextTokens int) (Model, error) {
	model := Model{
		Name:             strings.TrimSpace(name),
		MaxContextTokens: maxContextTokens,
	}
	if err := model.Validate(); err != nil {
		return Model{}, err
	}
	return model, nil
}

func (m Model) Validate() error {
	if m.Name == "" {
		return ErrEmptyModelName
	}
	if m.MaxContextTokens <= 0 {
		return ErrInvalidModelTokens
	}
	return nil
}

type RawPDFRecord interface {
	RawPDFRecord()
}

type OpinionRecord interface {
	OpinionRecord()
}

type PassageRecord interface {
	PassageRecord()
}

type ProgressRecord interface {
	ProgressRecord()
}

type QuestionRecord interface {
	QuestionRecord()
}

type Storage interface {
	SaveRawPDF(RawPDFRecord) (RawPDFRecord, error)
	SaveOpinion(OpinionRecord) (OpinionRecord, error)
	SavePassages([]PassageRecord) ([]PassageRecord, error)
	SaveProgress(ProgressRecord) (ProgressRecord, error)
	SaveQuestionRecord(QuestionRecord) (QuestionRecord, error)

	LoadRawPDF(OpinionID) (RawPDFRecord, error)
	LoadOpinion(OpinionID) (OpinionRecord, error)
	LoadPassage(PassageID) (PassageRecord, error)
	LoadProgress(UserID, OpinionID) (ProgressRecord, error)
	LoadQuestions(UserID, OpinionID) ([]QuestionRecord, error)
}

type PassageLister interface {
	ListPassages(OpinionID) ([]PassageRecord, error)
}
