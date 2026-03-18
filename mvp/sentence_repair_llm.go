package mvp

import (
	"context"
	"errors"
	"fmt"
)

var (
	ErrSentenceRepairGuessMissingBoundary = errors.New("sentence repair guess is missing a suspicious boundary")
	ErrSentenceRepairGuessMalformed       = errors.New("sentence repair guess is malformed")
	ErrSentenceRepairGuessLowConfidence   = errors.New("sentence repair guess confidence is too low")
)

type SentenceRepairConfig struct {
	Enabled       bool
	MinConfidence float64
}

func DefaultSentenceRepairConfig() SentenceRepairConfig {
	return SentenceRepairConfig{
		Enabled:       false,
		MinConfidence: 0.8,
	}
}

type SentenceRepairRequest struct {
	Sentences            []string
	SuspiciousBoundaries []SuspiciousSentenceBoundary
}

type SentenceRepairGuess struct {
	LeftIndex  SentenceNo
	RightIndex SentenceNo
	MergedText Text
	Confidence float64
}

type SentenceRepairGuesser interface {
	GuessSentenceRepairs(ctx context.Context, model Model, request SentenceRepairRequest) ([]SentenceRepairGuess, error)
}

func MaybeGuessSentenceRepairs(
	ctx context.Context,
	model Model,
	config SentenceRepairConfig,
	guesser SentenceRepairGuesser,
	sentences []string,
) []string {
	if !config.Enabled || guesser == nil || len(sentences) < 2 {
		return append([]string(nil), sentences...)
	}

	suspiciousBoundaries := DetectSuspiciousSentenceBoundaries(sentences)
	if len(suspiciousBoundaries) == 0 {
		return append([]string(nil), sentences...)
	}

	request := SentenceRepairRequest{
		Sentences:            append([]string(nil), sentences...),
		SuspiciousBoundaries: suspiciousBoundaries,
	}
	guesses, err := guesser.GuessSentenceRepairs(ctx, model, request)
	if err != nil {
		return append([]string(nil), sentences...)
	}
	if err := validateSentenceRepairGuesses(request, config, guesses); err != nil {
		return append([]string(nil), sentences...)
	}

	repaired, err := applySentenceRepairGuesses(request.Sentences, guesses)
	if err != nil {
		return append([]string(nil), sentences...)
	}
	return repaired
}

func validateSentenceRepairGuesses(request SentenceRepairRequest, config SentenceRepairConfig, guesses []SentenceRepairGuess) error {
	_, candidates, err := buildSentenceRepairCandidates(request)
	if err != nil {
		return err
	}
	if len(guesses) != len(candidates) {
		return ErrSentenceRepairGuessMissingBoundary
	}

	expected := make(map[string]SentenceRepairCandidate, len(candidates))
	for _, candidate := range candidates {
		expected[sentenceRepairBoundaryKey(candidate.StartIndex, candidate.EndIndex)] = candidate
	}

	for _, guess := range guesses {
		if guess.RightIndex <= guess.LeftIndex {
			return ErrSentenceRepairGuessMalformed
		}
		if guess.LeftIndex < 0 || guess.RightIndex >= SentenceNo(len(request.Sentences)) {
			return ErrSentenceRepairGuessMalformed
		}
		if guess.MergedText == "" {
			return ErrSentenceRepairGuessMalformed
		}
		if guess.Confidence < config.MinConfidence {
			return ErrSentenceRepairGuessLowConfidence
		}

		key := sentenceRepairBoundaryKey(guess.LeftIndex, guess.RightIndex)
		if _, ok := expected[key]; !ok {
			return fmt.Errorf("%w: %s", ErrSentenceRepairGuessMissingBoundary, key)
		}
		delete(expected, key)
	}

	if len(expected) != 0 {
		return ErrSentenceRepairGuessMissingBoundary
	}
	return nil
}

func applySentenceRepairGuesses(sentences []string, guesses []SentenceRepairGuess) ([]string, error) {
	if len(guesses) == 0 {
		return append([]string(nil), sentences...), nil
	}

	mergedByLeft := make(map[int]SentenceRepairGuess, len(guesses))
	skipped := make(map[int]struct{}, len(sentences))
	for _, guess := range guesses {
		left := int(guess.LeftIndex)
		right := int(guess.RightIndex)
		if _, exists := mergedByLeft[left]; exists {
			return nil, ErrSentenceRepairGuessMalformed
		}
		for index := left + 1; index <= right; index++ {
			if _, exists := skipped[index]; exists {
				return nil, ErrSentenceRepairGuessMalformed
			}
		}
		mergedByLeft[left] = guess
		for index := left + 1; index <= right; index++ {
			skipped[index] = struct{}{}
		}
	}

	repaired := make([]string, 0, len(sentences)-len(guesses))
	for index := 0; index < len(sentences); index++ {
		if guess, ok := mergedByLeft[index]; ok {
			repaired = append(repaired, string(guess.MergedText))
			index = int(guess.RightIndex)
			continue
		}
		if _, ok := skipped[index]; ok {
			continue
		}
		repaired = append(repaired, sentences[index])
	}
	return repaired, nil
}

func sentenceRepairBoundaryKey(left, right SentenceNo) string {
	return fmt.Sprintf("%d:%d", left, right)
}
