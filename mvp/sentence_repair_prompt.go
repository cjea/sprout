package mvp

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	ErrSentenceRepairOutputInvalidJSON   = errors.New("sentence repair output is not valid json")
	ErrSentenceRepairOutputInvalidAction = errors.New("sentence repair output has an invalid action")
	ErrSentenceRepairOutputInvalidSpan   = errors.New("sentence repair output has an invalid span")
	ErrSentenceRepairOutputAmbiguous     = errors.New("sentence repair output is ambiguous")
	ErrSentenceRepairOutputDestructive   = errors.New("sentence repair output is destructive")
)

var wordTokenPattern = regexp.MustCompile(`[A-Za-z0-9]+`)

type SentenceRepairAction string

const (
	SentenceRepairActionKeep  SentenceRepairAction = "keep"
	SentenceRepairActionMerge SentenceRepairAction = "merge"
)

type SentenceRepairCandidate struct {
	StartIndex SentenceNo
	EndIndex   SentenceNo
	StartSpan  Span
	EndSpan    Span
	WindowSpan Span
	Reasons    []BoundarySuspicion
}

type SentenceRepairOutput struct {
	Decisions []SentenceRepairDecisionOutput `json:"decisions"`
}

type SentenceRepairDecisionOutput struct {
	StartOffset Offset               `json:"start_offset"`
	EndOffset   Offset               `json:"end_offset"`
	Action      SentenceRepairAction `json:"action"`
	MergedText  string               `json:"merged_text,omitempty"`
	Confidence  float64              `json:"confidence"`
}

func BuildSentenceRepairPrompt(request SentenceRepairRequest) (string, error) {
	sourceText, candidates, err := buildSentenceRepairCandidates(request)
	if err != nil {
		return "", err
	}

	var builder strings.Builder
	builder.WriteString("Repair only suspicious sentence boundaries in Supreme Court opinion text.\n")
	builder.WriteString("Return JSON with one decision per candidate window.\n")
	builder.WriteString("Allowed actions: keep, merge.\n")
	builder.WriteString("Do not rewrite outside the candidate window.\n")
	builder.WriteString("Use these offsets against source_text exactly.\n")
	builder.WriteString("JSON schema:\n")
	builder.WriteString(`{"decisions":[{"start_offset":0,"end_offset":0,"action":"keep|merge","merged_text":"","confidence":0.0}]}`)
	builder.WriteString("\nsource_text:\n")
	builder.WriteString(string(sourceText))
	builder.WriteString("\ncandidates:\n")
	for _, candidate := range candidates {
		builder.WriteString(fmt.Sprintf(
			"- start=%d end=%d reasons=%s window=[%d,%d] start_text=%q end_text=%q\n",
			candidate.StartIndex,
			candidate.EndIndex,
			strings.Join(boundarySuspicionStrings(candidate.Reasons), ","),
			candidate.WindowSpan.StartOffset,
			candidate.WindowSpan.EndOffset,
			candidate.StartSpan.Quote,
			candidate.EndSpan.Quote,
		))
	}
	return builder.String(), nil
}

func ParseSentenceRepairOutput(request SentenceRepairRequest, raw string) ([]SentenceRepairGuess, error) {
	sourceText, candidates, err := buildSentenceRepairCandidates(request)
	if err != nil {
		return nil, err
	}

	var output SentenceRepairOutput
	if err := json.Unmarshal([]byte(raw), &output); err != nil {
		return nil, ErrSentenceRepairOutputInvalidJSON
	}

	if len(output.Decisions) != len(candidates) {
		return nil, ErrSentenceRepairGuessMissingBoundary
	}

	candidateByWindow := make(map[string]SentenceRepairCandidate, len(candidates))
	for _, candidate := range candidates {
		candidateByWindow[sentenceRepairOffsetKey(candidate.WindowSpan.StartOffset, candidate.WindowSpan.EndOffset)] = candidate
	}

	guesses := make([]SentenceRepairGuess, 0, len(output.Decisions))
	lastEnd := Offset(-1)
	for _, decision := range output.Decisions {
		if decision.StartOffset < 0 || decision.EndOffset <= decision.StartOffset {
			return nil, ErrSentenceRepairOutputInvalidSpan
		}
		if decision.StartOffset < lastEnd {
			return nil, ErrSentenceRepairOutputAmbiguous
		}
		lastEnd = decision.EndOffset
		if decision.Confidence < 0 || decision.Confidence > 1 {
			return nil, ErrSentenceRepairGuessMalformed
		}

		key := sentenceRepairOffsetKey(decision.StartOffset, decision.EndOffset)
		candidate, ok := candidateByWindow[key]
		if !ok {
			return nil, ErrSentenceRepairOutputInvalidSpan
		}
		delete(candidateByWindow, key)

		switch decision.Action {
		case SentenceRepairActionKeep:
			if strings.TrimSpace(decision.MergedText) != "" {
				return nil, ErrSentenceRepairOutputAmbiguous
			}
		case SentenceRepairActionMerge:
			if strings.TrimSpace(decision.MergedText) == "" {
				return nil, ErrSentenceRepairOutputDestructive
			}
			windowText := string(sourceText[candidate.WindowSpan.StartOffset:candidate.WindowSpan.EndOffset])
			if !preservesRepairWindow(windowText, decision.MergedText) {
				return nil, ErrSentenceRepairOutputDestructive
			}
			guesses = append(guesses, SentenceRepairGuess{
				LeftIndex:  candidate.StartIndex,
				RightIndex: candidate.EndIndex,
				MergedText: Text(strings.TrimSpace(decision.MergedText)),
				Confidence: decision.Confidence,
			})
		default:
			return nil, ErrSentenceRepairOutputInvalidAction
		}
	}

	if len(candidateByWindow) != 0 {
		return nil, ErrSentenceRepairGuessMissingBoundary
	}
	return guesses, nil
}

