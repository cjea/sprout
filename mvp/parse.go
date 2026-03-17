package mvp

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"

	pdf "github.com/ledongthuc/pdf"
)

var (
	fullCaptionPattern    = regexp.MustCompile(`([A-Z][A-Z\-'., ]+ v\. [A-Z][A-Z\-'., ]+)`)
	docketPattern         = regexp.MustCompile(`\bNo\.\s*([0-9]+(?:[-–][0-9]+)+)\b`)
	decisionPattern       = regexp.MustCompile(`Decided ([A-Za-z]+ \d{1,2}, \d{4})`)
	arguedDecisionPattern = regexp.MustCompile(`Argued [A-Za-z]+ \d{1,2}, \d{4}[—-]Decided ([A-Za-z]+ \d{1,2}, \d{4})`)
	concurrencePattern    = regexp.MustCompile(`(?i)^JUSTICE .* concurring`)
	dissentPattern        = regexp.MustCompile(`(?i)^JUSTICE .* dissenting`)
	majorityPattern       = regexp.MustCompile(`(?i)^JUSTICE .* delivered the opinion of the Court\.`)
	majorityInlinePattern = regexp.MustCompile(`JUSTICE [A-Z]+ delivered the opinion of the Court\.`)
)

type sectionStart struct {
	kind  SectionKind
	title string
	page  PageNo
	line  int
}

func ParsePDF(raw RawPDF) (ParsedPDF, error) {
	reader, err := pdf.NewReader(bytes.NewReader(raw.Bytes), int64(len(raw.Bytes)))
	if err != nil {
		return ParsedPDF{}, fmt.Errorf("open pdf reader: %w", err)
	}

	fontCache := map[string]*pdf.Font{}
	pages := make([]ParsedPage, 0, reader.NumPage())
	var fullText strings.Builder
	offset := Offset(0)

	for index := 1; index <= reader.NumPage(); index++ {
		page := reader.Page(index)
		for _, name := range page.Fonts() {
			if _, ok := fontCache[name]; !ok {
				font := page.Font(name)
				fontCache[name] = &font
			}
		}
		text, err := page.GetPlainText(fontCache)
		if err != nil {
			return ParsedPDF{}, fmt.Errorf("extract page %d text: %w", index, err)
		}
		clean := cleanExtractedText(text)
		if clean == "" {
			continue
		}

		if fullText.Len() > 0 {
			fullText.WriteString("\n\f\n")
		}
		fullText.WriteString(clean)

		endOffset := offset + Offset(len(clean))
		pages = append(pages, ParsedPage{
			PageNo: PageNo(index),
			Text:   Text(clean),
			Blocks: []TextBlock{{
				PageNo:      PageNo(index),
				StartOffset: offset,
				EndOffset:   endOffset,
				Text:        Text(clean),
			}},
		})
		offset = endOffset + 1
	}

	return NewParsedPDF(raw.OpinionID, pages, Text(fullText.String()), nil)
}

func ExtractMeta(parsed ParsedPDF) (Meta, error) {
	fullText := string(parsed.FullText)
	fullCaption := firstFullCaption(fullText)
	caseName := deriveCaseName(fullCaption)
	docket := normalizeDocket(firstMatch([]string{fullText}, docketPattern))
	decidedOn := firstDecisionDate([]string{fullText})
	termLabel := deriveTermLabel(decidedOn)

	return NewMeta(caseName, docket, decidedOn, termLabel, nil)
}

func GuessSections(model Model, parsed ParsedPDF) ([]Section, error) {
	guesser := HeuristicSectionGuesser{Model: model}
	return guesser.GuessSections(context.Background(), parsed)
}

