package mvp

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fixtureExpectation struct {
	SourceURL    string `json:"source_url"`
	FileName     string `json:"file_name"`
	SHA256       string `json:"sha256"`
	CaseName     string `json:"case_name"`
	FullCaption  string `json:"full_caption"`
	DocketNumber string `json:"docket_no"`
	DecidedOn    string `json:"decided_on"`
	Term         string `json:"term"`
	Pages        int    `json:"pages"`
	Sections     []struct {
		Kind      string `json:"kind"`
		Heading   string `json:"heading"`
		StartPage int    `json:"start_page"`
		Justice   string `json:"justice,omitempty"`
		Evidence  string `json:"evidence"`
	} `json:"sections"`
	CitationPassages []struct {
		Excerpt   string   `json:"excerpt"`
		Citations []string `json:"citations"`
	} `json:"citation_passages"`
}

func TestRealSCOTUSFixtureLocked(t *testing.T) {
	var expect fixtureExpectation
	expectPath := filepath.Join("..", "fixtures", "scotus", "24-777_9ol1.expect.json")
	expectJSON, err := os.ReadFile(expectPath)
	if err != nil {
		t.Fatalf("read fixture expectation: %v", err)
	}
	if err := json.Unmarshal(expectJSON, &expect); err != nil {
		t.Fatalf("unmarshal fixture expectation: %v", err)
	}

	pdfPath := filepath.Join("..", "fixtures", "scotus", expect.FileName)
	bytes, err := os.ReadFile(pdfPath)
	if err != nil {
		t.Fatalf("read fixture pdf: %v", err)
	}

	sum := sha256.Sum256(bytes)
	gotHash := hex.EncodeToString(sum[:])
	if gotHash != expect.SHA256 {
		t.Fatalf("got fixture hash %q, want %q", gotHash, expect.SHA256)
	}

	excerptsPath := filepath.Join("..", "fixtures", "scotus", "24-777_9ol1.excerpts.txt")
	fixtureExcerpts, err := os.ReadFile(excerptsPath)
	if err != nil {
		t.Fatalf("read fixture excerpts: %v", err)
	}

	excerpts := normalizeFixtureText(string(fixtureExcerpts))
	mustContain(t, excerpts, expect.FullCaption)
	mustContain(t, excerpts, expect.DocketNumber)
	mustContain(t, excerpts, expect.DecidedOn)

	for _, section := range expect.Sections {
		mustContain(t, excerpts, section.Heading)
		mustContain(t, excerpts, section.Evidence)
		if section.Justice != "" {
			mustContain(t, excerpts, section.Justice)
		}
	}

	for _, passage := range expect.CitationPassages {
		mustContain(t, excerpts, passage.Excerpt)
		for _, citation := range passage.Citations {
			mustContain(t, excerpts, citation)
		}
	}
}

func normalizeFixtureText(text string) string {
	normalized := strings.NewReplacer(
		"–", "-",
		"—", "-",
		"“", "\"",
		"”", "\"",
		"’", "'",
	).Replace(text)
	return strings.Join(strings.Fields(normalized), " ")
}

func mustContain(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, normalizeFixtureText(needle)) {
		t.Fatalf("fixture text missing expected content: %q", needle)
	}
}
