package mvp

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestEnterURLAndMakeOpinionID(t *testing.T) {
	opinionURL, err := EnterURL("https://www.supremecourt.gov/opinions/25pdf/24-777_9ol1.pdf")
	if err != nil {
		t.Fatalf("enter url: %v", err)
	}
	opinionID, err := MakeOpinionID(opinionURL)
	if err != nil {
		t.Fatalf("make opinion id: %v", err)
	}
	if opinionID == "" {
		t.Fatalf("expected non-empty opinion id")
	}
}

func TestFetchAndStorePDF(t *testing.T) {
	fixture := loadRealFixturePDF(t)
	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(fixture.Bytes)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	bytes, err := FetchPDF(context.Background(), client, URL("https://example.com/opinion.pdf"))
	if err != nil {
		t.Fatalf("fetch pdf: %v", err)
	}

	opinionID, _ := NewOpinionID("24-777")
	raw, err := MakeRawPDF(opinionID, URL("https://example.com/opinion.pdf"), bytes, time.Now())
	if err != nil {
		t.Fatalf("make raw pdf: %v", err)
	}

	storage := NewMemoryStorage()
	if _, err := StorePDF(storage, raw); err != nil {
		t.Fatalf("store pdf: %v", err)
	}
}
