package jira

import (
	"sort"
	"strings"
)

// BuildCommentADF takes plain text with @mentions and a mention map (displayName→accountId)
// and produces an ADF document with proper mention nodes.
func BuildCommentADF(text string, mentions map[string]string) any {
	lines := strings.Split(text, "\n")
	var blocks []any

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			blocks = append(blocks, map[string]any{
				"type":    adfParagraph,
				"content": []any{map[string]any{"type": adfText, "text": " "}},
			})
			continue
		}
		content := buildInlineWithMentions(line, mentions)
		blocks = append(blocks, map[string]any{
			"type":    adfParagraph,
			"content": content,
		})
	}

	if len(blocks) == 0 {
		blocks = []any{
			map[string]any{
				"type":    adfParagraph,
				"content": []any{map[string]any{"type": adfText, "text": " "}},
			},
		}
	}

	return map[string]any{
		"type":    adfDoc,
		"version": float64(1),
		"content": blocks,
	}
}

func buildInlineWithMentions(line string, mentions map[string]string) []any {
	if len(mentions) == 0 {
		return []any{map[string]any{"type": adfText, "text": line}}
	}

	// Sort mention names longest-first to avoid partial matches (e.g. "Al" vs "Alice").
	names := make([]string, 0, len(mentions))
	for name := range mentions {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		return len(names[i]) > len(names[j])
	})

	var nodes []any
	remaining := line

	for remaining != "" {
		earliest := -1
		var matchedName string
		var matchedID string

		for _, name := range names {
			pattern := "@" + name
			idx := strings.Index(remaining, pattern)
			if idx != -1 && (earliest == -1 || idx < earliest) {
				earliest = idx
				matchedName = name
				matchedID = mentions[name]
			}
		}

		if earliest == -1 {
			nodes = append(nodes, map[string]any{"type": adfText, "text": remaining})
			break
		}

		if earliest > 0 {
			nodes = append(nodes, map[string]any{"type": adfText, "text": remaining[:earliest]})
		}

		nodes = append(nodes, map[string]any{
			"type": adfMention,
			"attrs": map[string]any{
				"id":   matchedID,
				"text": "@" + matchedName,
			},
		})

		remaining = remaining[earliest+len("@"+matchedName):]
	}

	return nodes
}
