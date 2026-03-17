package mvp

import (
	"context"
	"testing"
)

func TestParseExtractMetaAndGuessSectionsFromRealFixture(t *testing.T) {
	raw := loadRealFixturePDF(t)

	parsed, err := ParsePDF(raw)
	if err != nil {
		t.Fatalf("parse pdf: %v", err)
	}
	if len(parsed.Pages) < 10 {
		t.Fatalf("expected double-digit page count, got %d", len(parsed.Pages))
	}

	meta, err := ExtractMeta(parsed)
	if err != nil {
		t.Fatalf("extract meta: %v", err)
	}
	if meta.CaseName != "Urias-Orellana v. Bondi" {
		t.Fatalf("got case name %q", meta.CaseName)
	}
	if meta.DocketNumber != "24-777" {
		t.Fatalf("got docket number %q", meta.DocketNumber)
	}
	if meta.DecidedOn != "March 4, 2026" {
		t.Fatalf("got decided date %q", meta.DecidedOn)
	}

	model, _ := NewModel("gpt-5", 8000)
	sections, err := GuessSections(model, parsed)
	if err != nil {
		t.Fatalf("guess sections: %v", err)
	}
	if len(sections) < 2 {
		t.Fatalf("expected at least syllabus and majority, got %d sections", len(sections))
	}
	if sections[0].Kind != SectionKindSyllabus {
		t.Fatalf("expected first section to be syllabus, got %q", sections[0].Kind)
	}

	opinion, err := BuildOpinion(raw.OpinionID, meta, sections, parsed)
	if err != nil {
		t.Fatalf("build opinion: %v", err)
	}
	storage := NewMemoryStorage()
	if _, err := StoreOpinion(storage, opinion); err != nil {
		t.Fatalf("store opinion: %v", err)
	}
}

func TestHeuristicGuessersUseInterfaces(t *testing.T) {
	model, _ := NewModel("gpt-5", 8000)
	raw := loadRealFixturePDF(t)
	parsed, err := ParsePDF(raw)
	if err != nil {
		t.Fatalf("parse pdf: %v", err)
	}

	var sectionGuesser SectionGuesser = HeuristicSectionGuesser{Model: model}
	sections, err := sectionGuesser.GuessSections(context.Background(), parsed)
	if err != nil {
		t.Fatalf("guesser sections: %v", err)
	}
	if len(sections) == 0 {
		t.Fatalf("expected heuristic section guesser to return sections")
	}
}
