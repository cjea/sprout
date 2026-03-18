package mvp

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	caseCitationPattern         = regexp.MustCompile(`([A-Z][A-Za-z.&'\-]+ v\. [A-Z][A-Za-z.&'\-]+)`)
	statuteCitationPattern      = regexp.MustCompile(`([0-9]+ U\.?\s*S\.?\s*C\.?\s*§+\s*[0-9A-Za-z\-]+)`)
	sentenceSplitPattern        = regexp.MustCompile(`[^.!?]+(?:[.!?](?:["'”’)\]]+)?)?`)
	slipOpinionHeaderPattern    = regexp.MustCompile(`(?i)^\s*\d+\s+\(Slip Opinion\)\s+OCTOBER TERM,\s+\d{4}\s+Syllabus\s*`)
	runningHeadPattern          = regexp.MustCompile(`(?im)\s*\n?\s*\d+\s+[A-Z][A-Z .,'\-]+ v\. [A-Z][A-Za-z .,'\-]+(?:\s+Syllabus|\s+Opinion of the Court)\s*`)
	citationContinuationPattern = regexp.MustCompile(`^(?:[0-9]+ U\.?\s*S\.?\s*C\.?\s*§+\s*[0-9A-Za-z\-().]+|§[0-9A-Za-z\-().]+|[A-Z][A-Za-z.&'\-]+ v\. [A-Z][A-Za-z.&'\-]+)`)
	lineBreakHyphenPattern      = regexp.MustCompile(`([A-Za-z]{1,20})-\s*\n\s*([A-Za-z]{1,20})`)
)

func ChunkSections(policy ChunkPolicy, sections []Section) ([]Passage, error) {
	if err := policy.Validate(); err != nil {
		return nil, err
	}

	var passages []Passage
	for _, section := range sections {
		sentences := splitSentences(cleanPassageSourceText(string(section.Text)))
		for start, ordinal := 0, 0; start < len(sentences); ordinal++ {
			end := start + policy.TargetSentences
			if end > len(sentences) {
				end = len(sentences)
			}
			if span := end - start; span > policy.MaxSentences {
				end = start + policy.MaxSentences
			}

			text := strings.TrimSpace(strings.Join(sentences[start:end], " "))
			passageID, err := NewPassageID(fmt.Sprintf("%s-%d", section.SectionID, ordinal+1))
			if err != nil {
				return nil, err
			}
			passage, err := NewPassage(
				passageID,
				section.OpinionID,
				section.SectionID,
				SentenceNo(start),
				SentenceNo(end-1),
				section.StartPage,
				section.EndPage,
				Text(text),
				nil,
				false,
			)
			if err != nil {
				return nil, err
			}
			passages = append(passages, passage)
			start = end
		}
	}
	return passages, nil
}

func ExtractCitations(passage Passage) ([]Citation, error) {
	if err := passage.Validate(); err != nil {
		return nil, err
	}

	var citations []Citation
	addMatches := func(pattern *regexp.Regexp, kind CitationKind) error {
		matches := pattern.FindAllStringIndex(string(passage.Text), -1)
		for index, match := range matches {
			raw := string(passage.Text[match[0]:match[1]])
			span, err := NewSpan(Offset(match[0]), Offset(match[1]), Text(raw))
			if err != nil {
				return err
			}
			citationID, err := NewCitationID(fmt.Sprintf("%s-%s-%d", passage.PassageID, kind, index+1))
			if err != nil {
				return err
			}
			citation, err := NewCitation(citationID, kind, raw, nil, span)
			if err != nil {
				return err
			}
			citations = append(citations, citation)
		}
		return nil
	}

	if err := addMatches(caseCitationPattern, CitationKindCase); err != nil {
		return nil, err
	}
	if err := addMatches(statuteCitationPattern, CitationKindStatute); err != nil {
		return nil, err
	}

	return citations, nil
}

