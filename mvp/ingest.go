package mvp

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
)

func EnterURL(raw string) (URL, error) {
	value := URL(strings.TrimSpace(raw))
	if value == "" {
		return "", ErrEmptyURL
	}
	if _, err := url.ParseRequestURI(string(value)); err != nil {
		return "", fmt.Errorf("parse url: %w", err)
	}
	return value, nil
}

func MakeOpinionID(sourceURL URL) (OpinionID, error) {
	parsed, err := url.Parse(string(sourceURL))
	if err != nil {
		return "", fmt.Errorf("parse opinion url: %w", err)
	}

	base := strings.TrimSuffix(path.Base(parsed.Path), path.Ext(parsed.Path))
	base = strings.TrimSpace(base)
	if base == "" || base == "." || base == "/" {
		sum := sha1.Sum([]byte(sourceURL))
		base = hex.EncodeToString(sum[:8])
	}

	return NewOpinionID(base)
}

func FetchPDF(ctx context.Context, client *http.Client, sourceURL URL) (PDFBytes, error) {
	if client == nil {
		client = http.DefaultClient
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, string(sourceURL), nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("fetch pdf: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch pdf: unexpected status %d", response.StatusCode)
	}

	bytes, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read pdf response: %w", err)
	}
	if len(bytes) == 0 {
		return nil, ErrEmptyPDFBytes
	}
	return bytes, nil
}

func MakeRawPDF(opinionID OpinionID, sourceURL URL, bytes PDFBytes, fetchedAt Timestamp) (RawPDF, error) {
	return NewRawPDF(opinionID, sourceURL, bytes, fetchedAt)
}

func StorePDF(storage Storage, raw RawPDF) (RawPDF, error) {
	return saveRawPDF(storage, raw)
}
