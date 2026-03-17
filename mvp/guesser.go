package mvp

import (
	"context"
	"errors"
	"time"
)

var (
	ErrEmptyContext = errors.New("context is required")
)

type SectionGuesser interface {
	GuessSections(context.Context, ParsedPDF) ([]Section, error)
}

type AnswerGuesser interface {
	GuessAnswer(context.Context, Context, Question) (AnswerDraft, error)
}

type HeuristicSectionGuesser struct {
	Model Model
}

func (g HeuristicSectionGuesser) GuessSections(_ context.Context, parsed ParsedPDF) ([]Section, error) {
	if err := g.Model.Validate(); err != nil {
		return nil, err
	}
	return guessSectionsHeuristic(parsed)
}

type HeuristicAnswerGuesser struct {
	Model Model
	Clock Clock
}

func (g HeuristicAnswerGuesser) GuessAnswer(_ context.Context, context Context, question Question) (AnswerDraft, error) {
	if err := g.Model.Validate(); err != nil {
		return AnswerDraft{}, err
	}
	if err := context.Validate(); err != nil {
		return AnswerDraft{}, err
	}
	if err := question.Validate(); err != nil {
		return AnswerDraft{}, err
	}
	if question.Anchor.Span.Quote == "" {
		return AnswerDraft{}, ErrEmptyContext
	}

	evidence, err := NewEvidence(question.Anchor, question.Anchor.Span.Quote, "selected passage")
	if err != nil {
		return AnswerDraft{}, err
	}
	answer := buildHeuristicAnswer(context, question)
	return NewAnswerDraft(
		question.QuestionID,
		AnswerText(answer),
		[]Evidence{evidence},
		[]string{"This is a heuristic answer grounded in the local source window."},
		g.Clock.Now(),
		g.Model.Name,
	)
}

func defaultHeuristicClock() Clock {
	return FixedClock{Time: time.Now().UTC()}
}
