package mvp

import (
	"errors"
	"testing"
	"time"
)

func TestNewContextAndAnswerDraft(t *testing.T) {
	opinionID, _ := NewOpinionID("24-777")
	sectionID, _ := NewSectionID("majority")
	passageID, _ := NewPassageID("p-1")
	questionID, _ := NewQuestionID("q-1")
	userID, _ := NewUserID("reader-1")

	meta, _ := NewMeta("Case", "24-777", "2026-03-16", "2025", nil)
	section, _ := NewSection(opinionID, sectionID, SectionKindMajority, "Majority", nil, 1, 1, "Opinion text")
	opinion, _ := NewOpinion(opinionID, meta, []Section{section}, "Opinion text")
	span, _ := NewSpan(0, 4, "text")
	citationID, _ := NewCitationID("c-1")
	citation, _ := NewCitation(citationID, CitationKindCase, "Roe v. Wade", nil, span)
	passage, _ := NewPassage(passageID, opinionID, sectionID, 0, 0, 1, 1, "Opinion text", []Citation{citation}, true)
	anchor, _ := NewAnchor(opinionID, sectionID, passageID, span)
	question, _ := NewQuestion(questionID, userID, anchor, "What does this mean?", time.Now(), QuestionStatusOpen)

	context, err := NewContext(opinion, passage, anchor, []Question{question}, []Citation{citation})
	if err != nil {
		t.Fatalf("new context: %v", err)
	}
	if len(context.OpenQuestions) != 1 {
		t.Fatalf("got %d questions, want 1", len(context.OpenQuestions))
	}

	evidence, err := NewEvidence(anchor, "Opinion text", "source")
	if err != nil {
		t.Fatalf("new evidence: %v", err)
	}
	draft, err := NewAnswerDraft(questionID, "It explains the standard.", []Evidence{evidence}, []string{"draft"}, time.Now(), "gpt-5")
	if err != nil {
		t.Fatalf("new answer draft: %v", err)
	}
	if draft.ModelName != "gpt-5" {
		t.Fatalf("got model name %q", draft.ModelName)
	}
}

func TestAnswerValidation(t *testing.T) {
	opinionID, _ := NewOpinionID("24-777")
	sectionID, _ := NewSectionID("majority")
	passageID, _ := NewPassageID("p-1")
	questionID, _ := NewQuestionID("q-1")
	span, _ := NewSpan(0, 4, "text")
	anchor, _ := NewAnchor(opinionID, sectionID, passageID, span)

	tests := []struct {
		name string
		fn   func() error
		err  error
	}{
		{
			name: "missing evidence label",
			fn: func() error {
				_, err := NewEvidence(anchor, "quote", "")
				return err
			},
			err: ErrEmptyEvidenceLabel,
		},
		{
			name: "missing answer text",
			fn: func() error {
				_, err := NewAnswerDraft(questionID, "", nil, nil, time.Now(), "gpt-5")
				return err
			},
			err: ErrEmptyAnswerText,
		},
		{
			name: "missing model name",
			fn: func() error {
				_, err := NewAnswerDraft(questionID, "answer", nil, nil, time.Now(), "")
				return err
			},
			err: ErrEmptyModelName,
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