func NormalizeCitations(citations []Citation) ([]Citation, error) {
	normalized := make([]Citation, 0, len(citations))
	for _, citation := range citations {
		raw := strings.Join(strings.Fields(citation.RawText), " ")
		copy := citation
		copy.RawText = raw
		normalizedText := raw
		copy.Normalized = &normalizedText
		if err := copy.Validate(); err != nil {
			return nil, err
		}
		normalized = append(normalized, copy)
	}
	return normalized, nil
}

func AttachCitations(passages []Passage) ([]Passage, error) {
	attached := make([]Passage, 0, len(passages))
	for _, passage := range passages {
		citations, err := ExtractCitations(passage)
		if err != nil {
			return nil, err
		}
		citations, err = NormalizeCitations(citations)
		if err != nil {
			return nil, err
		}
		updated, err := NewPassage(
			passage.PassageID,
			passage.OpinionID,
			passage.SectionID,
			passage.SentenceStart,
			passage.SentenceEnd,
			passage.PageStart,
			passage.PageEnd,
			passage.Text,
			citations,
			passage.FitsOnScreen,
		)
		if err != nil {
			return nil, err
		}
		attached = append(attached, updated)
	}
	return attached, nil
}

func FitPassage(policy ScreenPolicy, passage Passage) PassageFit {
	if err := policy.Validate(); err != nil {
		return PassageFitNeedsRepair
	}

	lines := strings.Count(string(passage.Text), "\n") + 1
	characters := len(string(passage.Text))
	switch {
	case lines <= policy.MaxRenderedLines && characters <= policy.MaxCharacters:
		return PassageFitFitsScreen
	case passage.SentenceStart == passage.SentenceEnd:
		return PassageFitTooLong
	default:
		return PassageFitNeedsRepair
	}
}

func RepairPassages(policy ScreenPolicy, passages []Passage) ([]Passage, error) {
	if err := policy.Validate(); err != nil {
		return nil, err
	}

	var repaired []Passage
	for _, passage := range passages {
		fit := FitPassage(policy, passage)
		if fit == PassageFitFitsScreen || fit == PassageFitTooLong {
			passage.FitsOnScreen = fit == PassageFitFitsScreen
			repaired = append(repaired, passage)
			continue
		}

		sentences := splitSentences(cleanPassageSourceText(string(passage.Text)))
		for index, sentence := range sentences {
			childID, err := NewPassageID(fmt.Sprintf("%s-r%d", passage.PassageID, index+1))
			if err != nil {
				return nil, err
			}
			child, err := NewPassage(
				childID,
				passage.OpinionID,
				passage.SectionID,
				passage.SentenceStart+SentenceNo(index),
				passage.SentenceStart+SentenceNo(index),
				passage.PageStart,
				passage.PageEnd,
				Text(sentence),
				nil,
				len(sentence) <= policy.MaxCharacters,
			)
			if err != nil {
				return nil, err
			}
			repaired = append(repaired, child)
		}
	}

	return AttachCitations(repaired)
}

func StorePassages(storage Storage, passages []Passage) ([]Passage, error) {
	return savePassages(storage, passages)
}

func splitSentences(text string) []string {
	protected := protectAbbreviations(text)
	matches := sentenceSplitPattern.FindAllString(protected, -1)
	sentences := make([]string, 0, len(matches))
	for _, match := range matches {
		trimmed := strings.TrimSpace(restoreAbbreviations(match))
		if trimmed != "" {
			sentences = append(sentences, trimmed)
		}
	}
	if len(sentences) == 0 && strings.TrimSpace(text) != "" {
		return []string{strings.TrimSpace(text)}
	}
	return mergeSentenceContinuations(sentences)
}

func cleanPassageSourceText(text string) string {
	cleaned := slipOpinionHeaderPattern.ReplaceAllString(text, "")
	cleaned = runningHeadPattern.ReplaceAllString(cleaned, " ")
	cleaned = repairLineBreakHyphenation(cleaned)
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	cleaned = repairKnownHyphenationArtifacts(cleaned)
	cleaned = repairKnownJoinedWordArtifacts(cleaned)
	return strings.TrimSpace(cleaned)
}