func guessSectionsHeuristic(parsed ParsedPDF) ([]Section, error) {
	var starts []sectionStart
	for _, page := range parsed.Pages {
		for _, marker := range classifySectionText(string(page.Text)) {
			starts = append(starts, sectionStart{
				kind:  marker.kind,
				title: marker.title,
				page:  page.PageNo,
				line:  0,
			})
		}
	}
	starts = dedupeSectionPages(starts)
	if len(starts) == 0 {
		starts = append(starts, sectionStart{
			kind:  SectionKindUnknown,
			title: "Opinion",
			page:  1,
			line:  0,
		})
	}

	sections := make([]Section, 0, len(starts))
	for index, start := range starts {
		endPage := parsed.Pages[len(parsed.Pages)-1].PageNo
		if index+1 < len(starts) {
			endPage = starts[index+1].page - 1
			if endPage < start.page {
				endPage = start.page
			}
		}
		text := extractSectionText(parsed.Pages, start.page, endPage, start.line, nextStartLine(starts, index, endPage))
		sectionID, err := NewSectionID(sectionSlug(start.title, index))
		if err != nil {
			return nil, err
		}
		section, err := NewSection(parsed.OpinionID, sectionID, start.kind, start.title, guessJustice(start.title), start.page, endPage, Text(text))
		if err != nil {
			return nil, err
		}
		sections = append(sections, section)
	}
	return sections, nil
}

func BuildOpinion(opinionID OpinionID, meta Meta, sections []Section, parsed ParsedPDF) (Opinion, error) {
	return NewOpinion(opinionID, meta, sections, parsed.FullText)
}

func StoreOpinion(storage Storage, opinion Opinion) (Opinion, error) {
	return saveOpinion(storage, opinion)
}

func cleanExtractedText(text string) string {
	lines := strings.Split(text, "\n")
	clean := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		clean = append(clean, trimmed)
	}
	return strings.Join(clean, "\n")
}

func nonEmptyLines(text string) []string {
	rawLines := strings.Split(text, "\n")
	lines := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && trimmed != "\f" {
			lines = append(lines, trimmed)
		}
	}
	return lines
}

func firstFullCaption(text string) string {
	matches := fullCaptionPattern.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		candidate := strings.TrimSpace(match[1])
		upper := strings.ToUpper(candidate)
		if strings.Contains(upper, "SUPREME COURT") || strings.Contains(upper, "FIRST CIRCUIT") {
			continue
		}
		if index := strings.Index(upper, " SYLLABUS "); index >= 0 {
			upper = strings.TrimSpace(upper[:index])
		}
		if index := strings.Index(upper, " CERTIORARI "); index >= 0 {
			upper = strings.TrimSpace(upper[:index])
		}
		return upper
	}
	return ""
}

func deriveCaseName(fullCaption string) string {
	if fullCaption == "" {
		return ""
	}
	parts := strings.Split(fullCaption, ",")
	clean := strings.TrimSpace(parts[0])
	words := strings.Fields(clean)
	if len(words) > 0 {
		last := words[len(words)-1]
		if len(last) == 1 && strings.ToUpper(last) == last {
			clean = strings.TrimSpace(strings.TrimSuffix(clean, last))
		}
	}
	words = strings.Fields(strings.ToLower(clean))
	for index, word := range words {
		switch word {
		case "v.":
			words[index] = "v."
		case "et", "al.":
			words[index] = word
		default:
			segments := strings.Split(word, "-")
			for i, segment := range segments {
				if segment == "" {
					continue
				}
				segments[i] = strings.ToUpper(segment[:1]) + segment[1:]
			}
			words[index] = strings.Join(segments, "-")
		}
	}
	return strings.Join(words, " ")
}

func firstMatch(lines []string, pattern *regexp.Regexp) string {
	for _, line := range lines {
		matches := pattern.FindStringSubmatch(line)
		if len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
	}
	return ""
}

func normalizeDocket(value string) string {
	return strings.NewReplacer("–", "-", "—", "-").Replace(strings.TrimSpace(value))
}

func firstDecisionDate(lines []string) string {
	for _, line := range lines {
		if matches := arguedDecisionPattern.FindStringSubmatch(line); len(matches) > 1 {
			return matches[1]
		}
		if matches := decisionPattern.FindStringSubmatch(line); len(matches) > 1 {
			return matches[1]
		}
	}
	return ""
}

