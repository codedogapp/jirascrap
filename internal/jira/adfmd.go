package jira

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

const (
	adfBulletList   = "bulletList"
	adfOrderedList  = "orderedList"
	adfDoc          = "doc"
	adfParagraph    = "paragraph"
	adfText         = "text"
	adfMention      = "mention"
	adfHardBreak    = "hardBreak"
	adfType         = "type"
	adfAttrs        = "attrs"
	adfContent      = "content"
	adfMarks        = "marks"
	adfOpaquePrefix = "<!-- adf:"
)

// ADFToMarkdown converts an ADF document to Markdown text
func ADFToMarkdown(node any) string {
	return adfToMarkdown(node)
}

// MarkdownToADF converts Markdown text to an ADF document
func MarkdownToADF(md string) any {
	return markdownToADF(md)
}

// maxADFDepth limits recursion depth to prevent stack overflow on deeply nested documents.
const maxADFDepth = 50

func adfToMarkdown(node any) string {
	doc, ok := node.(map[string]any)
	if !ok {
		return ""
	}
	content, ok := doc[adfContent].([]any)
	if !ok {
		return ""
	}
	var parts []string
	for _, child := range content {
		if s := blockToMarkdown(child, 0, 0); s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, "\n\n")
}

func blockToMarkdown(node any, indent int, depth int) string {
	if depth > maxADFDepth {
		return ""
	}
	block, ok := node.(map[string]any)
	if !ok {
		return ""
	}
	nodeType, _ := block[adfType].(string)
	content, _ := block[adfContent].([]any)
	prefix := strings.Repeat(" ", indent)

	switch nodeType {
	case adfParagraph:
		return prefix + inlineToMarkdown(content)
	case "heading":
		return prefix + headingToMarkdown(block, content)
	case adfBulletList:
		return bulletListToMarkdown(content, indent, depth)
	case adfOrderedList:
		return orderedListToMarkdown(content, indent, depth)
	case "codeBlock":
		return codeBlockToMarkdown(block, content, prefix)
	case "blockquote":
		return blockquoteToMarkdown(content, prefix, depth)
	case "rule":
		return prefix + "---"
	case "table":
		return tableToMarkdown(content, indent)
	default:
		return opaqueMarker(block)
	}
}

func headingToMarkdown(block map[string]any, content []any) string {
	level := 1
	if attrs, ok := block[adfAttrs].(map[string]any); ok {
		if l, ok := attrs["level"].(float64); ok {
			level = int(l)
		}
	}
	text := inlineToMarkdown(content)
	text = strings.ReplaceAll(text, "**", "")
	return strings.Repeat("#", level) + " " + text
}

func bulletListToMarkdown(content []any, indent int, depth int) string {
	var items []string
	for _, item := range content {
		items = append(items, listItemToMarkdown(item, indent, "- ", depth+1))
	}
	return strings.Join(items, "\n")
}

func orderedListToMarkdown(content []any, indent int, depth int) string {
	var items []string
	for i, item := range content {
		marker := fmt.Sprintf("%d. ", i+1)
		items = append(items, listItemToMarkdown(item, indent, marker, depth+1))
	}
	return strings.Join(items, "\n")
}

func codeBlockToMarkdown(block map[string]any, content []any, prefix string) string {
	lang := ""
	if attrs, ok := block[adfAttrs].(map[string]any); ok {
		lang, _ = attrs["language"].(string)
	}
	text := collectPlainText(content)
	return prefix + "```" + lang + "\n" + text + "\n" + prefix + "```"
}

func blockquoteToMarkdown(content []any, prefix string, depth int) string {
	var lines []string
	for _, child := range content {
		md := blockToMarkdown(child, 0, depth+1)
		for _, line := range strings.Split(md, "\n") {
			lines = append(lines, prefix+"> "+line)
		}
	}
	return strings.Join(lines, "\n")
}

