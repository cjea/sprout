package mvp

import (
	"os"
	"path/filepath"
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
