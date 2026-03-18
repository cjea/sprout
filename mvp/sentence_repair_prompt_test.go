package mvp

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestBuildSentenceRepairPromptIncludesCandidatesAndOffsets(t *testing.T) {
	request := sentenceRepairRequestFixture()

	prompt, err := BuildSentenceRepairPrompt(request)
	if err != nil {
		t.Fatalf("build prompt: %v", err)
	}

	wantSnippets := []string{
		`"action":"keep|merge"`,
		"source_text:",
		"candidates:",
		"reasons=page_reference_lead,page_range_continuation",
		"reasons=trailing_authority",
		`start_text="Held: Courts should review the Board's determination."`,
		`end_text="5-13."`,
	}
	for _, snippet := range wantSnippets {
		if !strings.Contains(prompt, snippet) {
			t.Fatalf("prompt missing %q", snippet)
		}
	}
}

func TestParseSentenceRepairOutputAcceptsOrderedValidatedDecisions(t *testing.T) {
	request := sentenceRepairRequestFixture()
	_, candidates, err := buildSentenceRepairCandidates(request)
	if err != nil {
		t.Fatalf("build candidates: %v", err)
	}

	raw := `{
		"decisions": [
			{"start_offset": %d, "end_offset": %d, "action": "merge", "merged_text": "Held: Courts should review the Board's determination. Pp. 5-13.", "confidence": 0.96},
			{"start_offset": %d, "end_offset": %d, "action": "merge", "merged_text": "Under the INA, the U. S. Government may grant asylum to a noncitizen if it determines that he is a refugee. 8 U. S. C. §1158(b)(1)(A).", "confidence": 0.94}
		]
	}`
	raw = fmt.Sprintf(
		raw,
		candidates[0].WindowSpan.StartOffset,
		candidates[0].WindowSpan.EndOffset,
		candidates[1].WindowSpan.StartOffset,
		candidates[1].WindowSpan.EndOffset,
	)

	guesses, err := ParseSentenceRepairOutput(request, raw)
	if err != nil {
		t.Fatalf("parse output: %v", err)
	}
	if len(guesses) != 2 {
		t.Fatalf("got %d guesses want 2", len(guesses))
	}
	if guesses[0].MergedText != "Held: Courts should review the Board's determination. Pp. 5-13." {
		t.Fatalf("got first merge %q", guesses[0].MergedText)
	}
	if guesses[1].RightIndex != 4 {
		t.Fatalf("got right index %d want 4", guesses[1].RightIndex)
	}
}

func TestParseSentenceRepairOutputRejectsInvalidAction(t *testing.T) {
	request := sentenceRepairRequestFixture()
	_, candidates, err := buildSentenceRepairCandidates(request)
	if err != nil {
		t.Fatalf("build candidates: %v", err)
	}

	raw := `{
		"decisions": [
			{"start_offset": %d, "end_offset": %d, "action": "rewrite", "merged_text": "Held: Courts should review the Board's determination. Pp. 5-13.", "confidence": 0.96},
			{"start_offset": %d, "end_offset": %d, "action": "keep", "confidence": 0.94}
		]
	}`
	raw = fmt.Sprintf(
		raw,
		candidates[0].WindowSpan.StartOffset,
		candidates[0].WindowSpan.EndOffset,
		candidates[1].WindowSpan.StartOffset,
		candidates[1].WindowSpan.EndOffset,
	)

	_, err = ParseSentenceRepairOutput(request, raw)
	if !errors.Is(err, ErrSentenceRepairOutputInvalidAction) {
		t.Fatalf("got err %v want %v", err, ErrSentenceRepairOutputInvalidAction)
	}
}

func TestParseSentenceRepairOutputRejectsAmbiguousOverlap(t *testing.T) {
	request := sentenceRepairRequestFixture()
	_, candidates, err := buildSentenceRepairCandidates(request)
	if err != nil {
		t.Fatalf("build candidates: %v", err)
	}

	raw := `{
		"decisions": [
			{"start_offset": %d, "end_offset": %d, "action": "merge", "merged_text": "Held: Courts should review the Board's determination. Pp. 5-13. Under the INA, the U. S. Government may grant asylum to a noncitizen if it determines that he is a refugee. 8 U. S. C. §1158(b)(1)(A).", "confidence": 0.96},
			{"start_offset": %d, "end_offset": %d, "action": "merge", "merged_text": "Under the INA, the U. S. Government may grant asylum to a noncitizen if it determines that he is a refugee. 8 U. S. C. §1158(b)(1)(A).", "confidence": 0.94}
		]
	}`
	raw = fmt.Sprintf(
		raw,
		candidates[0].WindowSpan.StartOffset,
		candidates[1].WindowSpan.EndOffset,
		candidates[1].WindowSpan.StartOffset,
		candidates[1].WindowSpan.EndOffset,
	)

	_, err = ParseSentenceRepairOutput(request, raw)
	if !errors.Is(err, ErrSentenceRepairOutputInvalidSpan) && !errors.Is(err, ErrSentenceRepairOutputAmbiguous) {
		t.Fatalf("got err %v want ambiguous or invalid span", err)
	}
}

func TestParseSentenceRepairOutputRejectsDestructiveMerge(t *testing.T) {
	request := sentenceRepairRequestFixture()
	_, candidates, err := buildSentenceRepairCandidates(request)
	if err != nil {
		t.Fatalf("build candidates: %v", err)
	}

	raw := `{
		"decisions": [
			{"start_offset": %d, "end_offset": %d, "action": "merge", "merged_text": "pages 5 through 13", "confidence": 0.96},
			{"start_offset": %d, "end_offset": %d, "action": "keep", "confidence": 0.94}
		]
	}`
	raw = fmt.Sprintf(
		raw,
		candidates[0].WindowSpan.StartOffset,
		candidates[0].WindowSpan.EndOffset,
		candidates[1].WindowSpan.StartOffset,
		candidates[1].WindowSpan.EndOffset,
	)

	_, err = ParseSentenceRepairOutput(request, raw)
	if !errors.Is(err, ErrSentenceRepairOutputDestructive) {
		t.Fatalf("got err %v want %v", err, ErrSentenceRepairOutputDestructive)
	}
}

func TestParseSentenceRepairOutputRejectsMissingCoverage(t *testing.T) {
	request := sentenceRepairRequestFixture()
	_, candidates, err := buildSentenceRepairCandidates(request)
	if err != nil {
		t.Fatalf("build candidates: %v", err)
	}

	raw := `{
		"decisions": [
			{"start_offset": %d, "end_offset": %d, "action": "merge", "merged_text": "Held: Courts should review the Board's determination. Pp. 5-13.", "confidence": 0.96}
		]
	}`
	raw = fmt.Sprintf(
		raw,
		candidates[0].WindowSpan.StartOffset,
		candidates[0].WindowSpan.EndOffset,
	)

	_, err = ParseSentenceRepairOutput(request, raw)
	if !errors.Is(err, ErrSentenceRepairGuessMissingBoundary) {
		t.Fatalf("got err %v want %v", err, ErrSentenceRepairGuessMissingBoundary)
	}
}

func sentenceRepairRequestFixture() SentenceRepairRequest {
	sentences := []string{
		"Held: Courts should review the Board's determination.",
		"Pp.",
		"5-13.",
		"Under the INA, the U. S. Government may grant asylum to a noncitizen if it determines that he is a refugee.",
		"8 U. S. C. §1158(b)(1)(A).",
	}
	return SentenceRepairRequest{
		Sentences:            sentences,
		SuspiciousBoundaries: DetectSuspiciousSentenceBoundaries(sentences),
	}
}