func listItemToMarkdown(node any, indent int, marker string, depth int) string {
	if depth > maxADFDepth {
		return ""
	}
	item, ok := node.(map[string]any)
	if !ok {
		return ""
	}
	content, _ := item[adfContent].([]any)
	prefix := strings.Repeat(" ", indent)
	contIndent := indent + len(marker)

	var parts []string
	first := true
	for _, child := range content {
		childBlock, ok := child.(map[string]any)
		if !ok {
			continue
		}
		childType, _ := childBlock[adfType].(string)

		switch childType {
		case adfParagraph:
			childContent, _ := childBlock[adfContent].([]any)
			text := inlineToMarkdown(childContent)
			if first {
				parts = append(parts, prefix+marker+text)
				first = false
			} else {
				parts = append(parts, strings.Repeat(" ", contIndent)+text)
			}
		case adfBulletList, adfOrderedList:
			parts = append(parts, blockToMarkdown(child, contIndent, depth+1))
		default:
			parts = append(parts, blockToMarkdown(child, contIndent, depth+1))
		}
	}
	return strings.Join(parts, "\n")
}

func inlineToMarkdown(content []any) string {
	var parts []string
	for _, child := range content {
		inline, ok := child.(map[string]any)
		if !ok {
			continue
		}
		if md := inlineNodeToMarkdown(inline); md != "" {
			parts = append(parts, md)
		}
	}
	return strings.Join(parts, "")
}

func inlineNodeToMarkdown(inline map[string]any) string {
	nodeType, _ := inline[adfType].(string)

	switch nodeType {
	case adfText:
		text, _ := inline[adfText].(string)
		marks, _ := inline[adfMarks].([]any)
		return applyMarksMD(text, marks)

	case adfMention:
		return mentionToMarkdown(inline)

	case "emoji":
		if attrs, ok := inline[adfAttrs].(map[string]any); ok {
			if shortName, ok := attrs["shortName"].(string); ok {
				return shortName
			}
		}
		return ""

	case adfHardBreak:
		return "  \n"

	case "inlineCard":
		if attrs, ok := inline[adfAttrs].(map[string]any); ok {
			if url, ok := attrs["url"].(string); ok {
				return url
			}
		}
		return ""

	default:
		return opaqueMarker(inline)
	}
}

func mentionToMarkdown(inline map[string]any) string {
	attrs, ok := inline[adfAttrs].(map[string]any)
	if !ok {
		return ""
	}
	displayName, _ := attrs[adfText].(string)
	accountID, _ := attrs["id"].(string)
	return fmt.Sprintf(
		"[@%s](accountid:%s)",
		strings.TrimPrefix(displayName, "@"),
		accountID,
	)
}

func applyMarksMD(text string, marks []any) string {
	var linkHref string
	for _, m := range marks {
		mark, ok := m.(map[string]any)
		if !ok {
			continue
		}
		markType, _ := mark[adfType].(string)
		switch markType {
		case "strong":
			text = "**" + text + "**"
		case "em":
			text = "*" + text + "*"
		case "code":
			text = "`" + text + "`"
		case "strike":
			text = "~~" + text + "~~"
		case "underline":
			text = "<u>" + text + "</u>"
		case "link":
			if attrs, ok := mark[adfAttrs].(map[string]any); ok {
				linkHref, _ = attrs["href"].(string)
			}
		}
	}
	if linkHref != "" {
		text = "[" + text + "](" + linkHref + ")"
	}
	return text
}

func tableToMarkdown(rows []any, indent int) string {
	prefix := strings.Repeat(" ", indent)
	table := extractTableCells(rows)
	if len(table) == 0 {
		return ""
	}
	table = removeEmptyColumns(table)
	return renderMarkdownTable(table, prefix)
}

func extractTableCells(rows []any) [][]string {
	var table [][]string
	for _, row := range rows {
		rowMap, ok := row.(map[string]any)
		if !ok {
			continue
		}
		cells, _ := rowMap[adfContent].([]any)
		var rowCells []string
		for _, cell := range cells {
			rowCells = append(rowCells, cellToMarkdown(cell))
		}
		table = append(table, rowCells)
	}
	return table
}

