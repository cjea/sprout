package mvp

import (
	"context"
	"testing"
)

type cannedSentenceRepairGuesser struct {
	guesses map[string][]SentenceRepairGuess
}

func (guesser cannedSentenceRepairGuesser) GuessSentenceRepairs(_ context.Context, _ Model, request SentenceRepairRequest) ([]SentenceRepairGuess, error) {
	return guesser.guesses[sentenceRepairEvalKey(request.Sentences)], nil
}

func TestOptionalSentenceRepairImprovesBrokenFixtureCases(t *testing.T) {
	model, err := NewModel("gpt-4.1", 8192)
	if err != nil {
		t.Fatalf("new model: %v", err)
	}

	config := SentenceRepairConfig{Enabled: true, MinConfidence: 0.8}
	evalCases := []struct {
		name      string
		sentences []string
		want      []string
	}{
		{
			name: "real fixture held page reference",
			sentences: []string{
				"Held: Courts should review the Board's determination that a given set of facts does not rise to the level of persecution under a substantial-evidence standard.",
				"Pp.",
				"3-10.",
			},
			want: []string{
				"Held: Courts should review the Board's determination that a given set of facts does not rise to the level of persecution under a substantial-evidence standard. Pp. 3-10.",
			},
		},
		{
			name: "real fixture trailing statute citation",
			sentences: []string{
				"The Court has previously interpreted this provision to prescribe a deferential substantial-evidence standard, Nasrallah v. Barr, 590 U. S. 573, 584, under which administrative findings of fact are conclusive unless any reasonable adjudicator would be compelled to conclude to the contrary,",
				"8 U. S. C. §1252(b)(4)(B).",
			},
			want: []string{
				"The Court has previously interpreted this provision to prescribe a deferential substantial-evidence standard, Nasrallah v. Barr, 590 U. S. 573, 584, under which administrative findings of fact are conclusive unless any reasonable adjudicator would be compelled to conclude to the contrary, 8 U. S. C. §1252(b)(4)(B).",
			},
		},
		{
			name: "focused edge case id pin cite",
			sentences: []string{
				"The court rejected that argument.",
				"Id.",
				", at 484.",
				"The Government presses a different theory.",
			},
			want: []string{
				"The court rejected that argument.",
				"Id., at 484.",
				"The Government presses a different theory.",
			},
		},
	}

	guesser := cannedSentenceRepairGuesser{
		guesses: map[string][]SentenceRepairGuess{
			sentenceRepairEvalKey(evalCases[0].sentences): {
				{
					LeftIndex:  0,
					RightIndex: 2,
					MergedText: Text(evalCases[0].want[0]),
					Confidence: 0.96,
				},
			},
			sentenceRepairEvalKey(evalCases[1].sentences): {
				{
					LeftIndex:  0,
					RightIndex: 1,
					MergedText: Text(evalCases[1].want[0]),
					Confidence: 0.95,
				},
			},
			sentenceRepairEvalKey(evalCases[2].sentences): {
				{
					LeftIndex:  1,
					RightIndex: 2,
					MergedText: Text(evalCases[2].want[1]),
					Confidence: 0.94,
				},
			},
		},
	}

	for _, testCase := range evalCases {
		t.Run(testCase.name, func(t *testing.T) {
			heuristicOnly := append([]string(nil), testCase.sentences...)
			repaired := MaybeGuessSentenceRepairs(context.Background(), model, config, guesser, testCase.sentences)

			if sameSentences(heuristicOnly, testCase.want) {
				t.Fatalf("heuristic baseline should still exhibit the broken segmentation")
			}
			if !sameSentences(repaired, testCase.want) {
				t.Fatalf("got repaired %#v want %#v", repaired, testCase.want)
			}
		})
	}
}

func TestOptionalSentenceRepairLeavesCleanFixtureCasesUnchanged(t *testing.T) {
	model, err := NewModel("gpt-4.1", 8192)
	if err != nil {
		t.Fatalf("new model: %v", err)
	}

	config := SentenceRepairConfig{Enabled: true, MinConfidence: 0.8}
	corpus := loadQualityCorpus(t)
	guesser := cannedSentenceRepairGuesser{guesses: map[string][]SentenceRepairGuess{}}

	for _, testCase := range corpus.Cases {
		if testCase.Kind != "sentence_boundary" {
			continue
		}

		t.Run(testCase.ID, func(t *testing.T) {
			heuristicOnly := splitSentences(testCase.Input)
			repaired := MaybeGuessSentenceRepairs(context.Background(), model, config, guesser, heuristicOnly)

			if !sameSentences(heuristicOnly, testCase.WantSentences) {
				t.Fatalf("heuristic output drifted from the quality corpus: %#v", heuristicOnly)
			}
			if !sameSentences(repaired, heuristicOnly) {
				t.Fatalf("repair pass changed a clean segmentation: %#v", repaired)
			}
		})
	}
}

func sameSentences(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func sentenceRepairEvalKey(sentences []string) string {
	return textKey(sentences)
}

func textKey(values []string) string {
	return string(Text(joinWithUnitSeparator(values)))
}

func joinWithUnitSeparator(values []string) string {
	if len(values) == 0 {
		return ""
	}
	joined := values[0]
	for _, value := range values[1:] {
		joined += "\x1f" + value
	}
	return joined
}
