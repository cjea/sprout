package mvp

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrQueueHasNoNextPassage = errors.New("queue has no next passage")
	ErrQuestionOutsideText   = errors.New("selected span is outside passage text")
)

func BuildQueue(userID UserID, opinion Opinion, passages []Passage) (Queue, error) {
	pending := make([]PassageID, 0, len(passages))
	for _, passage := range passages {
		pending = append(pending, passage.PassageID)
	}
	return NewQueue(userID, opinion.OpinionID, pending)
}

func NextPassage(queue Queue) (*PassageID, error) {
	if err := queue.Validate(); err != nil {
		return nil, err
	}
	if len(queue.Pending) == 0 {
		return nil, ErrQueueHasNoNextPassage
	}
	next := queue.Pending[0]
	return &next, nil
}

func OpenPassage(userID UserID, passageID PassageID, storage Storage) (ReadingState, error) {
	passage, err := loadPassage(storage, passageID)
	if err != nil {
		return ReadingState{}, err
	}
	opinion, err := loadOpinion(storage, passage.OpinionID)
	if err != nil {
		return ReadingState{}, err
	}

	progress, err := loadProgress(storage, userID, opinion.OpinionID)
	if errors.Is(err, ErrNotFound) {
		progress, err = NewProgress(userID, opinion.OpinionID, &passageID, nil, nil, time.Time{})
	}
	if err != nil {
		return ReadingState{}, err
	}
	progress.CurrentPassage = &passageID
	if _, err := saveProgress(storage, progress); err != nil {
		return ReadingState{}, err
	}

	trail, err := NewTrail(passageID, passageID, progress.OpenQuestionIDs)
	if err != nil {
		return ReadingState{}, err
	}
	return NewReadingState(opinion, passage, passage.Citations, progress, trail)
}

func CompletePassage(userID UserID, passageID PassageID, storage Storage, clock Clock) (Progress, error) {
	passage, err := loadPassage(storage, passageID)
	if err != nil {
		return Progress{}, err
	}
	progress, err := loadProgress(storage, userID, passage.OpinionID)
	if err != nil {
		return Progress{}, err
	}

	progress.CurrentPassage = &passageID
	if !containsPassageID(progress.CompletedPassages, passageID) {
		progress.CompletedPassages = append(progress.CompletedPassages, passageID)
	}
	progress.UpdatedAt = clock.Now()

	return saveProgress(storage, progress)
}

func ResumeReading(userID UserID, opinionID OpinionID, storage Storage) (ReadingState, error) {
	progress, err := loadProgress(storage, userID, opinionID)
	if err != nil {
		return ReadingState{}, err
	}
	if progress.CurrentPassage == nil {
		return ReadingState{}, ErrProgressPassage
	}
	return OpenPassage(userID, *progress.CurrentPassage, storage)
}

func SelectSpan(passage Passage, start, end Offset) (Span, error) {
	text := string(passage.Text)
	if start < 0 || end < 0 || int(end) > len(text) || start > end {
		return Span{}, ErrQuestionOutsideText
	}
	return NewSpan(start, end, Text(text[start:end]))
}

func AnchorSpan(opinionID OpinionID, sectionID SectionID, passageID PassageID, span Span) (Anchor, error) {
	return NewAnchor(opinionID, sectionID, passageID, span)
}

func AskQuestion(userID UserID, anchor Anchor, text QuestionText, clock Clock) (Question, error) {
	questionID, err := NewQuestionID(fmt.Sprintf("%s-%d", anchor.PassageID, clock.Now().UnixNano()))
	if err != nil {
		return Question{}, err
	}
	return NewQuestion(questionID, userID, anchor, text, clock.Now(), QuestionStatusOpen)
}

func SaveQuestion(storage Storage, question Question) (Question, error) {
	return saveQuestion(storage, question)
}

func GatherContext(opinion Opinion, passage Passage, anchor Anchor, openQuestions []Question) (Context, error) {
	return NewContext(opinion, passage, anchor, openQuestions, passage.Citations)
}

func SaveAnswer(storage Storage, answer AnswerDraft) (AnswerDraft, error) {
	return saveAnswer(storage, answer)
}

func containsPassageID(ids []PassageID, candidate PassageID) bool {
	for _, id := range ids {
		if id == candidate {
			return true
		}
	}
	return false
}

func buildHeuristicAnswer(context Context, question Question) string {
	citations := make([]string, 0, len(context.NearbyCitations))
	for _, citation := range context.NearbyCitations {
		citations = append(citations, citation.RawText)
	}
	citationText := "No nearby citations were extracted."
	if len(citations) > 0 {
		citationText = "Nearby citations: " + strings.Join(citations, "; ") + "."
	}

	return strings.TrimSpace(fmt.Sprintf(
		"Question: %q. The selected passage says %q. %s Review the cited authority and the surrounding opinion text before treating this as final.",
		question.Text,
		question.Anchor.Span.Quote,
		citationText,
	))
}
