package mvp

import (
	"context"
	"strings"
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

func TestParsePDFDoesNotIntroduceJoinedWordArtifactsOnRealFixture(t *testing.T) {
	raw := loadRealFixturePDF(t)

	parsed, err := ParsePDF(raw)
	if err != nil {
		t.Fatalf("parse pdf: %v", err)
	}

	banned := []string{
		"socialgroup",
		"butconcluded",
		"applicationof",
		"concludingthat",
		"appropriatestandard",
		"substantial-evidencestandard",
		"reviewof",
		"thatCongress",
		"receivedeference",
		"thejudgment",
		"de 6novo",
	}

	text := string(parsed.FullText)
	for _, needle := range banned {
		if strings.Contains(text, needle) {
			t.Fatalf("parsed full text still contains joined-word artifact %q", needle)
		}
	}
}

func TestParsePDFDoesNotIntroduceHyphenationArtifactsOnRealFixture(t *testing.T) {
	raw := loadRealFixturePDF(t)

	parsed, err := ParsePDF(raw)
	if err != nil {
		t.Fatalf("parse pdf: %v", err)
	}

	banned := []string{
		"asy-lum",
		"refu-gee",
		"ac-count",
		"per-secution",
		"pre-scribe",
		"rea-sonable",
		"de-termination",
		"con-stitute",
		"or-dered",
		"ei-ther",
		"underly-ing",
		"signifi-cant",
		"subpar-agraph",
		"partic-ular",
		"pri-marily",
		"re-view",
		"noncit- izen",
		"noncit-izen",
		"Zac- arias",
		"Zac-arias",
	}
	allowed := []string{
		"substantial-evidence",
		"well-founded",
	}

	text := string(parsed.FullText)
	for _, needle := range banned {
		if strings.Contains(text, needle) {
			t.Fatalf("parsed full text still contains hyphenation artifact %q", needle)
		}
	}
	for _, needle := range allowed {
		if !strings.Contains(text, needle) {
			t.Fatalf("parsed full text should preserve legitimate compound %q", needle)
		}
	}
}

func TestRepairLineBreakHyphenationLines(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "repairs soft hyphen break",
			input: []string{"petitioners applied for asy-", "lum under the Act."},
			want:  []string{"petitioners applied for asylum under the Act."},
		},
		{
			name:  "repairs spaced soft hyphen break",
			input: []string{"the applicant showed per -", "secution in El Salvador."},
			want:  []string{"the applicant showed persecution in El Salvador."},
		},
		{
			name:  "preserves substantial evidence compound",
			input: []string{"the court applied a substantial-", "evidence standard."},
			want:  []string{"the court applied a substantial-evidence standard."},
		},
		{
			name:  "preserves well founded compound",
			input: []string{"a well-", "founded fear remained."},
			want:  []string{"a well-founded fear remained."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := repairLineBreakHyphenationLines(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d lines want %d: %#v", len(got), len(tt.want), got)
			}
			for index := range tt.want {
				if got[index] != tt.want[index] {
					t.Fatalf("line %d got %q want %q", index, got[index], tt.want[index])
				}
			}
		})
	}
}
