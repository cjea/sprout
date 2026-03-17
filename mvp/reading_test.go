package mvp

import (
	"errors"
	"testing"
	"time"
)

func TestNewQueueProgressTrailAndReadingState(t *testing.T) {
	opinionID, _ := NewOpinionID("24-777")
	sectionID, _ := NewSectionID("majority")
	passageID, _ := NewPassageID("p-1")
	userID, _ := NewUserID("reader-1")
	questionID, _ := NewQuestionID("q-1")

	meta, _ := NewMeta("Case", "24-777", "2026-03-16", "2025", nil)
	section, _ := NewSection(opinionID, sectionID, SectionKindMajority, "Majority", nil, 1, 1, "Opinion text")
	opinion, _ := NewOpinion(opinionID, meta, []Section{section}, "Opinion text")
	passage, _ := NewPassage(passageID, opinionID, sectionID, 0, 0, 1, 1, "Opinion text", nil, true)

	queue, err := NewQueue(userID, opinionID, []PassageID{passageID})
	if err != nil {
		t.Fatalf("new queue: %v", err)
	}
	if len(queue.Pending) != 1 {
		t.Fatalf("got %d pending passages, want 1", len(queue.Pending))
	}

	progress, err := NewProgress(userID, opinionID, &passageID, []PassageID{passageID}, []QuestionID{questionID}, time.Now())
	if err != nil {
		t.Fatalf("new progress: %v", err)
	}

	trail, err := NewTrail(passageID, passageID, []QuestionID{questionID})
	if err != nil {
		t.Fatalf("new trail: %v", err)
	}

	state, err := NewReadingState(opinion, passage, nil, progress, trail)
	if err != nil {
		t.Fatalf("new reading state: %v", err)
	}
	if state.Passage.PassageID != passageID {
		t.Fatalf("got passage id %q, want %q", state.Passage.PassageID, passageID)
	}
}

func TestReadingValidation(t *testing.T) {
	userID, _ := NewUserID("reader-1")
	opinionID, _ := NewOpinionID("24-777")
	passageID, _ := NewPassageID("p-1")

	tests := []struct {
		name string
		fn   func() error
		err  error
	}{
		{
			name: "empty queue",
			fn: func() error {
				_, err := NewQueue(userID, opinionID, nil)
				return err
			},
			err: ErrEmptyQueue,
		},
		{
			name: "duplicate queue passage",
			fn: func() error {
				_, err := NewQueue(userID, opinionID, []PassageID{passageID, passageID})
				return err
			},
			err: ErrDuplicatePassage,
		},
		{
			name: "invalid progress passage",
			fn: func() error {
				blank := PassageID("")
				_, err := NewProgress(userID, opinionID, &blank, nil, nil, time.Now())
				return err
			},
			err: ErrProgressPassage,
		},
		{
			name: "missing trail origin",
			fn: func() error {
				_, err := NewTrail("", passageID, nil)
				return err
			},
			err: ErrTrailOrigin,
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

func TestProgressSatisfiesStorageRecord(t *testing.T) {
	var record ProgressRecord = Progress{}
	if record == nil {
		t.Fatalf("expected progress to satisfy ProgressRecord")
	}
}
