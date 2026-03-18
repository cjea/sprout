package mvp

import (
	"context"
	"errors"
	"testing"
)

type stubSentenceRepairGuesser struct {
	guesses []SentenceRepairGuess
	err     error
	called  bool
}

func (stub *stubSentenceRepairGuesser) GuessSentenceRepairs(_ context.Context, _ Model, _ SentenceRepairRequest) ([]SentenceRepairGuess, error) {
	stub.called = true
	if stub.err != nil {
		return nil, stub.err
	}
	return stub.guesses, nil
}

func TestMaybeGuessSentenceRepairsDisabledModeKeepsHeuristicSentences(t *testing.T) {
	model, err := NewModel("heuristic-v1", 8192)
	if err != nil {
		t.Fatalf("new model: %v", err)
	}

	guesser := &stubSentenceRepairGuesser{
		guesses: []SentenceRepairGuess{
			{
				LeftIndex:  0,
				RightIndex: 2,
				MergedText: "Held: Courts should review the Board's determination. Pp. 5-13.",
				Confidence: 0.95,
			},
		},
	}
	sentences := []string{
		"Held: Courts should review the Board's determination.",
		"Pp.",
		"5-13.",
	}

	got := MaybeGuessSentenceRepairs(context.Background(), model, DefaultSentenceRepairConfig(), guesser, sentences)
	if guesser.called {
		t.Fatalf("expected disabled mode to skip the guesser")
	}
	if len(got) != len(sentences) {
		t.Fatalf("got %d sentences want %d", len(got), len(sentences))
	}
	for index := range sentences {
		if got[index] != sentences[index] {
			t.Fatalf("sentence %d got %q want %q", index, got[index], sentences[index])
		}
	}
}

func TestMaybeGuessSentenceRepairsFallsBackOnMalformedGuess(t *testing.T) {
	model, err := NewModel("gpt-4.1", 8192)
	if err != nil {
		t.Fatalf("new model: %v", err)
	}

	guesser := &stubSentenceRepairGuesser{
		guesses: []SentenceRepairGuess{
			{
				LeftIndex:  0,
				RightIndex: 1,
				MergedText: "Held: Courts should review the Board's determination. Pp.",
				Confidence: 0.95,
			},
		},
	}
	config := SentenceRepairConfig{Enabled: true, MinConfidence: 0.8}
	sentences := []string{
		"Held: Courts should review the Board's determination.",
		"Pp.",
		"5-13.",
	}

	got := MaybeGuessSentenceRepairs(context.Background(), model, config, guesser, sentences)
	if !guesser.called {
		t.Fatalf("expected enabled mode to call the guesser")
	}
	if len(got) != len(sentences) {
		t.Fatalf("got %d sentences want %d", len(got), len(sentences))
	}
	for index := range sentences {
		if got[index] != sentences[index] {
			t.Fatalf("sentence %d got %q want %q", index, got[index], sentences[index])
		}
	}
}

func TestMaybeGuessSentenceRepairsFallsBackOnMissingBoundaryCoverage(t *testing.T) {
	model, err := NewModel("gpt-4.1", 8192)
	if err != nil {
		t.Fatalf("new model: %v", err)
	}

	guesser := &stubSentenceRepairGuesser{
		guesses: []SentenceRepairGuess{
			{
				LeftIndex:  0,
				RightIndex: 2,
				MergedText: "Held: Courts should review the Board's determination. Pp. 5-13.",
				Confidence: 0.95,
			},
		},
	}
	config := SentenceRepairConfig{Enabled: true, MinConfidence: 0.8}
	sentences := []string{
		"The court rejected that argument.",
		"Id.",
		", at 484.",
		"Under the INA, the U. S. Government may grant asylum to a noncitizen if it determines that he is a refugee.",
		"8 U. S. C. §1158(b)(1)(A).",
	}

	got := MaybeGuessSentenceRepairs(context.Background(), model, config, guesser, sentences)
	if len(got) != len(sentences) {
		t.Fatalf("got %d sentences want %d", len(got), len(sentences))
	}
	for index := range sentences {
		if got[index] != sentences[index] {
			t.Fatalf("sentence %d got %q want %q", index, got[index], sentences[index])
		}
	}
}

