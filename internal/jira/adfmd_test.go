package jira

import (
	"encoding/json"
	"testing"
)

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return string(b)
}

func adf(content ...any) map[string]any {
	return map[string]any{
		"type":    "doc",
		"version": float64(1),
		"content": content,
	}
}

func paragraph(inlines ...any) map[string]any {
	return map[string]any{
		"type":    "paragraph",
		"content": inlines,
	}
}

func text(s string) map[string]any {
	return map[string]any{
		"type": "text",
		"text": s,
	}
}

func textWithMark(s string, marks ...map[string]any) map[string]any {
	ms := make([]any, len(marks))
	for i, m := range marks {
		ms[i] = m
	}
	return map[string]any{
		"type":  "text",
		"text":  s,
		"marks": ms,
	}
}

func mark(markType string) map[string]any {
	return map[string]any{"type": markType}
}

func linkMark(href string) map[string]any {
	return map[string]any{
		"type":  "link",
		"attrs": map[string]any{"href": href},
	}
}

func heading(level int, inlines ...any) map[string]any {
	return map[string]any{
		"type":    "heading",
		"attrs":   map[string]any{"level": float64(level)},
		"content": inlines,
	}
}

func bulletList(items ...any) map[string]any {
	return map[string]any{
		"type":    "bulletList",
		"content": items,
	}
}

func orderedList(items ...any) map[string]any {
	return map[string]any{
		"type":    "orderedList",
		"content": items,
	}
}

func listItem(blocks ...any) map[string]any {
	return map[string]any{
		"type":    "listItem",
		"content": blocks,
	}
}

func codeBlock(lang string, code string) map[string]any {
	return map[string]any{
		"type":    "codeBlock",
		"attrs":   map[string]any{"language": lang},
		"content": []any{text(code)},
	}
}

func blockquote(blocks ...any) map[string]any {
	return map[string]any{
		"type":    "blockquote",
		"content": blocks,
	}
}

func rule() map[string]any {
	return map[string]any{"type": "rule"}
}

func mention(name, id string) map[string]any {
	return map[string]any{
		"type": "mention",
		"attrs": map[string]any{
			"text": "@" + name,
			"id":   id,
		},
	}
}

func emoji(shortName string) map[string]any {
	return map[string]any{
		"type":  "emoji",
		"attrs": map[string]any{"shortName": shortName},
	}
}

func hardBreak() map[string]any {
	return map[string]any{"type": "hardBreak"}
}

func TestADFToMarkdown_Paragraph(t *testing.T) {
	doc := adf(paragraph(text("Hello world")))
	got := ADFToMarkdown(doc)
	if got != "Hello world" {
		t.Errorf("got %q, want %q", got, "Hello world")
	}
}

