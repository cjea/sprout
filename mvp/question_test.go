package mvp

import (
	"errors"
	"testing"
	"time"
)

func TestNewAnchorAndQuestion(t *testing.T) {
	opinionID, _ := NewOpinionID("24-777")
	sectionID, _ := NewSectionID("majority")
	passageID, _ := NewPassageID("p-1")
	questionID, _ := NewQuestionID("q-1")
	userID, _ := NewUserID("reader-1")
	span, _ := NewSpan(10, 20, "substantial-evidence review")

	anchor, err := NewAnchor(opinionID, sectionID, passageID, span)
	if err != nil {
		t.Fatalf("new anchor: %v", err)
	}

	question, err := NewQuestion(questionID, userID, anchor, "What does this standard mean?", time.Now(), QuestionStatusOpen)
	if err != nil {
		t.Fatalf("new question: %v", err)
	}
	if question.Status != QuestionStatusOpen {
		t.Fatalf("got status %q, want %q", question.Status, QuestionStatusOpen)
	}
}

func TestParseQuestionStatus(t *testing.T) {
	got, err := ParseQuestionStatus(" Answered ")
	if err != nil {
		t.Fatalf("parse question status: %v", err)
	}
	if got != QuestionStatusAnswered {
		t.Fatalf("got %q, want %q", got, QuestionStatusAnswered)
	}
}

func TestQuestionValidation(t *testing.T) {
	opinionID, _ := NewOpinionID("24-777")
	sectionID, _ := NewSectionID("majority")
	passageID, _ := NewPassageID("p-1")
	questionID, _ := NewQuestionID("q-1")
	userID, _ := NewUserID("reader-1")
	span, _ := NewSpan(10, 20, "quote")
	anchor, _ := NewAnchor(opinionID, sectionID, passageID, span)

	tests := []struct {
		name string
		fn   func() error
		err  error
	}{
		{
			name: "missing question text",
			fn: func() error {
				_, err := NewQuestion(questionID, userID, anchor, "", time.Now(), QuestionStatusOpen)
				return err
			},
			err: ErrEmptyQuestionText,
		},
		{
			name: "invalid status",
			fn: func() error {
				_, err := NewQuestion(questionID, userID, anchor, "question", time.Now(), QuestionStatus("weird"))
				return err
			},
			err: ErrInvalidQuestionState,
		},
		{
			name: "missing anchor passage",
			fn: func() error {
				badAnchor := Anchor{OpinionID: opinionID, SectionID: sectionID, Span: span}
				_, err := NewQuestion(questionID, userID, badAnchor, "question", time.Now(), QuestionStatusOpen)
				return err
			},
			err: ErrAnchorPassageMissing,
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

func TestQuestionSatisfiesStorageRecord(t *testing.T) {
	var record QuestionRecord = Question{}
	if record == nil {
		t.Fatalf("expected question to satisfy QuestionRecord")
	}
}
