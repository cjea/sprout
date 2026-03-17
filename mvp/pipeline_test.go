package mvp

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestChunkAttachRepairAndStorePassages(t *testing.T) {
	opinionID, _ := NewOpinionID("24-777")
	sectionID, _ := NewSectionID("majority")
	section, err := NewSection(
		opinionID,
		sectionID,
		SectionKindMajority,
		"Majority",
		nil,
		1,
		1,
		"The Court cites Roe v. Wade. It also cites 5 U.S.C. § 706. The standard matters.",
	)
	if err != nil {
		t.Fatalf("new section: %v", err)
	}

	passages, err := ChunkSections(DefaultChunkPolicy(), []Section{section})
	if err != nil {
		t.Fatalf("chunk sections: %v", err)
	}
	if len(passages) == 0 {
		t.Fatalf("expected at least one passage")
	}

	passages, err = AttachCitations(passages)
	if err != nil {
		t.Fatalf("attach citations: %v", err)
	}
	if len(passages[0].Citations) == 0 {
		t.Fatalf("expected citations to be attached")
	}

	repaired, err := RepairPassages(DefaultScreenPolicy(), passages)
	if err != nil {
		t.Fatalf("repair passages: %v", err)
	}
	if len(repaired) == 0 {
		t.Fatalf("expected repaired passages")
	}

	storage := NewMemoryStorage()
	saved, err := StorePassages(storage, repaired)
	if err != nil {
		t.Fatalf("store passages: %v", err)
	}
	if len(saved) != len(repaired) {
		t.Fatalf("got %d saved passages, want %d", len(saved), len(repaired))
	}
}

func TestSplitSentencesKeepsLegalAbbreviationsIntactOnRealFixtureExcerpt(t *testing.T) {
	bytes, err := os.ReadFile(filepath.Join("..", "fixtures", "scotus", "24-777_9ol1.excerpts.txt"))
	if err != nil {
		t.Fatalf("read fixture excerpts: %v", err)
	}
	text := string(bytes)
	start := strings.Index(text, "The Court has previously interpreted")
	end := strings.Index(text, "Substantial evidence, we have long emphasized")
	if start < 0 || end <= start {
		t.Fatalf("failed to locate legal citation excerpt in real fixture text")
	}
	excerpt := strings.TrimSpace(text[start:end])
	sentences := splitSentences(excerpt)
	if len(sentences) != 1 {
		t.Fatalf("expected one sentence, got %d: %#v", len(sentences), sentences)
	}
	if !strings.Contains(sentences[0], "Nasrallah v. Barr, 590 U. S. 573, 584") {
		t.Fatalf("expected case citation to remain intact")
	}
}

func TestChunkSectionsStripsRunningHeadsFromRealFixture(t *testing.T) {
	fixture := loadRealFixturePDF(t)

	opinionID, err := MakeOpinionID(fixture.SourceURL)
	if err != nil {
		t.Fatalf("make opinion id: %v", err)
	}
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

	headerPattern := regexp.MustCompile(`\bURIAS-ORELLANA v\. BONDI\b.*\b(?:Syllabus|Opinion of the Court)\b`)
	slipPattern := regexp.MustCompile(`\(Slip Opinion\)|OCTOBER TERM`)
	for _, passage := range passages {
		if headerPattern.MatchString(string(passage.Text)) {
			t.Fatalf("passage %s still contains running head: %q", passage.PassageID, passage.Text)
		}
		if slipPattern.MatchString(string(passage.Text)) {
			t.Fatalf("passage %s still contains slip header: %q", passage.PassageID, passage.Text)
		}
	}
}

func TestSplitSentencesAgainstQualityCorpusSentenceCases(t *testing.T) {
	corpus := loadQualityCorpus(t)

	for _, testCase := range corpus.Cases {
		if testCase.Kind != "sentence_boundary" {
			continue
		}

		t.Run(testCase.ID, func(t *testing.T) {
			got := splitSentences(testCase.Input)
			if len(got) != len(testCase.WantSentences) {
				t.Fatalf("got %d sentences, want %d: %#v", len(got), len(testCase.WantSentences), got)
			}
			for index := range testCase.WantSentences {
				if got[index] != testCase.WantSentences[index] {
					t.Fatalf("sentence %d got %q want %q", index, got[index], testCase.WantSentences[index])
				}
			}
		})
	}
}

func TestCleanPassageSourceTextAgainstQualityCorpusCleanupCases(t *testing.T) {
	corpus := loadQualityCorpus(t)

	for _, testCase := range corpus.Cases {
		if testCase.Kind != "text_cleanup" {
			continue
		}

		t.Run(testCase.ID, func(t *testing.T) {
			got := cleanPassageSourceText(testCase.Input)
			if got != testCase.WantNormalized {
				t.Fatalf("got %q want %q", got, testCase.WantNormalized)
			}
		})
	}
}
