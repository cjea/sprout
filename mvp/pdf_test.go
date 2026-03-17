package mvp

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"
)

func TestNewRawPDF(t *testing.T) {
	opinionID, err := NewOpinionID("24-777")
	if err != nil {
		t.Fatalf("new opinion id: %v", err)
	}

	fetchedAt := time.Date(2026, time.March, 16, 23, 0, 0, 0, time.UTC)
	bytes := PDFBytes("%PDF-1.7 example")

	raw, err := NewRawPDF(opinionID, URL("https://www.supremecourt.gov/opinions/25pdf/24-777_9ol1.pdf"), bytes, fetchedAt)
	if err != nil {
		t.Fatalf("new raw pdf: %v", err)
	}

	if raw.OpinionID != opinionID {
		t.Fatalf("got opinion id %q, want %q", raw.OpinionID, opinionID)
	}
	if raw.FetchedAt != fetchedAt {
		t.Fatalf("got fetched at %v, want %v", raw.FetchedAt, fetchedAt)
	}
	if string(raw.Bytes) != string(bytes) {
		t.Fatalf("got bytes %q, want %q", raw.Bytes, bytes)
	}

	sum := sha256.Sum256(bytes)
	wantHash := hex.EncodeToString(sum[:])
	if raw.SHA256 != wantHash {
		t.Fatalf("got hash %q, want %q", raw.SHA256, wantHash)
	}

	bytes[0] = 'X'
	if raw.Bytes[0] != '%' {
		t.Fatalf("expected raw pdf bytes to be copied defensively")
	}
}

func TestNewRawPDFValidation(t *testing.T) {
	validOpinionID, err := NewOpinionID("24-777")
	if err != nil {
		t.Fatalf("new opinion id: %v", err)
	}

	tests := []struct {
		name      string
		opinionID OpinionID
		sourceURL URL
		bytes     PDFBytes
		wantErr   error
	}{
		{
			name:      "empty opinion id",
			opinionID: OpinionID(""),
			sourceURL: URL("https://www.supremecourt.gov/opinions/25pdf/24-777_9ol1.pdf"),
			bytes:     PDFBytes("pdf"),
			wantErr:   ErrEmptyOpinionID,
		},
		{
			name:      "empty url",
			opinionID: validOpinionID,
			sourceURL: URL(""),
			bytes:     PDFBytes("pdf"),
			wantErr:   ErrEmptyURL,
		},
		{
			name:      "empty bytes",
			opinionID: validOpinionID,
			sourceURL: URL("https://www.supremecourt.gov/opinions/25pdf/24-777_9ol1.pdf"),
			bytes:     nil,
			wantErr:   ErrEmptyPDFBytes,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewRawPDF(tt.opinionID, tt.sourceURL, tt.bytes, time.Time{})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestNewParsedPDF(t *testing.T) {
	opinionID, err := NewOpinionID("24-777")
	if err != nil {
		t.Fatalf("new opinion id: %v", err)
	}

	pages := []ParsedPage{
		{
			PageNo: 1,
			Text:   "Syllabus text",
			Blocks: []TextBlock{
				{PageNo: 1, StartOffset: 0, EndOffset: 8, Text: "Syllabus"},
				{PageNo: 1, StartOffset: 9, EndOffset: 13, Text: "text"},
			},
		},
		{
			PageNo: 2,
			Text:   "Majority text",
			Blocks: []TextBlock{
				{PageNo: 2, StartOffset: 0, EndOffset: 8, Text: "Majority"},
			},
		},
	}

	warnings := []ParseWarning{{Code: "ocr-gap", Message: "missing footer text"}}
	parsed, err := NewParsedPDF(opinionID, pages, Text("Syllabus text\nMajority text"), warnings)
	if err != nil {
		t.Fatalf("new parsed pdf: %v", err)
	}

	if len(parsed.Pages) != len(pages) {
		t.Fatalf("got %d pages, want %d", len(parsed.Pages), len(pages))
	}
	if len(parsed.Warnings) != len(warnings) {
		t.Fatalf("got %d warnings, want %d", len(parsed.Warnings), len(warnings))
	}

	pages[0].Text = "changed"
	warnings[0].Message = "changed"
	if parsed.Pages[0].Text != "Syllabus text" {
		t.Fatalf("expected parsed pages slice to be copied")
	}
	if parsed.Warnings[0].Message != "missing footer text" {
		t.Fatalf("expected parsed warnings slice to be copied")
	}
}

func TestNewParsedPDFValidation(t *testing.T) {
	opinionID, err := NewOpinionID("24-777")
	if err != nil {
		t.Fatalf("new opinion id: %v", err)
	}

	tests := []struct {
		name     string
		opinion  OpinionID
		pages    []ParsedPage
		fullText Text
		wantErr  error
	}{
		{
			name:     "empty opinion id",
			opinion:  OpinionID(""),
			pages:    []ParsedPage{{PageNo: 1, Blocks: []TextBlock{{PageNo: 1, StartOffset: 0, EndOffset: 1}}}},
			fullText: Text("text"),
			wantErr:  ErrEmptyOpinionID,
		},
		{
			name:     "empty full text",
			opinion:  opinionID,
			pages:    []ParsedPage{{PageNo: 1, Blocks: []TextBlock{{PageNo: 1, StartOffset: 0, EndOffset: 1}}}},
			fullText: Text(""),
			wantErr:  ErrEmptyFullText,
		},
		{
			name:    "pages out of order",
			opinion: opinionID,
			pages: []ParsedPage{
				{PageNo: 2, Blocks: []TextBlock{{PageNo: 2, StartOffset: 0, EndOffset: 1}}},
				{PageNo: 1, Blocks: []TextBlock{{PageNo: 1, StartOffset: 0, EndOffset: 1}}},
			},
			fullText: Text("text"),
			wantErr:  ErrPagesOutOfOrder,
		},
		{
			name:    "block page mismatch",
			opinion: opinionID,
			pages: []ParsedPage{
				{PageNo: 1, Blocks: []TextBlock{{PageNo: 2, StartOffset: 0, EndOffset: 1}}},
			},
			fullText: Text("text"),
			wantErr:  ErrBlockPageMismatch,
		},
		{
			name:    "blocks out of order",
			opinion: opinionID,
			pages: []ParsedPage{
				{
					PageNo: 1,
					Blocks: []TextBlock{
						{PageNo: 1, StartOffset: 10, EndOffset: 20},
						{PageNo: 1, StartOffset: 5, EndOffset: 9},
					},
				},
			},
			fullText: Text("text"),
			wantErr:  ErrBlocksOutOfOrder,
		},
		{
			name:    "invalid block offsets",
			opinion: opinionID,
			pages: []ParsedPage{
				{PageNo: 1, Blocks: []TextBlock{{PageNo: 1, StartOffset: 2, EndOffset: 1}}},
			},
			fullText: Text("text"),
			wantErr:  ErrInvalidBlockOffsets,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewParsedPDF(tt.opinion, tt.pages, tt.fullText, nil)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestRawPDFSatisfiesStorageRecord(t *testing.T) {
	var record RawPDFRecord = RawPDF{}
	if record == nil {
		t.Fatalf("expected raw pdf to satisfy RawPDFRecord")
	}
}
