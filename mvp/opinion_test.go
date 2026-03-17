package mvp

import (
	"errors"
	"testing"
)

func TestParseSectionKind(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  SectionKind
		err   error
	}{
		{name: "majority", input: "Majority", want: SectionKindMajority},
		{name: "syllabus", input: " syllabus ", want: SectionKindSyllabus},
		{name: "empty", input: "", err: ErrEmptySectionKind},
		{name: "unknown", input: "bench memo", err: ErrUnknownSectionKind},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSectionKind(tt.input)
			if !errors.Is(err, tt.err) {
				t.Fatalf("expected error %v, got %v", tt.err, err)
			}
			if tt.err == nil && got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewMeta(t *testing.T) {
	author := JusticeName("Roberts")
	meta, err := NewMeta("Trump v. CASA, Inc.", "24-777", "2026-03-16", "2025", &author)
	if err != nil {
		t.Fatalf("new meta: %v", err)
	}
	if meta.CaseName != "Trump v. CASA, Inc." {
		t.Fatalf("got case name %q", meta.CaseName)
	}
	if meta.PrimaryAuthor == nil || *meta.PrimaryAuthor != author {
		t.Fatalf("expected primary author to be preserved")
	}
}

func TestMetaValidation(t *testing.T) {
	tests := []struct {
		name string
		meta Meta
		err  error
	}{
		{name: "missing case name", meta: Meta{DocketNumber: "24-777", DecidedOn: "2026-03-16", TermLabel: "2025"}, err: ErrEmptyCaseName},
		{name: "missing docket", meta: Meta{CaseName: "Case", DecidedOn: "2026-03-16", TermLabel: "2025"}, err: ErrEmptyDocketNumber},
		{name: "missing decision date", meta: Meta{CaseName: "Case", DocketNumber: "24-777", TermLabel: "2025"}, err: ErrEmptyDecisionDate},
		{name: "missing term", meta: Meta{CaseName: "Case", DocketNumber: "24-777", DecidedOn: "2026-03-16"}, err: ErrEmptyTermLabel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.meta.Validate(), tt.err) {
				t.Fatalf("expected error %v", tt.err)
			}
		})
	}
}

func TestNewOpinion(t *testing.T) {
	opinionID, _ := NewOpinionID("24-777")
	sectionID, _ := NewSectionID("syllabus")
	meta, _ := NewMeta("Case", "24-777", "2026-03-16", "2025", nil)
	section, err := NewSection(opinionID, sectionID, SectionKindSyllabus, "Syllabus", nil, 1, 2, "Summary text")
	if err != nil {
		t.Fatalf("new section: %v", err)
	}

	opinion, err := NewOpinion(opinionID, meta, []Section{section}, "Summary text")
	if err != nil {
		t.Fatalf("new opinion: %v", err)
	}
	if len(opinion.Sections) != 1 {
		t.Fatalf("got %d sections, want 1", len(opinion.Sections))
	}
}

func TestOpinionValidation(t *testing.T) {
	opinionID, _ := NewOpinionID("24-777")
	otherOpinionID, _ := NewOpinionID("24-888")
	sectionID, _ := NewSectionID("majority")
	meta, _ := NewMeta("Case", "24-777", "2026-03-16", "2025", nil)

	goodSection, _ := NewSection(opinionID, sectionID, SectionKindMajority, "Majority", nil, 2, 3, "Majority text")
	earlySection, _ := NewSection(opinionID, sectionID, SectionKindSyllabus, "Syllabus", nil, 1, 1, "Syllabus text")
	mismatchSection, _ := NewSection(otherOpinionID, sectionID, SectionKindMajority, "Majority", nil, 2, 3, "Majority text")

	tests := []struct {
		name     string
		sections []Section
		fullText Text
		err      error
	}{
		{name: "empty full text", sections: []Section{goodSection}, fullText: "", err: ErrEmptyOpinionText},
		{name: "sections out of order", sections: []Section{goodSection, earlySection}, fullText: "text", err: ErrSectionsOutOfOrder},
		{name: "section mismatch", sections: []Section{mismatchSection}, fullText: "text", err: ErrSectionOpinionMismatch},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewOpinion(opinionID, meta, tt.sections, tt.fullText)
			if !errors.Is(err, tt.err) {
				t.Fatalf("expected error %v, got %v", tt.err, err)
			}
		})
	}
}

func TestOpinionSatisfiesStorageRecord(t *testing.T) {
	var record OpinionRecord = Opinion{}
	if record == nil {
		t.Fatalf("expected opinion to satisfy OpinionRecord")
	}
}