func cellToMarkdown(cell any) string {
	cellMap, ok := cell.(map[string]any)
	if !ok {
		return ""
	}
	cellContent, _ := cellMap[adfContent].([]any)
	var cellParts []string
	for _, block := range cellContent {
		blockMap, ok := block.(map[string]any)
		if !ok {
			continue
		}
		blockContent, _ := blockMap[adfContent].([]any)
		cellParts = append(cellParts, inlineToMarkdown(blockContent))
	}
	return strings.Join(cellParts, " ")
}

func removeEmptyColumns(table [][]string) [][]string {
	headerLen := len(table[0])
	emptyColumns := map[int]bool{}
	for col := range headerLen {
		empty := true
		for _, row := range table[1:] {
			if col < len(row) && row[col] != "" {
				empty = false
				break
			}
		}
		if empty {
			emptyColumns[col] = true
		}
	}

	var filteredTable [][]string
	for _, row := range table {
		var filteredRow []string
		for i := range headerLen {
			if emptyColumns[i] {
				continue
			}
			if i < len(row) {
				filteredRow = append(filteredRow, row[i])
			} else {
				filteredRow = append(filteredRow, "")
			}
		}
		filteredTable = append(filteredTable, filteredRow)
	}
	return filteredTable
}

func renderMarkdownTable(table [][]string, prefix string) string {
	var lines []string
	for i, row := range table {
		lines = append(lines, prefix+"| "+strings.Join(row, " | ")+" |")
		if i == 0 {
			var sep []string
			for range row {
				sep = append(sep, "---")
			}
			lines = append(lines, prefix+"| "+strings.Join(sep, " | ")+" |")
		}
	}
	return strings.Join(lines, "\n")
}

func collectPlainText(content []any) string {
	var parts []string
	for _, child := range content {
		inline, ok := child.(map[string]any)
		if !ok {
			continue
		}
		nodeType, _ := inline[adfType].(string)
		switch nodeType {
		case adfText:
			text, _ := inline[adfText].(string)
			parts = append(parts, text)
		case adfHardBreak:
			parts = append(parts, "\n")
		}
	}
	return strings.Join(parts, "")
}

func opaqueMarker(node map[string]any) string {
	nodeType, _ := node[adfType].(string)
	data, err := json.Marshal(node)
	if err != nil {
		return fmt.Sprintf("%s%s (marshal error) -->", adfOpaquePrefix, nodeType)
	}
	return fmt.Sprintf("%s%s %s -->", adfOpaquePrefix, nodeType, string(data))
}

func markdownToADF(md string) any {
	lines := strings.Split(md, "\n")
	var blocks []any
	i := 0

	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		if block, consumed := tryParseBlock(lines, &i, line, trimmed); consumed {
			if block != nil {
				blocks = append(blocks, block)
			}
			continue
		}

		blocks = append(blocks, parseParagraph(lines, &i))
	}

	return map[string]any{
		adfType:    adfDoc,
		"version":  float64(1),
		adfContent: blocks,
	}
}