func TestADFToMarkdown_MultipleParagraphs(t *testing.T) {
	doc := adf(paragraph(text("First")), paragraph(text("Second")))
	got := ADFToMarkdown(doc)
	want := "First\n\nSecond"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestADFToMarkdown_Heading(t *testing.T) {
	tests := []struct {
		level int
		want  string
	}{
		{1, "# Title"},
		{2, "## Title"},
		{3, "### Title"},
	}
	for _, tt := range tests {
		doc := adf(heading(tt.level, text("Title")))
		got := ADFToMarkdown(doc)
		if got != tt.want {
			t.Errorf("heading level %d: got %q, want %q", tt.level, got, tt.want)
		}
	}
}

func TestADFToMarkdown_InlineMarks(t *testing.T) {
	tests := []struct {
		name string
		node map[string]any
		want string
	}{
		{"bold", textWithMark("bold", mark("strong")), "**bold**"},
		{"italic", textWithMark("italic", mark("em")), "*italic*"},
		{"code", textWithMark("code", mark("code")), "`code`"},
		{"strike", textWithMark("strike", mark("strike")), "~~strike~~"},
		{"underline", textWithMark("underline", mark("underline")), "<u>underline</u>"},
		{
			"link",
			textWithMark("click", linkMark("https://example.com")),
			"[click](https://example.com)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := adf(paragraph(tt.node))
			got := ADFToMarkdown(doc)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestADFToMarkdown_BulletList(t *testing.T) {
	doc := adf(bulletList(
		listItem(paragraph(text("item one"))),
		listItem(paragraph(text("item two"))),
	))
	got := ADFToMarkdown(doc)
	want := "- item one\n- item two"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestADFToMarkdown_OrderedList(t *testing.T) {
	doc := adf(orderedList(
		listItem(paragraph(text("first"))),
		listItem(paragraph(text("second"))),
	))
	got := ADFToMarkdown(doc)
	want := "1. first\n2. second"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestADFToMarkdown_CodeBlock(t *testing.T) {
	doc := adf(codeBlock("go", "fmt.Println(\"hi\")"))
	got := ADFToMarkdown(doc)
	want := "```go\nfmt.Println(\"hi\")\n```"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestADFToMarkdown_Blockquote(t *testing.T) {
	doc := adf(blockquote(paragraph(text("quoted text"))))
	got := ADFToMarkdown(doc)
	want := "> quoted text"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestADFToMarkdown_Rule(t *testing.T) {
	doc := adf(paragraph(text("before")), rule(), paragraph(text("after")))
	got := ADFToMarkdown(doc)
	want := "before\n\n---\n\nafter"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestADFToMarkdown_Mention(t *testing.T) {
	doc := adf(paragraph(mention("John", "abc123")))
	got := ADFToMarkdown(doc)
	want := "[@John](accountid:abc123)"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestADFToMarkdown_Emoji(t *testing.T) {
	doc := adf(paragraph(emoji(":thumbsup:")))
	got := ADFToMarkdown(doc)
	if got != ":thumbsup:" {
		t.Errorf("got %q, want %q", got, ":thumbsup:")
	}
}

func TestADFToMarkdown_HardBreak(t *testing.T) {
	doc := adf(paragraph(text("line1"), hardBreak(), text("line2")))
	got := ADFToMarkdown(doc)
	want := "line1  \nline2"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestADFToMarkdown_Table(t *testing.T) {
	table := map[string]any{
		"type": "table",
		"content": []any{
			map[string]any{
				"type": "tableRow",
				"content": []any{
					map[string]any{
						"type":    "tableHeader",
						"content": []any{paragraph(text("Name"))},
					},
					map[string]any{"type": "tableHeader", "content": []any{paragraph(text("Age"))}},
				},
			},
			map[string]any{
				"type": "tableRow",
				"content": []any{
					map[string]any{"type": "tableCell", "content": []any{paragraph(text("Alice"))}},
					map[string]any{"type": "tableCell", "content": []any{paragraph(text("30"))}},
				},
			},
		},
	}
	doc := adf(table)
	got := ADFToMarkdown(doc)
	want := "| Name | Age |\n| --- | --- |\n| Alice | 30 |"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestADFToMarkdown_OpaqueMarker(t *testing.T) {
	unknown := map[string]any{"type": "unknownBlock", "data": "stuff"}
	doc := adf(unknown)
	got := ADFToMarkdown(doc)
	if got == "" {
		t.Error("expected opaque marker, got empty string")
	}
	if len(got) < 10 || got[:9] != "<!-- adf:" {
		t.Errorf("expected opaque marker comment, got %q", got)
	}
}

func TestADFToMarkdown_EmptyDoc(t *testing.T) {
	got := ADFToMarkdown(adf())
	if got != "" {
		t.Errorf("got %q, want empty string", got)
	}
}

func TestADFToMarkdown_NilInput(t *testing.T) {
	got := ADFToMarkdown(nil)
	if got != "" {
		t.Errorf("got %q, want empty string", got)
	}
}

func TestADFToMarkdown_NestedList(t *testing.T) {
	doc := adf(bulletList(
		listItem(
			paragraph(text("parent")),
			bulletList(
				listItem(paragraph(text("child"))),
			),
		),
	))
	got := ADFToMarkdown(doc)
	want := "- parent\n  - child"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// Round-trip tests: ADF → Markdown → ADF
func TestRoundTrip_Paragraph(t *testing.T) {
	original := adf(paragraph(text("Hello world")))
	md := ADFToMarkdown(original)
	rebuilt := MarkdownToADF(md)
	md2 := ADFToMarkdown(rebuilt)
	if md != md2 {
		t.Errorf("round-trip mismatch:\nfirst:  %q\nsecond: %q", md, md2)
	}
}

func TestRoundTrip_HeadingAndParagraph(t *testing.T) {
	original := adf(
		heading(2, text("Title")),
		paragraph(text("body text")),
	)
	md := ADFToMarkdown(original)
	rebuilt := MarkdownToADF(md)
	md2 := ADFToMarkdown(rebuilt)
	if md != md2 {
		t.Errorf("round-trip mismatch:\nfirst:  %q\nsecond: %q", md, md2)
	}
}

func TestRoundTrip_CodeBlock(t *testing.T) {
	original := adf(codeBlock("go", "package main"))
	md := ADFToMarkdown(original)
	rebuilt := MarkdownToADF(md)
	md2 := ADFToMarkdown(rebuilt)
	if md != md2 {
		t.Errorf("round-trip mismatch:\nfirst:  %q\nsecond: %q", md, md2)
	}
}

func TestRoundTrip_Blockquote(t *testing.T) {
	original := adf(blockquote(paragraph(text("quoted"))))
	md := ADFToMarkdown(original)
	rebuilt := MarkdownToADF(md)
	md2 := ADFToMarkdown(rebuilt)
	if md != md2 {
		t.Errorf("round-trip mismatch:\nfirst:  %q\nsecond: %q", md, md2)
	}
}

func TestRoundTrip_Rule(t *testing.T) {
	original := adf(paragraph(text("before")), rule(), paragraph(text("after")))
	md := ADFToMarkdown(original)
	rebuilt := MarkdownToADF(md)
	md2 := ADFToMarkdown(rebuilt)
	if md != md2 {
		t.Errorf("round-trip mismatch:\nfirst:  %q\nsecond: %q", md, md2)
	}
}

func TestRoundTrip_BulletList(t *testing.T) {
	original := adf(bulletList(
		listItem(paragraph(text("a"))),
		listItem(paragraph(text("b"))),
	))
	md := ADFToMarkdown(original)
	rebuilt := MarkdownToADF(md)
	md2 := ADFToMarkdown(rebuilt)
	if md != md2 {
		t.Errorf("round-trip mismatch:\nfirst:  %q\nsecond: %q", md, md2)
	}
}

func TestRoundTrip_OpaqueMarker(t *testing.T) {
	unknown := map[string]any{
		"type":    "panel",
		"attrs":   map[string]any{"panelType": "info"},
		"content": []any{paragraph(text("note"))},
	}
	original := adf(unknown)
	md := ADFToMarkdown(original)
	rebuilt := MarkdownToADF(md)
	roundTripped := mustJSON(t, rebuilt)
	// The opaque marker should round-trip through JSON preservation
	if ADFToMarkdown(rebuilt) != md {
		t.Errorf(
			"opaque round-trip markdown mismatch:\nfirst:  %q\nsecond: %q\njson: %s",
			md,
			ADFToMarkdown(rebuilt),
			roundTripped,
		)
	}
}

func TestMarkdownToADF_EmptyString(t *testing.T) {
	result := MarkdownToADF("")
	doc, ok := result.(map[string]any)
	if !ok {
		t.Fatal("expected map")
	}
	content, _ := doc["content"].([]any)
	if len(content) != 0 {
		t.Errorf("expected empty content, got %d blocks", len(content))
	}
}

func TestMarkdownToADF_Table(t *testing.T) {
	md := "| A | B |\n| --- | --- |\n| 1 | 2 |"
	result := MarkdownToADF(md)
	doc := result.(map[string]any)
	content := doc["content"].([]any)
	if len(content) != 1 {
		t.Fatalf("expected 1 block, got %d", len(content))
	}
	table := content[0].(map[string]any)
	if table["type"] != "table" {
		t.Errorf("expected table, got %v", table["type"])
	}
}