func protectAbbreviations(text string) string {
	replacer := strings.NewReplacer(
		"Pp.", "Pp§",
		"Id.", "Id§",
		"v.", "v§",
		"U.S.C.", "U§S§C§",
		"U. S. C.", "U§ S§ C§",
		"U. S. C", "U§ S§ C",
		"U.S.", "U§S§",
		"U. S.", "U§ S§",
		"Inc.", "Inc§",
	)
	return replacer.Replace(text)
}

func restoreAbbreviations(text string) string {
	replacer := strings.NewReplacer(
		"Pp§", "Pp.",
		"Id§", "Id.",
		"v§", "v.",
		"U§S§C§", "U.S.C.",
		"U§ S§ C§", "U. S. C.",
		"U§ S§ C", "U. S. C",
		"U§S§", "U.S.",
		"U§ S§", "U. S.",
		"Inc§", "Inc.",
	)
	return replacer.Replace(text)
}

func mergeSentenceContinuations(sentences []string) []string {
	merged := make([]string, 0, len(sentences))
	for _, sentence := range sentences {
		if len(merged) == 0 {
			merged = append(merged, sentence)
			continue
		}
		if isCitationContinuation(sentence) {
			merged[len(merged)-1] = strings.TrimSpace(merged[len(merged)-1] + " " + sentence)
			continue
		}
		merged = append(merged, sentence)
	}
	return merged
}

func isCitationContinuation(sentence string) bool {
	trimmed := strings.TrimSpace(sentence)
	if trimmed == "" {
		return false
	}
	if citationContinuationPattern.MatchString(trimmed) {
		return true
	}
	if strings.HasPrefix(trimmed, "Pp. ") {
		return true
	}
	if strings.HasPrefix(trimmed, "” ") || strings.HasPrefix(trimmed, "\" ") || strings.HasPrefix(trimmed, "’ ") {
		rest := strings.TrimSpace(strings.TrimLeft(trimmed, "\"”’"))
		return citationContinuationPattern.MatchString(rest) || strings.HasPrefix(rest, "Pp. ")
	}
	return false
}

func repairLineBreakHyphenation(text string) string {
	protectedCompounds := map[string]struct{}{
		"substantial-evidence": {},
		"well-founded":         {},
	}

	return lineBreakHyphenPattern.ReplaceAllStringFunc(text, func(match string) string {
		parts := lineBreakHyphenPattern.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}

		left := parts[1]
		right := parts[2]
		compound := strings.ToLower(left + "-" + right)
		if _, ok := protectedCompounds[compound]; ok {
			return left + "-" + right
		}
		return left + right
	})
}

func repairKnownHyphenationArtifacts(text string) string {
	replacer := strings.NewReplacer(
		"asy-lum", "asylum",
		"refu-gee", "refugee",
		"ac-count", "account",
		"per-secution", "persecution",
		"pre-scribe", "prescribe",
		"rea-sonable", "reasonable",
		"de-termination", "determination",
		"con-stitute", "constitute",
		"or-dered", "ordered",
		"ei-ther", "either",
		"underly-ing", "underlying",
		"signifi-cant", "significant",
		"subpar-agraph", "subparagraph",
		"partic-ular", "particular",
		"pri-marily", "primarily",
		"re-view", "review",
		"noncit- izen", "noncitizen",
		"noncit-izen", "noncitizen",
		"Zac- arias", "Zacarias",
		"Zac-arias", "Zacarias",
	)
	return replacer.Replace(text)
}

func repairKnownJoinedWordArtifacts(text string) string {
	replacer := strings.NewReplacer(
		"socialgroup", "social group",
		"butconcluded", "but concluded",
		"applicationof", "application of",
		"concludingthat", "concluding that",
		"appropriatestandard", "appropriate standard",
		"substantial-evidencestandard", "substantial-evidence standard",
		"reviewof", "review of",
		"thatCongress", "that Congress",
		"receivedeference", "receive deference",
		"thejudgment", "the judgment",
		"de 6novo", "de novo",
	)
	return replacer.Replace(text)
}
