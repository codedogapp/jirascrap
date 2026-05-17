package model

import "time"

// Comment represents a single Jira issue comment.
type Comment struct {
	ID        string
	Author    string
	CreatedAt time.Time
	Markdown  string
}
