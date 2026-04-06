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
}
