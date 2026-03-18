package mvp

import (
	"errors"
	"strings"
)

var (
	ErrEmptyEvidenceLabel = errors.New("evidence label is required")
	ErrEmptyAnswerText    = errors.New("answer text is required")
)

type Context struct {
	Opinion         Opinion
	ActivePassage   Passage
	Anchor          Anchor
	OpenQuestions   []Question
	NearbyCitations []Citation
}

func NewContext(opinion Opinion, activePassage Passage, anchor Anchor, openQuestions []Question, nearbyCitations []Citation) (Context, error) {
	context := Context{
		Opinion:         opinion,
		ActivePassage:   activePassage,
		Anchor:          anchor,
		OpenQuestions:   append([]Question(nil), openQuestions...),
		NearbyCitations: append([]Citation(nil), nearbyCitations...),
	}
	if err := context.Validate(); err != nil {
		return Context{}, err
	}
	return context, nil
}

func (c Context) Validate() error {
	if err := c.Opinion.Validate(); err != nil {
		return err
	}
	if err := c.ActivePassage.Validate(); err != nil {
		return err
	}
	if err := c.Anchor.Validate(); err != nil {
		return err
	}
	for _, question := range c.OpenQuestions {
		if err := question.Validate(); err != nil {
			return err
		}
	}
	for _, citation := range c.NearbyCitations {
		if err := citation.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type Evidence struct {
	Anchor Anchor
	Quote  Text
	Label  string
}

func NewEvidence(anchor Anchor, quote Text, label string) (Evidence, error) {
	evidence := Evidence{
		Anchor: anchor,
		Quote:  Text(strings.TrimSpace(string(quote))),
		Label:  strings.TrimSpace(label),
	}
	if err := evidence.Validate(); err != nil {
		return Evidence{}, err
	}
	return evidence, nil
}

func (e Evidence) Validate() error {
	switch {
	case strings.TrimSpace(e.Label) == "":
		return ErrEmptyEvidenceLabel
	case strings.TrimSpace(string(e.Quote)) == "":
		return ErrEmptySpanQuote
	default:
		return e.Anchor.Validate()
	}
}

type AnswerDraft struct {
	QuestionID  QuestionID
	Answer      AnswerText
	Evidence    []Evidence
	Caveats     []string
	GeneratedAt Timestamp
	ModelName   string
}

func NewAnswerDraft(questionID QuestionID, answer AnswerText, evidence []Evidence, caveats []string, generatedAt Timestamp, modelName string) (AnswerDraft, error) {
	draft := AnswerDraft{
		QuestionID:  questionID,
		Answer:      AnswerText(strings.TrimSpace(string(answer))),
		Evidence:    append([]Evidence(nil), evidence...),
		Caveats:     append([]string(nil), caveats...),
		GeneratedAt: generatedAt,
		ModelName:   strings.TrimSpace(modelName),
	}
	if err := draft.Validate(); err != nil {
		return AnswerDraft{}, err
	}
	return draft, nil
}

func (d AnswerDraft) Validate() error {
	switch {
	case strings.TrimSpace(string(d.QuestionID)) == "":
		return ErrEmptyQuestionID
	case strings.TrimSpace(string(d.Answer)) == "":
		return ErrEmptyAnswerText
	case strings.TrimSpace(d.ModelName) == "":
		return ErrEmptyModelName
	default:
		for _, evidence := range d.Evidence {
			if err := evidence.Validate(); err != nil {
				return err
			}
		}
		return nil
	}
}

func (AnswerDraft) AnswerRecord() {}
