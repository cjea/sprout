package mvp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type qualityCorpus struct {
	FileName string              `json:"fileName"`
	Cases    []qualityCorpusCase `json:"cases"`
}

type qualityCorpusCase struct {
	ID             string   `json:"id"`
	Kind           string   `json:"kind"`
	Source         string   `json:"source"`
	Input          string   `json:"input"`
	WantSentences  []string `json:"wantSentences"`
	WantNormalized string   `json:"wantNormalized"`
}

func TestQualityCorpusFixtureIsWellFormed(t *testing.T) {
	corpus := loadQualityCorpus(t)
	if corpus.FileName != "24-777_9ol1.pdf" {
		t.Fatalf("got file name %q", corpus.FileName)
	}
	if len(corpus.Cases) == 0 {
		t.Fatal("expected quality cases")
	}

	for _, testCase := range corpus.Cases {
		if testCase.ID == "" {
			t.Fatal("quality case id is required")
		}
		if testCase.Kind == "" {
			t.Fatalf("quality case %q kind is required", testCase.ID)
		}
		if testCase.Source == "" {
			t.Fatalf("quality case %q source is required", testCase.ID)
		}
		if testCase.Input == "" {
			t.Fatalf("quality case %q input is required", testCase.ID)
		}

		switch testCase.Kind {
		case "sentence_boundary":
			if len(testCase.WantSentences) == 0 {
				t.Fatalf("quality case %q requires wantSentences", testCase.ID)
			}
			if testCase.WantNormalized != "" {
				t.Fatalf("quality case %q should not mix sentence and normalization expectations", testCase.ID)
			}
		case "text_cleanup":
			if testCase.WantNormalized == "" {
				t.Fatalf("quality case %q requires wantNormalized", testCase.ID)
			}
			if len(testCase.WantSentences) != 0 {
				t.Fatalf("quality case %q should not mix cleanup and sentence expectations", testCase.ID)
			}
		default:
			t.Fatalf("quality case %q has unknown kind %q", testCase.ID, testCase.Kind)
		}
	}
}

func TestQualityCorpusIncludesFixtureAnchoredPassageCases(t *testing.T) {
	corpus := loadQualityCorpus(t)

	required := map[string]bool{
		"real-held-pp":         false,
		"real-nasrallah-usc":   false,
		"synthetic-id-at":      false,
		"synthetic-usc-trailing-cite": false,
		"synthetic-page-header-artifact": false,
		"synthetic-linebreak-hyphenation": false,
	}

	for _, testCase := range corpus.Cases {
		if _, ok := required[testCase.ID]; ok {
			required[testCase.ID] = true
		}
	}

	for id, found := range required {
		if !found {
			t.Fatalf("missing required quality case %q", id)
		}
	}
}

func loadQualityCorpus(t *testing.T) qualityCorpus {
	t.Helper()

	path := filepath.Join("..", "fixtures", "scotus", "24-777_9ol1.quality.json")
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read quality corpus: %v", err)
	}

	var corpus qualityCorpus
	if err := json.Unmarshal(bytes, &corpus); err != nil {
		t.Fatalf("unmarshal quality corpus: %v", err)
	}
	return corpus
}
