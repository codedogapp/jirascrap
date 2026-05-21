package jira

import (
	"sort"
	"strings"
)

// ADF (Atlassian Document Format) types for building structured comment bodies.

type AdfDocument struct {
	Type    string     `json:"type"`
	Version float64    `json:"version"`
	Content []adfBlock `json:"content"`
}

type adfBlock struct {
	Type    string      `json:"type"`
	Content []adfInline `json:"content"`
}

type adfInline struct {
	Type  string        `json:"type"`
	Text  string        `json:"text,omitempty"`
	Attrs *adfMentionAt `json:"attrs,omitempty"`
}

type adfMentionAt struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// BuildCommentADF takes plain text with @mentions and a mention map (displayName→accountId)
// and produces an ADF document with proper mention nodes.
func BuildCommentADF(text string, mentions map[string]string) AdfDocument {
	lines := strings.Split(text, "\n")
	var blocks []adfBlock

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			blocks = append(
				blocks,
				adfBlock{
					Type:    adfParagraph,
					Content: []adfInline{{Type: adfText, Text: " "}},
				},
			)
			continue
		}
		content := buildInlineWithMentions(line, mentions)
		blocks = append(
			blocks,
			adfBlock{
				Type:    adfParagraph,
				Content: content,
			},
		)
	}

	if len(blocks) == 0 {
		blocks = []adfBlock{
			{
				Type:    adfParagraph,
				Content: []adfInline{{Type: adfText, Text: " "}},
			},
		}
	}

	return AdfDocument{
		Type:    adfDoc,
		Version: 1,
		Content: blocks,
	}
}

func buildInlineWithMentions(line string, mentions map[string]string) []adfInline {
	if len(mentions) == 0 {
		return []adfInline{{Type: adfText, Text: line}}
	}

	// Sort mention names longest-first to avoid partial matches (e.g. "Al" vs "Alice").
	names := make([]string, 0, len(mentions))
	for name := range mentions {
		names = append(names, name)
	}
	sort.Slice(
		names,
		func(i, j int) bool {
			return len(names[i]) > len(names[j])
		},
	)

	var nodes []adfInline
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
			nodes = append(nodes, adfInline{Type: adfText, Text: remaining})
			break
		}

		if earliest > 0 {
			nodes = append(nodes, adfInline{Type: adfText, Text: remaining[:earliest]})
		}

		nodes = append(
			nodes,
			adfInline{
				Type: adfMention,
				Attrs: &adfMentionAt{
					ID:   matchedID,
					Text: "@" + matchedName,
				},
			},
		)

		remaining = remaining[earliest+len("@"+matchedName):]
	}

	return nodes
}