// tryParseBlock attempts to parse a block-level element from the current line.
// Returns (block, true) if matched, (nil, true) for blank lines, (nil, false) if no match.
func tryParseBlock(lines []string, i *int, line, trimmed string) (any, bool) {
	if strings.HasPrefix(trimmed, adfOpaquePrefix) {
		if node := restoreOpaqueMarker(line); node != nil {
			*i++
			return node, true
		}
	}

	if block, ok := tryParseCodeBlock(lines, i, trimmed); ok {
		return block, true
	}

	if m := headingRe.FindStringSubmatch(line); m != nil {
		*i++
		return map[string]any{
			adfType:    "heading",
			adfAttrs:   map[string]any{"level": float64(len(m[1]))},
			adfContent: parseInline(m[2]),
		}, true
	}

	if trimmed == "---" || trimmed == "***" || trimmed == "___" {
		*i++
		return map[string]any{adfType: "rule"}, true
	}

	if block, ok := tryParseBlockquote(lines, i, trimmed); ok {
		return block, true
	}

	if block, ok := tryParseTable(lines, i, trimmed); ok {
		return block, true
	}

	if strings.HasPrefix(trimmed, "- ") {
		return parseList(lines, i, "bullet"), true
	}
	if orderedListRe.MatchString(trimmed) {
		return parseList(lines, i, "ordered"), true
	}

	if trimmed == "" {
		*i++
		return nil, true
	}

	return nil, false
}

func tryParseCodeBlock(lines []string, i *int, trimmed string) (any, bool) {
	lang, ok := strings.CutPrefix(trimmed, "```")
	if !ok {
		return nil, false
	}
	var codeLines []string
	*i++
	for *i < len(lines) {
		if strings.TrimSpace(lines[*i]) == "```" {
			*i++
			break
		}
		codeLines = append(codeLines, lines[*i])
		*i++
	}
	return map[string]any{
		adfType:    "codeBlock",
		adfAttrs:   map[string]any{"language": lang},
		adfContent: []any{map[string]any{adfType: adfText, adfText: strings.Join(codeLines, "\n")}},
	}, true
}

func tryParseBlockquote(lines []string, i *int, trimmed string) (any, bool) {
	if !strings.HasPrefix(trimmed, "> ") {
		return nil, false
	}
	var quoteLines []string
	for *i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[*i]), "> ") {
		quoteLines = append(quoteLines, strings.TrimPrefix(strings.TrimSpace(lines[*i]), "> "))
		*i++
	}
	inner := markdownToADF(strings.Join(quoteLines, "\n"))
	innerDoc, _ := inner.(map[string]any)
	innerContent, _ := innerDoc[adfContent].([]any)
	return map[string]any{
		adfType:    "blockquote",
		adfContent: innerContent,
	}, true
}

func tryParseTable(lines []string, i *int, trimmed string) (any, bool) {
	if !strings.HasPrefix(trimmed, "|") || !strings.Contains(trimmed[1:], "|") {
		return nil, false
	}
	var tableLines []string
	for *i < len(lines) {
		tl := strings.TrimSpace(lines[*i])
		if !strings.HasPrefix(tl, "|") {
			break
		}
		tableLines = append(tableLines, tl)
		*i++
	}
	return parseTable(tableLines), true
}

func parseParagraph(lines []string, i *int) any {
	var paraLines []string
	for *i < len(lines) {
		pl := lines[*i]
		ptrimmed := strings.TrimSpace(pl)
		if isBlockBreak(ptrimmed) {
			break
		}
		paraLines = append(paraLines, pl)
		*i++
	}
	text := strings.Join(paraLines, "\n")
	return map[string]any{
		adfType:    adfParagraph,
		adfContent: parseInline(text),
	}
}

func isBlockBreak(trimmed string) bool {
	return trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "```") ||
		strings.HasPrefix(trimmed, "> ") || strings.HasPrefix(trimmed, "- ") ||
		orderedListRe.MatchString(trimmed) || strings.HasPrefix(trimmed, "|") ||
		trimmed == "---" || trimmed == "***" || trimmed == "___" ||
		strings.HasPrefix(trimmed, adfOpaquePrefix)
}

var (
	headingRe       = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
	orderedListRe   = regexp.MustCompile(`^\d+\.\s+`)
	boldRe          = regexp.MustCompile(`\*\*(.+?)\*\*`)
	italicRe        = regexp.MustCompile(`(?:^|[^*])\*([^*]+?)\*(?:[^*]|$)`)
	codeRe          = regexp.MustCompile("`([^`]+)`")
	strikeRe        = regexp.MustCompile(`~~(.+?)~~`)
	underlineRe     = regexp.MustCompile(`<u>(.+?)</u>`)
	linkRe          = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	mentionRe       = regexp.MustCompile(`\[@([^\]]+)\]\(accountid:([^)]+)\)`)
	bulletMarkerRe  = regexp.MustCompile(`^(\s*)- (.*)$`)
	orderedMarkerRe = regexp.MustCompile(`^(\s*)\d+\.\s+(.*)$`)
)

