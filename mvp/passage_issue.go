package mvp

import (
	"errors"
	"strings"
)

var (
	ErrUnknownPassageIssueKind = errors.New("unknown passage issue kind")
)

var (
	joinedWordArtifactWords = []string{
		"applicationof",
		"thatCongress",
		"receivedeference",
		"socialgroup",
		"butconcluded",
	}
)

type PassageIssueKind string

const (
	PassageIssuePageHeaderArtifact    PassageIssueKind = "page_header_artifact"
	PassageIssueHyphenationArtifact   PassageIssueKind = "hyphenation_artifact"
	PassageIssueJoinedWordArtifact    PassageIssueKind = "joined_word_artifact"
	PassageIssueBadSentenceBoundary   PassageIssueKind = "bad_sentence_boundary"
	PassageIssueCitationDetached      PassageIssueKind = "citation_detached"
	PassageIssuePageReferenceDetached PassageIssueKind = "page_reference_detached"
	PassageIssueAcronymSplit          PassageIssueKind = "acronym_split"
	PassageIssuePinCiteSplit          PassageIssueKind = "pin_cite_split"
	PassageIssuePassageTooLong        PassageIssueKind = "passage_too_long"
	PassageIssuePassageTooShort       PassageIssueKind = "passage_too_short"
	PassageIssueDuplicatePassage      PassageIssueKind = "duplicate_passage"
	PassageIssueOrphanFragment        PassageIssueKind = "orphan_fragment"
	PassageIssueUnknown               PassageIssueKind = "unknown_issue"
)

func ParsePassageIssueKind(value string) (PassageIssueKind, error) {
	switch PassageIssueKind(strings.TrimSpace(value)) {
	case PassageIssuePageHeaderArtifact,
		PassageIssueHyphenationArtifact,
		PassageIssueJoinedWordArtifact,
		PassageIssueBadSentenceBoundary,
		PassageIssueCitationDetached,
		PassageIssuePageReferenceDetached,
		PassageIssueAcronymSplit,
		PassageIssuePinCiteSplit,
		PassageIssuePassageTooLong,
		PassageIssuePassageTooShort,
		PassageIssueDuplicatePassage,
		PassageIssueOrphanFragment,
		PassageIssueUnknown:
		return PassageIssueKind(strings.TrimSpace(value)), nil
	default:
		return "", ErrUnknownPassageIssueKind
	}
}

type PassageIssue struct {
	Kind      PassageIssueKind
	PassageID PassageID
	Summary   string
}

func NewPassageIssue(kind PassageIssueKind, passageID PassageID, summary string) (PassageIssue, error) {
	if _, err := ParsePassageIssueKind(string(kind)); err != nil {
		return PassageIssue{}, err
	}
	if strings.TrimSpace(string(passageID)) == "" {
		return PassageIssue{}, ErrEmptyPassageID
	}
	return PassageIssue{
		Kind:      kind,
		PassageID: passageID,
		Summary:   strings.TrimSpace(summary),
	}, nil
}

func ClassifyPassageIssues(passage Passage, previous *Passage, next *Passage) ([]PassageIssue, error) {
	if err := passage.Validate(); err != nil {
		return nil, err
	}

	issues := make([]PassageIssue, 0)
	text := string(passage.Text)

	add := func(kind PassageIssueKind, summary string) error {
		issue, err := NewPassageIssue(kind, passage.PassageID, summary)
		if err != nil {
			return err
		}
		issues = append(issues, issue)
		return nil
	}

	if headerPattern := runningHeadPattern; headerPattern.MatchString(text) || strings.Contains(text, "(Slip Opinion)") {
		if err := add(PassageIssuePageHeaderArtifact, "passage contains running header text"); err != nil {
			return nil, err
		}
	}
	if containsKnownHyphenationArtifact(text) {
		if err := add(PassageIssueHyphenationArtifact, "passage contains an extraction hyphenation artifact"); err != nil {
			return nil, err
		}
	}
	if containsJoinedWordArtifact(text) {
		if err := add(PassageIssueJoinedWordArtifact, "passage contains a joined-word extraction artifact"); err != nil {
			return nil, err
		}
	}
	if suspicious := DetectSuspiciousSentenceBoundaries([]string{text}); len(suspicious) > 0 {
		if err := add(PassageIssueBadSentenceBoundary, "passage contains a suspicious internal sentence boundary"); err != nil {
			return nil, err
		}
	}
	if next != nil {
		nextText := strings.TrimSpace(string(next.Text))
		if strings.HasPrefix(nextText, "Pp.") || pageRangeFragmentPattern.MatchString(nextText) {
			if err := add(PassageIssuePageReferenceDetached, "next passage begins with a page-reference fragment"); err != nil {
				return nil, err
			}
		} else if isCitationContinuation(nextText) {
			if err := add(PassageIssueCitationDetached, "next passage begins with a trailing authority cite"); err != nil {
				return nil, err
			}
		}
		if strings.EqualFold(nextText, "Id.") || pinCiteFragmentPattern.MatchString(nextText) {
			if err := add(PassageIssuePinCiteSplit, "next passage begins with a pin cite fragment"); err != nil {
				return nil, err
			}
		}
	}
	if previous != nil {
		previousText := strings.TrimSpace(string(previous.Text))
		if strings.HasSuffix(previousText, "U. S.") || strings.HasSuffix(previousText, "U.S.") {
			if err := add(PassageIssueAcronymSplit, "previous passage appears to end inside an acronym"); err != nil {
				return nil, err
			}
		}
	}
	if len(text) > DefaultScreenPolicy().MaxCharacters {
		if err := add(PassageIssuePassageTooLong, "passage exceeds screen-fit character policy"); err != nil {
			return nil, err
		}
	}
	if len(splitSentences(text)) == 1 && len(text) < 20 {
		if err := add(PassageIssuePassageTooShort, "passage is likely an orphan fragment"); err != nil {
			return nil, err
		}
	}
	return issues, nil
}

func containsKnownHyphenationArtifact(text string) bool {
	needles := []string{
		"asy-lum",
		"per-secution",
		"pre-scribe",
		"con-stitute",
		"re-view",
	}
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}

func containsJoinedWordArtifact(text string) bool {
	for _, needle := range joinedWordArtifactWords {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}