func deriveTermLabel(decidedOn string) string {
	fields := strings.Fields(decidedOn)
	if len(fields) == 0 {
		return ""
	}
	return fields[len(fields)-1]
}

func classifySectionLine(line string) (SectionKind, string, bool) {
	trimmed := strings.TrimSpace(line)
	switch {
	case strings.EqualFold(trimmed, "Syllabus"):
		return SectionKindSyllabus, "Syllabus", true
	case majorityPattern.MatchString(trimmed):
		return SectionKindMajority, trimmed, true
	case concurrencePattern.MatchString(trimmed):
		return SectionKindConcurrence, trimmed, true
	case dissentPattern.MatchString(trimmed):
		return SectionKindDissent, trimmed, true
	case strings.Contains(strings.ToUpper(trimmed), "APPENDIX"):
		return SectionKindAppendix, trimmed, true
	default:
		return "", "", false
	}
}

func classifySectionText(text string) []sectionStart {
	var starts []sectionStart
	if strings.Contains(text, "Syllabus") {
		starts = append(starts, sectionStart{kind: SectionKindSyllabus, title: "Syllabus"})
	}
	if match := majorityInlinePattern.FindString(text); match != "" {
		starts = append(starts, sectionStart{kind: SectionKindMajority, title: match})
	}
	for _, line := range nonEmptyLines(text) {
		if kind, title, ok := classifySectionLine(line); ok {
			if kind == SectionKindSyllabus || kind == SectionKindMajority {
				continue
			}
			starts = append(starts, sectionStart{kind: kind, title: title})
		}
	}
	return dedupeSectionStarts(starts)
}

func dedupeSectionStarts(starts []sectionStart) []sectionStart {
	seen := map[string]struct{}{}
	out := make([]sectionStart, 0, len(starts))
	for _, start := range starts {
		key := fmt.Sprintf("%s:%s", start.kind, start.title)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, start)
	}
	return out
}

func dedupeSectionPages(starts []sectionStart) []sectionStart {
	seen := map[string]struct{}{}
	out := make([]sectionStart, 0, len(starts))
	for _, start := range starts {
		key := fmt.Sprintf("%s:%s", start.kind, start.title)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, start)
	}
	return out
}

func extractSectionText(pages []ParsedPage, startPage, endPage PageNo, startLine, endLine int) string {
	var builder strings.Builder
	for _, page := range pages {
		if page.PageNo < startPage || page.PageNo > endPage {
			continue
		}
		lines := nonEmptyLines(string(page.Text))
		from := 0
		to := len(lines)
		if page.PageNo == startPage {
			from = startLine
		}
		if page.PageNo == endPage && endLine >= 0 && endLine < len(lines) {
			to = endLine
		}
		if from >= to {
			continue
		}
		if builder.Len() > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(strings.Join(lines[from:to], "\n"))
	}
	return strings.TrimSpace(builder.String())
}

func nextStartLine(starts []sectionStart, index int, endPage PageNo) int {
	if index+1 < len(starts) && starts[index+1].page == endPage+1 {
		return -1
	}
	if index+1 < len(starts) && starts[index+1].page == starts[index].page {
		return starts[index+1].line
	}
	return -1
}

func guessJustice(title string) *JusticeName {
	trimmed := strings.TrimSpace(title)
	if !strings.HasPrefix(trimmed, "JUSTICE ") {
		return nil
	}
	remainder := strings.TrimPrefix(trimmed, "JUSTICE ")
	name := strings.Fields(remainder)
	if len(name) == 0 {
		return nil
	}
	justice := JusticeName(strings.Trim(name[0], ",."))
	return &justice
}

func sectionSlug(title string, index int) string {
	slug := strings.ToLower(title)
	replacer := strings.NewReplacer(
		" ", "-",
		".", "",
		",", "",
		"'", "",
		"’", "",
	)
	slug = replacer.Replace(slug)
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return fmt.Sprintf("section-%d", index+1)
	}
	return slug
}
