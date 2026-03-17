package mvp

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
)

var (
	ErrEmptyPDFBytes       = errors.New("pdf bytes are required")
	ErrEmptyFullText       = errors.New("full text is required")
	ErrPagesOutOfOrder     = errors.New("pages must be in ascending order")
	ErrBlocksOutOfOrder    = errors.New("blocks must be in ascending order")
	ErrBlockPageMismatch   = errors.New("block page must match parsed page")
	ErrInvalidBlockOffsets = errors.New("text block offsets are invalid")
)

type RawPDF struct {
	OpinionID OpinionID
	SourceURL URL
	Bytes     PDFBytes
	FetchedAt Timestamp
	SHA256    string
}

func NewRawPDF(opinionID OpinionID, sourceURL URL, bytes PDFBytes, fetchedAt Timestamp) (RawPDF, error) {
	raw := RawPDF{
		OpinionID: opinionID,
		SourceURL: sourceURL,
		Bytes:     append(PDFBytes(nil), bytes...),
		FetchedAt: fetchedAt,
		SHA256:    hashPDFBytes(bytes),
	}
	if err := raw.Validate(); err != nil {
		return RawPDF{}, err
	}
	return raw, nil
}

func (r RawPDF) Validate() error {
	if strings.TrimSpace(string(r.OpinionID)) == "" {
		return ErrEmptyOpinionID
	}
	if strings.TrimSpace(string(r.SourceURL)) == "" {
		return ErrEmptyURL
	}
	if len(r.Bytes) == 0 {
		return ErrEmptyPDFBytes
	}
	return nil
}

func (RawPDF) RawPDFRecord() {}

type ParsedPDF struct {
	OpinionID OpinionID
	Pages     []ParsedPage
	FullText  Text
	Warnings  []ParseWarning
}

func NewParsedPDF(opinionID OpinionID, pages []ParsedPage, fullText Text, warnings []ParseWarning) (ParsedPDF, error) {
	parsed := ParsedPDF{
		OpinionID: opinionID,
		Pages:     append([]ParsedPage(nil), pages...),
		FullText:  fullText,
		Warnings:  append([]ParseWarning(nil), warnings...),
	}
	if err := parsed.Validate(); err != nil {
		return ParsedPDF{}, err
	}
	return parsed, nil
}

func (p ParsedPDF) Validate() error {
	if strings.TrimSpace(string(p.OpinionID)) == "" {
		return ErrEmptyOpinionID
	}
	if strings.TrimSpace(string(p.FullText)) == "" {
		return ErrEmptyFullText
	}

	lastPage := PageNo(0)
	for index, page := range p.Pages {
		if index > 0 && page.PageNo <= lastPage {
			return ErrPagesOutOfOrder
		}
		if err := page.Validate(); err != nil {
			return err
		}
		lastPage = page.PageNo
	}

	return nil
}

type ParsedPage struct {
	PageNo PageNo
	Text   Text
	Blocks []TextBlock
}

func (p ParsedPage) Validate() error {
	lastEnd := Offset(-1)
	for _, block := range p.Blocks {
		if block.PageNo != p.PageNo {
			return ErrBlockPageMismatch
		}
		if err := block.Validate(); err != nil {
			return err
		}
		if block.StartOffset < lastEnd {
			return ErrBlocksOutOfOrder
		}
		lastEnd = block.EndOffset
	}
	return nil
}

type TextBlock struct {
	PageNo      PageNo
	StartOffset Offset
	EndOffset   Offset
	Text        Text
}

func (b TextBlock) Validate() error {
	if b.StartOffset > b.EndOffset {
		return ErrInvalidBlockOffsets
	}
	return nil
}

type ParseWarning struct {
	Code    string
	Message string
}

func hashPDFBytes(bytes PDFBytes) string {
	sum := sha256.Sum256(bytes)
	return hex.EncodeToString(sum[:])
}
