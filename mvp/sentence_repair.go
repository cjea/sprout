package mvp

import (
	"regexp"
	"strings"
)

var (
	pageRangeFragmentPattern = regexp.MustCompile(`^\d+[–-]\d+\.$`)
	pinCiteFragmentPattern   = regexp.MustCompile(`^,?\s*at\s+\d+\.$`)
)

type BoundarySuspicion string

const (
	BoundarySuspicionPageReferenceLead     BoundarySuspicion = "page_reference_lead"
	BoundarySuspicionPageRangeContinuation BoundarySuspicion = "page_range_continuation"
	BoundarySuspicionPinCiteContinuation   BoundarySuspicion = "pin_cite_continuation"
	BoundarySuspicionTrailingAuthority     BoundarySuspicion = "trailing_authority"
)

type SuspiciousSentenceBoundary struct {
	LeftIndex  SentenceNo
	RightIndex SentenceNo
	LeftText   Text
	RightText  Text
	Reason     BoundarySuspicion
}

func DetectSuspiciousSentenceBoundaries(sentences []string) []SuspiciousSentenceBoundary {
	if len(sentences) < 2 {
		return nil
	}

	boundaries := make([]SuspiciousSentenceBoundary, 0)
	for index := 1; index < len(sentences); index++ {
		left := strings.TrimSpace(sentences[index-1])
		right := strings.TrimSpace(sentences[index])
		reason, suspicious := classifyBoundarySuspicion(left, right)
		if !suspicious {
			continue
		}
		boundaries = append(boundaries, SuspiciousSentenceBoundary{
			LeftIndex:  SentenceNo(index - 1),
			RightIndex: SentenceNo(index),
			LeftText:   Text(left),
			RightText:  Text(right),
			Reason:     reason,
		})
	}
	return boundaries
}

func classifyBoundarySuspicion(left, right string) (BoundarySuspicion, bool) {
	switch {
	case strings.EqualFold(right, "Pp."):
		return BoundarySuspicionPageReferenceLead, true
	case strings.EqualFold(left, "Pp.") && pageRangeFragmentPattern.MatchString(right):
		return BoundarySuspicionPageRangeContinuation, true
	case strings.EqualFold(left, "Id.") && pinCiteFragmentPattern.MatchString(right):
		return BoundarySuspicionPinCiteContinuation, true
	case isCitationContinuation(right):
		return BoundarySuspicionTrailingAuthority, true
	default:
		return "", false
	}
}
