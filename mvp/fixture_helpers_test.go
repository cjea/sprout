package mvp

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func loadRealFixturePDF(t *testing.T) RawPDF {
	t.Helper()
	path := filepath.Join("..", "fixtures", "scotus", "24-777_9ol1.pdf")
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read real fixture pdf: %v", err)
	}
	opinionID, err := NewOpinionID("24-777_9ol1")
	if err != nil {
		t.Fatalf("new opinion id: %v", err)
	}
	raw, err := NewRawPDF(opinionID, URL("https://www.supremecourt.gov/opinions/25pdf/24-777_9ol1.pdf"), bytes, time.Date(2026, time.March, 17, 1, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("new raw pdf: %v", err)
	}
	return raw
}
