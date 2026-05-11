package model

import (
	"time"
)

type Ticket struct {
	ID             string
	Summary        string
	Reporter       string
	Status         string
	StatusCategory string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Markdown       string
	Tags           []string
	Priority       string
	Type           string  // issue type from Jira (e.g., "Epic", "Task", "Story", "Bug")
	EpicID         *string // optional, links ticket to parent epic
}

func (t Ticket) IsEpic() bool {
	return t.Type == "Epic"
}
