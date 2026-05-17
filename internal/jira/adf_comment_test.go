package jira

import (
	"encoding/json"
	"testing"
)

func TestBuildCommentADF_PlainText(t *testing.T) {
	result := BuildCommentADF("Hello world", nil)
	got := mustJSON(t, result)

	expected := mustJSON(t, adf(
		paragraph(map[string]any{"type": "text", "text": "Hello world"}),
	))
	if got != expected {
		t.Errorf("got %s, want %s", got, expected)
	}
}

func TestBuildCommentADF_Multiline(t *testing.T) {
	result := BuildCommentADF("line one\nline two", nil)

	doc, ok := result.(map[string]any)
	if !ok {
		t.Fatal("expected map")
	}
	content := doc["content"].([]any)
	if len(content) != 2 {
		t.Fatalf("expected 2 paragraphs, got %d", len(content))
	}
}

func TestBuildCommentADF_EmptyLine(t *testing.T) {
	result := BuildCommentADF("before\n\nafter", nil)

	doc := result.(map[string]any)
	content := doc["content"].([]any)
	if len(content) != 3 {
		t.Fatalf("expected 3 paragraphs, got %d", len(content))
	}
	// middle paragraph should be a space (empty line)
	mid := content[1].(map[string]any)
	inlines := mid["content"].([]any)
	txt := inlines[0].(map[string]any)["text"].(string)
	if txt != " " {
		t.Errorf("empty line text = %q, want \" \"", txt)
	}
}

func TestBuildCommentADF_EmptyInput(t *testing.T) {
	result := BuildCommentADF("", nil)
	doc := result.(map[string]any)
	content := doc["content"].([]any)
	if len(content) != 1 {
		t.Fatalf("expected 1 fallback paragraph, got %d", len(content))
	}
}

func TestBuildCommentADF_WithMention(t *testing.T) {
	mentions := map[string]string{"Alice": "abc-123"}
	result := BuildCommentADF("Hey @Alice check this", mentions)

	doc := result.(map[string]any)
	content := doc["content"].([]any)
	para := content[0].(map[string]any)
	inlines := para["content"].([]any)

	if len(inlines) != 3 {
		t.Fatalf("expected 3 inline nodes, got %d: %s", len(inlines), mustJSON(t, inlines))
	}

	// "Hey "
	if inlines[0].(map[string]any)["text"] != "Hey " {
		t.Errorf("node[0] text = %v", inlines[0])
	}

	// mention
	mention := inlines[1].(map[string]any)
	if mention["type"] != "mention" {
		t.Errorf("node[1] type = %v", mention["type"])
	}
	attrs := mention["attrs"].(map[string]any)
	if attrs["id"] != "abc-123" {
		t.Errorf("mention id = %v", attrs["id"])
	}
	if attrs["text"] != "@Alice" {
		t.Errorf("mention text = %v", attrs["text"])
	}

	// " check this"
	if inlines[2].(map[string]any)["text"] != " check this" {
		t.Errorf("node[2] text = %v", inlines[2])
	}
}

func TestBuildCommentADF_MultipleMentions(t *testing.T) {
	mentions := map[string]string{
		"Alice": "abc-123",
		"Bob":   "def-456",
	}
	result := BuildCommentADF("@Alice and @Bob", mentions)

	doc := result.(map[string]any)
	para := doc["content"].([]any)[0].(map[string]any)
	inlines := para["content"].([]any)

	// Should have: mention(Alice), " and ", mention(Bob)
	if len(inlines) != 3 {
		t.Fatalf("expected 3 nodes, got %d: %s", len(inlines), mustJSON(t, inlines))
	}

	if inlines[0].(map[string]any)["type"] != "mention" {
		t.Errorf("node[0] should be mention")
	}
	if inlines[1].(map[string]any)["text"] != " and " {
		t.Errorf("node[1] text = %v", inlines[1])
	}
	if inlines[2].(map[string]any)["type"] != "mention" {
		t.Errorf("node[2] should be mention")
	}
}

func TestBuildCommentADF_OverlappingMentions(t *testing.T) {
	// "Al" is a prefix of "Alice" — longest should match
	mentions := map[string]string{
		"Al":    "short-id",
		"Alice": "long-id",
	}
	result := BuildCommentADF("Hey @Alice", mentions)

	doc := result.(map[string]any)
	para := doc["content"].([]any)[0].(map[string]any)
	inlines := para["content"].([]any)

	if len(inlines) != 2 {
		t.Fatalf("expected 2 nodes, got %d: %s", len(inlines), mustJSON(t, inlines))
	}

	mention := inlines[1].(map[string]any)
	attrs := mention["attrs"].(map[string]any)
	if attrs["id"] != "long-id" {
		t.Errorf("should match longest name 'Alice' (long-id), got %v", attrs["id"])
	}
	if attrs["text"] != "@Alice" {
		t.Errorf("mention text = %v", attrs["text"])
	}
}

func TestBuildCommentADF_NoMentionMatch(t *testing.T) {
	mentions := map[string]string{"Alice": "abc-123"}
	result := BuildCommentADF("Hey @Bob", mentions)

	doc := result.(map[string]any)
	para := doc["content"].([]any)[0].(map[string]any)
	inlines := para["content"].([]any)

	// No mention should be created — just plain text
	if len(inlines) != 1 {
		t.Fatalf("expected 1 text node, got %d", len(inlines))
	}
	if inlines[0].(map[string]any)["text"] != "Hey @Bob" {
		t.Errorf("text = %v", inlines[0])
	}
}

func TestBuildCommentADF_JSONRoundTrip(t *testing.T) {
	mentions := map[string]string{"Alice": "abc-123"}
	result := BuildCommentADF("Hello @Alice", mentions)

	// Ensure it marshals to valid JSON (what the API will receive)
	b, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if doc["type"] != "doc" {
		t.Errorf("type = %v", doc["type"])
	}
	if doc["version"] != float64(1) {
		t.Errorf("version = %v", doc["version"])
	}
}