func buildSentenceRepairCandidates(request SentenceRepairRequest) (Text, []SentenceRepairCandidate, error) {
	sourceText, sentenceSpans, err := buildSentenceRepairSourceText(request.Sentences)
	if err != nil {
		return "", nil, err
	}

	candidates := make([]SentenceRepairCandidate, 0, len(request.SuspiciousBoundaries))
	for _, cluster := range groupSuspiciousBoundaries(request.SuspiciousBoundaries) {
		startSpan := sentenceSpans[int(cluster.StartIndex)]
		endSpan := sentenceSpans[int(cluster.EndIndex)]
		windowSpan, err := NewSpan(startSpan.StartOffset, endSpan.EndOffset, Text(sourceText[startSpan.StartOffset:endSpan.EndOffset]))
		if err != nil {
			return "", nil, err
		}
		candidates = append(candidates, SentenceRepairCandidate{
			StartIndex: cluster.StartIndex,
			EndIndex:   cluster.EndIndex,
			StartSpan:  startSpan,
			EndSpan:    endSpan,
			WindowSpan: windowSpan,
			Reasons:    cluster.Reasons,
		})
	}
	return sourceText, candidates, nil
}

func buildSentenceRepairSourceText(sentences []string) (Text, []Span, error) {
	joined := strings.Join(sentences, "\n")
	sourceText := Text(joined)
	spans := make([]Span, 0, len(sentences))
	offset := Offset(0)
	for _, sentence := range sentences {
		end := offset + Offset(len(sentence))
		span, err := NewSpan(offset, end, Text(sentence))
		if err != nil {
			return "", nil, err
		}
		spans = append(spans, span)
		offset = end + 1
	}
	return sourceText, spans, nil
}

func preservesRepairWindow(windowText, mergedText string) bool {
	windowTokens := wordTokenPattern.FindAllString(strings.ToLower(windowText), -1)
	mergedTokens := wordTokenPattern.FindAllString(strings.ToLower(mergedText), -1)
	if len(windowTokens) == 0 || len(mergedTokens) == 0 {
		return false
	}

	cursor := 0
	for _, token := range windowTokens {
		found := false
		for cursor < len(mergedTokens) {
			if mergedTokens[cursor] == token {
				found = true
				cursor++
				break
			}
			cursor++
		}
		if !found {
			return false
		}
	}
	return true
}

func sentenceRepairOffsetKey(start, end Offset) string {
	return fmt.Sprintf("%d:%d", start, end)
}

type suspiciousBoundaryCluster struct {
	StartIndex SentenceNo
	EndIndex   SentenceNo
	Reasons    []BoundarySuspicion
}

func groupSuspiciousBoundaries(boundaries []SuspiciousSentenceBoundary) []suspiciousBoundaryCluster {
	if len(boundaries) == 0 {
		return nil
	}

	clusters := make([]suspiciousBoundaryCluster, 0, len(boundaries))
	current := suspiciousBoundaryCluster{
		StartIndex: boundaries[0].LeftIndex,
		EndIndex:   boundaries[0].RightIndex,
		Reasons:    []BoundarySuspicion{boundaries[0].Reason},
	}
	for _, boundary := range boundaries[1:] {
		if boundary.LeftIndex <= current.EndIndex {
			if boundary.RightIndex > current.EndIndex {
				current.EndIndex = boundary.RightIndex
			}
			current.Reasons = append(current.Reasons, boundary.Reason)
			continue
		}
		clusters = append(clusters, current)
		current = suspiciousBoundaryCluster{
			StartIndex: boundary.LeftIndex,
			EndIndex:   boundary.RightIndex,
			Reasons:    []BoundarySuspicion{boundary.Reason},
		}
	}
	clusters = append(clusters, current)
	return clusters
}

func boundarySuspicionStrings(reasons []BoundarySuspicion) []string {
	values := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		values = append(values, string(reason))
	}
	return values
}
