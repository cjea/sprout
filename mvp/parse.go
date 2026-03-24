package mvp

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"sort"
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
	trailingHyphenPattern = regexp.MustCompile(`([A-Za-z]{1,30})\s*-\s*$`)
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
		text, err := extractPageText(page, fontCache)
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

func extractPageText(page pdf.Page, fontCache map[string]*pdf.Font) (string, error) {
	texts := page.Content().Text
	if len(texts) > 0 {
		sort.Slice(texts, func(i, j int) bool {
			if texts[i].Y != texts[j].Y {
				return texts[i].Y > texts[j].Y
			}
			return texts[i].X < texts[j].X
		})

		var (
			builder strings.Builder
			line    []pdf.Text
			lineY   float64
		)
		flushLine := func() {
			if len(line) == 0 {
				return
			}
			if builder.Len() > 0 {
				builder.WriteByte('\n')
			}
			builder.WriteString(reconstructPageLine(line))
			line = nil
		}

		for _, text := range texts {
			if text.S == "\n" {
				continue
			}
			if len(line) == 0 {
				lineY = text.Y
			}
			if len(line) > 0 && text.Y != lineY {
				flushLine()
				lineY = text.Y
			}
			line = append(line, text)
		}
		flushLine()
		return builder.String(), nil
	}

	for _, name := range page.Fonts() {
		if _, ok := fontCache[name]; !ok {
			font := page.Font(name)
			fontCache[name] = &font
		}
	}
	return page.GetPlainText(fontCache)
}

func reconstructPageLine(line []pdf.Text) string {
	var (
		builder      strings.Builder
		lastEnd      float64
		hasWritten   bool
		pendingSpace bool
	)

	for _, text := range line {
		if text.S == " " {
			pendingSpace = true
			continue
		}

		gap := text.X - lastEnd
		switch {
		case !hasWritten:
		case pendingSpace && shouldInsertSpace(builder.String(), text.S, gap):
			builder.WriteByte(' ')
		case !pendingSpace && gap > 1.5:
			builder.WriteByte(' ')
		}

		builder.WriteString(text.S)
		lastEnd = text.X + text.W
		hasWritten = true
		pendingSpace = false
	}

	return builder.String()
}

func shouldInsertSpace(current string, next string, gap float64) bool {
	if current == "" || next == "" {
		return false
	}
	last := current[len(current)-1]
	first := next[0]
	if last == '-' || first == '-' {
		return false
	}
	if gap > 0.4 {
		return true
	}
	if isASCIIUpper(last) && isASCIIUpper(first) {
		return true
	}
	return false
}

func isASCIIUpper(b byte) bool {
	return b >= 'A' && b <= 'Z'
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
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if shouldSkipFootnoteMarkerLine(clean, lines, index, trimmed) {
			continue
		}
		clean = append(clean, trimmed)
	}
	return strings.Join(repairLineBreakHyphenationLines(clean), "\n")
}

func repairLineBreakHyphenationLines(lines []string) []string {
	if len(lines) == 0 {
		return nil
	}

	protectedCompounds := map[string]struct{}{
		"substantial-evidence": {},
		"well-founded":         {},
	}

	merged := make([]string, 0, len(lines))
	index := 0
	for index < len(lines) {
		current := lines[index]
		if index+1 >= len(lines) {
			merged = append(merged, current)
			break
		}

		prefix, left, ok := splitTrailingHyphenWord(current)
		right, rightRest, okNext := leadingWord(lines[index+1])
		if ok && okNext {
			compound := strings.ToLower(left + "-" + right)
			switch {
			case shouldPreserveLineBreakCompound(compound, protectedCompounds):
				merged = append(merged, strings.TrimSpace(prefix+left+"-"+right+rightRest))
				index += 2
				continue
			case shouldRepairLineBreakHyphen(left, right):
				merged = append(merged, strings.TrimSpace(prefix+left+right+rightRest))
				index += 2
				continue
			}
		}

		merged = append(merged, current)
		index++
	}
	return merged
}

func splitTrailingHyphenWord(line string) (string, string, bool) {
	match := trailingHyphenPattern.FindStringSubmatchIndex(line)
	if len(match) < 4 {
		return "", "", false
	}
	return line[:match[2]], line[match[2]:match[3]], true
}

func leadingWord(line string) (string, string, bool) {
	line = strings.TrimLeft(line, " ")
	end := 0
	for end < len(line) {
		b := line[end]
		if (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') {
			end++
			continue
		}
		break
	}
	if end == 0 {
		return "", "", false
	}
	return line[:end], line[end:], true
}

func shouldPreserveLineBreakCompound(compound string, protected map[string]struct{}) bool {
	_, ok := protected[compound]
	return ok
}

func shouldRepairLineBreakHyphen(left string, right string) bool {
	if left == "" || right == "" {
		return false
	}
	return startsWithLowerASCII(right)
}

func startsWithLowerASCII(text string) bool {
	if text == "" {
		return false
	}
	b := text[0]
	return b >= 'a' && b <= 'z'
}

func shouldSkipFootnoteMarkerLine(clean []string, lines []string, index int, line string) bool {
	if !digitsOnly(line) || len(clean) == 0 {
		return false
	}
	previous := clean[len(clean)-1]
	next, ok := nextNonEmptyLine(lines, index+1)
	if !ok {
		return false
	}
	return endsWithLower(previous) && startsWithLower(next)
}

func nextNonEmptyLine(lines []string, start int) (string, bool) {
	for index := start; index < len(lines); index++ {
		trimmed := strings.TrimSpace(lines[index])
		if trimmed != "" {
			return trimmed, true
		}
	}
	return "", false
}

func digitsOnly(line string) bool {
	for _, r := range line {
		if r < '0' || r > '9' {
			return false
		}
	}
	return line != ""
}

func endsWithLower(text string) bool {
	for index := len(text) - 1; index >= 0; index-- {
		r := text[index]
		if r >= 'a' && r <= 'z' {
			return true
		}
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return false
		}
	}
	return false
}

func startsWithLower(text string) bool {
	for index := 0; index < len(text); index++ {
		r := text[index]
		if r >= 'a' && r <= 'z' {
			return true
		}
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return false
		}
	}
	return false
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
	normalized := strings.Join(strings.Fields(text), " ")
	matches := fullCaptionPattern.FindAllStringSubmatch(normalized, -1)
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
