package jira

import (
	"encoding/json"
	"testing"
)

func TestBuildCommentADF_PlainText(t *testing.T) {
	result := BuildCommentADF("Hello world", nil)

	if result.Type != adfDoc {
		t.Errorf("type = %v, want doc", result.Type)
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 paragraph, got %d", len(result.Content))
	}
	if result.Content[0].Content[0].Text != "Hello world" {
		t.Errorf("text = %q", result.Content[0].Content[0].Text)
	}
}

func TestBuildCommentADF_Multiline(t *testing.T) {
	result := BuildCommentADF("line one\nline two", nil)

	if len(result.Content) != 2 {
		t.Fatalf("expected 2 paragraphs, got %d", len(result.Content))
	}
}

func TestBuildCommentADF_EmptyLine(t *testing.T) {
	result := BuildCommentADF("before\n\nafter", nil)

	if len(result.Content) != 3 {
		t.Fatalf("expected 3 paragraphs, got %d", len(result.Content))
	}
	// middle paragraph should be a space (empty line)
	if result.Content[1].Content[0].Text != " " {
		t.Errorf("empty line text = %q, want \" \"", result.Content[1].Content[0].Text)
	}
}

func TestBuildCommentADF_EmptyInput(t *testing.T) {
	result := BuildCommentADF("", nil)

	if len(result.Content) != 1 {
		t.Fatalf("expected 1 fallback paragraph, got %d", len(result.Content))
	}
}

func TestBuildCommentADF_WithMention(t *testing.T) {
	mentions := map[string]string{"Alice": "abc-123"}
	result := BuildCommentADF("Hey @Alice check this", mentions)

	inlines := result.Content[0].Content
	if len(inlines) != 3 {
		t.Fatalf("expected 3 inline nodes, got %d", len(inlines))
	}

	// "Hey "
	if inlines[0].Text != "Hey " {
		t.Errorf("node[0] text = %q", inlines[0].Text)
	}

	// mention
	if inlines[1].Type != adfMention {
		t.Errorf("node[1] type = %v", inlines[1].Type)
	}
	if inlines[1].Attrs.ID != "abc-123" {
		t.Errorf("mention id = %v", inlines[1].Attrs.ID)
	}
	if inlines[1].Attrs.Text != "@Alice" {
		t.Errorf("mention text = %v", inlines[1].Attrs.Text)
	}

	// " check this"
	if inlines[2].Text != " check this" {
		t.Errorf("node[2] text = %q", inlines[2].Text)
	}
}

func TestBuildCommentADF_MultipleMentions(t *testing.T) {
	mentions := map[string]string{
		"Alice": "abc-123",
		"Bob":   "def-456",
	}
	result := BuildCommentADF("@Alice and @Bob", mentions)

	inlines := result.Content[0].Content

	// Should have: mention(Alice), " and ", mention(Bob)
	if len(inlines) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(inlines))
	}

	if inlines[0].Type != adfMention {
		t.Errorf("node[0] should be mention")
	}
	if inlines[1].Text != " and " {
		t.Errorf("node[1] text = %q", inlines[1].Text)
	}
	if inlines[2].Type != adfMention {
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

	inlines := result.Content[0].Content
	if len(inlines) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(inlines))
	}

	if inlines[1].Attrs.ID != "long-id" {
		t.Errorf("should match longest name 'Alice' (long-id), got %v", inlines[1].Attrs.ID)
	}
	if inlines[1].Attrs.Text != "@Alice" {
		t.Errorf("mention text = %v", inlines[1].Attrs.Text)
	}
}

func TestBuildCommentADF_NoMentionMatch(t *testing.T) {
	mentions := map[string]string{"Alice": "abc-123"}
	result := BuildCommentADF("Hey @Bob", mentions)

	inlines := result.Content[0].Content

	// No mention should be created — just plain text
	if len(inlines) != 1 {
		t.Fatalf("expected 1 text node, got %d", len(inlines))
	}
	if inlines[0].Text != "Hey @Bob" {
		t.Errorf("text = %q", inlines[0].Text)
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