func parseInline(text string) []any {
	segments := strings.Split(text, "  \n")
	var result []any
	for si, segment := range segments {
		if si > 0 {
			result = append(result, map[string]any{adfType: adfHardBreak})
		}
		for li, line := range strings.Split(segment, "\n") {
			if li > 0 {
				result = append(result, map[string]any{adfType: adfText, adfText: " "})
			}
			result = append(result, parseInlineSegment(line)...)
		}
	}
	return result
}

type inlineRule struct {
	re    *regexp.Regexp
	build func(text string, loc []int, sub []string) (start, end int, node any, ok bool)
}

var inlineRules = []inlineRule{
	{mentionRe, func(_ string, loc []int, sub []string) (int, int, any, bool) {
		return loc[0], loc[1], map[string]any{
			adfType:  adfMention,
			adfAttrs: map[string]any{adfText: "@" + sub[1], "id": sub[2]},
		}, true
	}},
	{linkRe, func(text string, loc []int, sub []string) (int, int, any, bool) {
		if strings.HasPrefix(text[loc[0]:], "[@") {
			return 0, 0, nil, false
		}
		return loc[0], loc[1], map[string]any{
			adfType: adfText,
			adfText: sub[1],
			adfMarks: []any{
				map[string]any{adfType: "link", adfAttrs: map[string]any{"href": sub[2]}},
			},
		}, true
	}},
	{boldRe, markRule("strong")},
	{codeRe, markRule("code")},
	{strikeRe, markRule("strike")},
	{underlineRe, markRule("underline")},
	{italicRe, func(text string, loc []int, sub []string) (int, int, any, bool) {
		actualStart := strings.Index(text[loc[0]:], "*"+sub[1]+"*")
		if actualStart < 0 {
			return 0, 0, nil, false
		}
		actualStart += loc[0]
		actualEnd := actualStart + len("*"+sub[1]+"*")
		return actualStart, actualEnd, map[string]any{
			adfType: adfText, adfText: sub[1],
			adfMarks: []any{map[string]any{adfType: "em"}},
		}, true
	}},
}

func markRule(markType string) func(string, []int, []string) (int, int, any, bool) {
	return func(_ string, loc []int, sub []string) (int, int, any, bool) {
		return loc[0], loc[1], map[string]any{
			adfType: adfText, adfText: sub[1],
			adfMarks: []any{map[string]any{adfType: markType}},
		}, true
	}
}

func parseInlineSegment(text string) []any {
	if text == "" {
		return nil
	}

	type inlineMatch struct {
		start, end int
		node       any
	}

	var earliest *inlineMatch
	for _, rule := range inlineRules {
		loc := rule.re.FindStringIndex(text)
		if loc == nil {
			continue
		}
		sub := rule.re.FindStringSubmatch(text)
		start, end, node, ok := rule.build(text, loc, sub)
		if !ok {
			continue
		}
		if earliest == nil || start < earliest.start {
			earliest = &inlineMatch{start, end, node}
		}
	}

	if earliest == nil {
		return []any{map[string]any{adfType: adfText, adfText: text}}
	}

	var result []any
	if earliest.start > 0 {
		result = append(result, map[string]any{adfType: adfText, adfText: text[:earliest.start]})
	}
	result = append(result, earliest.node)
	if earliest.end < len(text) {
		result = append(result, parseInlineSegment(text[earliest.end:])...)
	}
	return result
}

