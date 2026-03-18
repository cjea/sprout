package mvp

import "testing"

func TestParsePassageIssueKind(t *testing.T) {
	kind, err := ParsePassageIssueKind("citation_detached")
	if err != nil {
		t.Fatalf("parse issue kind: %v", err)
	}
	if kind != PassageIssueCitationDetached {
		t.Fatalf("got kind %q", kind)
	}
}

func TestClassifyPassageIssuesForDetachedRealFixtureFragments(t *testing.T) {
	opinionID, _ := NewOpinionID("24-777_9ol1")
	sectionID, _ := NewSectionID("syllabus")
	passageID, _ := NewPassageID("p-1")
	nextID, _ := NewPassageID("p-2")

	passage, err := buildPassageFromSourceText(passageID, opinionID, sectionID, 0, 0, 1, 1, "Held: Courts should review the Board's determination that a given set of facts does not rise to the level of persecution under a substantial-evidence standard.", true)
	if err != nil {
		t.Fatalf("build passage: %v", err)
	}
	next, err := buildPassageFromSourceText(nextID, opinionID, sectionID, 1, 1, 1, 1, "Pp. 3-10.", true)
	if err != nil {
		t.Fatalf("build next passage: %v", err)
	}

	issues, err := ClassifyPassageIssues(passage, nil, &next)
	if err != nil {
		t.Fatalf("classify issues: %v", err)
	}
	if !hasIssue(issues, PassageIssuePageReferenceDetached) {
		t.Fatalf("expected page reference issue, got %#v", issues)
	}

	citationNext, err := buildPassageFromSourceText(nextID, opinionID, sectionID, 1, 1, 1, 1, "8 U. S. C. §1158(b)(1)(A).", true)
	if err != nil {
		t.Fatalf("build citation next passage: %v", err)
	}
	issues, err = ClassifyPassageIssues(passage, nil, &citationNext)
	if err != nil {
		t.Fatalf("classify issues: %v", err)
	}
	if !hasIssue(issues, PassageIssueCitationDetached) {
		t.Fatalf("expected citation issue, got %#v", issues)
	}
}

func TestClassifyPassageIssuesForSyntheticArtifacts(t *testing.T) {
	opinionID, _ := NewOpinionID("24-777_9ol1")
	sectionID, _ := NewSectionID("syllabus")
	passageID, _ := NewPassageID("p-1")

	hyphenated, err := NewPassage(passageID, opinionID, sectionID, 0, 0, 1, 1, "The statute permits asy-lum when the applicant shows persecution.", nil, true)
	if err != nil {
		t.Fatalf("new hyphenated passage: %v", err)
	}
	issues, err := ClassifyPassageIssues(hyphenated, nil, nil)
	if err != nil {
		t.Fatalf("classify hyphenated passage: %v", err)
	}
	if !hasIssue(issues, PassageIssueHyphenationArtifact) {
		t.Fatalf("expected hyphenation issue, got %#v", issues)
	}

	joined, err := NewPassage(passageID, opinionID, sectionID, 0, 0, 1, 1, "The statute requires applicationof the standard.", nil, true)
	if err != nil {
		t.Fatalf("new joined-word passage: %v", err)
	}
	issues, err = ClassifyPassageIssues(joined, nil, nil)
	if err != nil {
		t.Fatalf("classify joined-word passage: %v", err)
	}
	if !hasIssue(issues, PassageIssueJoinedWordArtifact) {
		t.Fatalf("expected joined-word issue, got %#v", issues)
	}
}

func TestClassifyPassageIssuesLeavesCleanFixturePassagesAlone(t *testing.T) {
	fixture := loadRealFixturePDF(t)
	opinionID, _ := MakeOpinionID(fixture.SourceURL)
	raw, err := MakeRawPDF(opinionID, fixture.SourceURL, fixture.Bytes, fixture.FetchedAt)
	if err != nil {
		t.Fatalf("make raw pdf: %v", err)
	}
	parsed, err := ParsePDF(raw)
	if err != nil {
		t.Fatalf("parse pdf: %v", err)
	}
	sections, err := GuessSections(Model{Name: "heuristic-v1", MaxContextTokens: 8192}, parsed)
	if err != nil {
		t.Fatalf("guess sections: %v", err)
	}
	passages, err := ChunkSections(DefaultChunkPolicy(), sections)
	if err != nil {
		t.Fatalf("chunk sections: %v", err)
	}

	for index, passage := range passages {
		var previous *Passage
		var next *Passage
		if index > 0 {
			previous = &passages[index-1]
		}
		if index+1 < len(passages) {
			next = &passages[index+1]
		}
		issues, err := ClassifyPassageIssues(passage, previous, next)
		if err != nil {
			t.Fatalf("classify clean passage: %v", err)
		}
		if hasIssue(issues, PassageIssueHyphenationArtifact) || hasIssue(issues, PassageIssueJoinedWordArtifact) || hasIssue(issues, PassageIssuePageHeaderArtifact) {
			t.Fatalf("clean fixture passage should not contain extraction artifact issues: %#v", issues)
		}
	}
}

func hasIssue(issues []PassageIssue, kind PassageIssueKind) bool {
	for _, issue := range issues {
		if issue.Kind == kind {
			return true
		}
	}
	return false
}