func TestMaybeGuessSentenceRepairsFallsBackOnLowConfidenceGuess(t *testing.T) {
	model, err := NewModel("gpt-4.1", 8192)
	if err != nil {
		t.Fatalf("new model: %v", err)
	}

	guesser := &stubSentenceRepairGuesser{
		guesses: []SentenceRepairGuess{
			{
				LeftIndex:  0,
				RightIndex: 2,
				MergedText: "Held: Courts should review the Board's determination. Pp. 5-13.",
				Confidence: 0.2,
			},
		},
	}
	config := SentenceRepairConfig{Enabled: true, MinConfidence: 0.8}
	sentences := []string{
		"Held: Courts should review the Board's determination.",
		"Pp.",
		"5-13.",
	}

	got := MaybeGuessSentenceRepairs(context.Background(), model, config, guesser, sentences)
	if len(got) != len(sentences) {
		t.Fatalf("got %d sentences want %d", len(got), len(sentences))
	}
	for index := range sentences {
		if got[index] != sentences[index] {
			t.Fatalf("sentence %d got %q want %q", index, got[index], sentences[index])
		}
	}
}

func TestMaybeGuessSentenceRepairsFallsBackOnGuesserError(t *testing.T) {
	model, err := NewModel("gpt-4.1", 8192)
	if err != nil {
		t.Fatalf("new model: %v", err)
	}

	guesser := &stubSentenceRepairGuesser{
		err: errors.New("boom"),
	}
	config := SentenceRepairConfig{Enabled: true, MinConfidence: 0.8}
	sentences := []string{
		"Held: Courts should review the Board's determination.",
		"Pp.",
		"5-13.",
	}

	got := MaybeGuessSentenceRepairs(context.Background(), model, config, guesser, sentences)
	if len(got) != len(sentences) {
		t.Fatalf("got %d sentences want %d", len(got), len(sentences))
	}
	for index := range sentences {
		if got[index] != sentences[index] {
			t.Fatalf("sentence %d got %q want %q", index, got[index], sentences[index])
		}
	}
}

func TestMaybeGuessSentenceRepairsAppliesValidatedMergeDecisions(t *testing.T) {
	model, err := NewModel("gpt-4.1", 8192)
	if err != nil {
		t.Fatalf("new model: %v", err)
	}

	guesser := &stubSentenceRepairGuesser{
		guesses: []SentenceRepairGuess{
			{
				LeftIndex:  0,
				RightIndex: 2,
				MergedText: "Held: Courts should review the Board's determination. Pp. 5-13.",
				Confidence: 0.95,
			},
			{
				LeftIndex:  3,
				RightIndex: 4,
				MergedText: "Under the INA, the U. S. Government may grant asylum to a noncitizen if it determines that he is a refugee. 8 U. S. C. §1158(b)(1)(A).",
				Confidence: 0.95,
			},
		},
	}
	config := SentenceRepairConfig{Enabled: true, MinConfidence: 0.8}
	sentences := []string{
		"Held: Courts should review the Board's determination.",
		"Pp.",
		"5-13.",
		"Under the INA, the U. S. Government may grant asylum to a noncitizen if it determines that he is a refugee.",
		"8 U. S. C. §1158(b)(1)(A).",
	}

	got := MaybeGuessSentenceRepairs(context.Background(), model, config, guesser, sentences)
	want := []string{
		"Held: Courts should review the Board's determination. Pp. 5-13.",
		"Under the INA, the U. S. Government may grant asylum to a noncitizen if it determines that he is a refugee. 8 U. S. C. §1158(b)(1)(A).",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d sentences want %d: %#v", len(got), len(want), got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("sentence %d got %q want %q", index, got[index], want[index])
		}
	}
}