func parseList(lines []string, idx *int, listType string) map[string]any {
	listNodeType := adfBulletList
	markerRe := bulletMarkerRe
	if listType == "ordered" {
		listNodeType = adfOrderedList
		markerRe = orderedMarkerRe
	}

	var items []any
	baseIndent := len(lines[*idx]) - len(strings.TrimLeft(lines[*idx], " "))

	for *idx < len(lines) {
		item, ok := parseListItem(lines, idx, baseIndent, markerRe)
		if !ok {
			break
		}
		items = append(items, item)
	}

	return map[string]any{
		adfType:    listNodeType,
		adfContent: items,
	}
}

func parseListItem(
	lines []string,
	idx *int,
	baseIndent int,
	markerRe *regexp.Regexp,
) (map[string]any, bool) {
	line := lines[*idx]
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return nil, false
	}
	currentIndent := len(line) - len(strings.TrimLeft(line, " "))
	if currentIndent != baseIndent {
		return nil, false
	}

	m := markerRe.FindStringSubmatch(line)
	if m == nil {
		return nil, false
	}

	*idx++
	itemContent := []any{
		map[string]any{
			adfType:    adfParagraph,
			adfContent: parseInline(m[2]),
		},
	}

	if nested, ok := tryParseNestedList(lines, idx, baseIndent); ok {
		itemContent = append(itemContent, nested)
	}

	return map[string]any{
		adfType:    "listItem",
		adfContent: itemContent,
	}, true
}

func tryParseNestedList(lines []string, idx *int, baseIndent int) (map[string]any, bool) {
	if *idx >= len(lines) {
		return nil, false
	}
	nextLine := lines[*idx]
	nextTrimmed := strings.TrimSpace(nextLine)
	nextIndent := len(nextLine) - len(strings.TrimLeft(nextLine, " "))
	if nextIndent <= baseIndent {
		return nil, false
	}

	if strings.HasPrefix(nextTrimmed, "- ") {
		return parseList(lines, idx, "bullet"), true
	}
	if orderedListRe.MatchString(nextTrimmed) {
		return parseList(lines, idx, "ordered"), true
	}
	return nil, false
}

func parseTable(lines []string) map[string]any {
	var rows []any
	for i, line := range lines {
		cells := splitTableRow(line)
		if i == 1 && len(cells) > 0 && isSeparatorRow(cells) {
			continue
		}
		cellType := "tableCell"
		if i == 0 {
			cellType = "tableHeader"
		}
		var adfCells []any
		for _, cell := range cells {
			adfCells = append(adfCells, map[string]any{
				adfType: cellType,
				adfContent: []any{
					map[string]any{
						adfType:    adfParagraph,
						adfContent: parseInline(strings.TrimSpace(cell)),
					},
				},
			})
		}
		rows = append(rows, map[string]any{
			adfType:    "tableRow",
			adfContent: adfCells,
		})
	}
	return map[string]any{
		adfType:    "table",
		adfContent: rows,
	}
}

func splitTableRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	parts := strings.Split(line, "|")
	cells := make([]string, 0, len(parts))
	for _, p := range parts {
		cells = append(cells, strings.TrimSpace(p))
	}
	return cells
}

func isSeparatorRow(cells []string) bool {
	for _, c := range cells {
		stripped := strings.TrimSpace(c)
		stripped = strings.Trim(stripped, ":-")
		if stripped != "" {
			return false
		}
	}
	return true
}

func restoreOpaqueMarker(line string) map[string]any {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, adfOpaquePrefix) || !strings.HasSuffix(trimmed, "-->") {
		return nil
	}
	inner := strings.TrimPrefix(trimmed, adfOpaquePrefix)
	inner = strings.TrimSuffix(inner, "-->")
	inner = strings.TrimSpace(inner)

	jsonStart := strings.Index(inner, "{")
	if jsonStart < 0 {
		return nil
	}
	jsonStr := inner[jsonStart:]

	var node map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &node); err != nil {
		return nil
	}
	return node
}
