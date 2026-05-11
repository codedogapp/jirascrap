package jira

import (
	"strings"
	"time"
)

type searchRequest struct {
	MaxResults int      `json:"maxResults"`
	JQL        string   `json:"jql"`
	Expand     string   `json:"expand,omitempty"`
	Fields     []string `json:"fields,omitempty"`
}

type searchResponse struct {
	Issues []issue `json:"issues"`
}

type issue struct {
	Key    string     `json:"key"`
	Fields issueField `json:"fields"`
}

type statusCategory struct {
	Name string `json:"name"`
}

type priority struct {
	Name string `json:"name"`
}

type issueType struct {
	Name string `json:"name"`
}

type issueField struct {
	Summary     string    `json:"summary"`
	Description any       `json:"description"`
	Reporter    reporter  `json:"reporter"`
	Status      status    `json:"status"`
	Priority    priority  `json:"priority"`
	IssueType   issueType `json:"issuetype"`
	CreatedAt   jiraTime  `json:"created"`
	UpdatedAt   jiraTime  `json:"updated"`
}

type reporter struct {
	DisplayName string `json:"displayName"`
}

type status struct {
	Name           string         `json:"name"`
	StatusCategory statusCategory `json:"statusCategory"`
}

const jiraTimeLayout = "2006-01-02T15:04:05.000-0700"

// Transition represents an available status transition for an issue.
type Transition struct {
	ID               string
	Name             string
	ToStatus         string
	ToStatusCategory string
}

type transitionsResponse struct {
	Transitions []transitionEntry `json:"transitions"`
}

type transitionEntry struct {
	ID   string           `json:"id"`
	Name string           `json:"name"`
	To   transitionTarget `json:"to"`
}

type transitionTarget struct {
	Name           string         `json:"name"`
	StatusCategory statusCategory `json:"statusCategory"`
}

type transitionRequest struct {
	Transition transitionRef `json:"transition"`
}

type transitionRef struct {
	ID string `json:"id"`
}

type jiraTime struct {
	time.Time
}

func (jt *jiraTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	if s == "null" || s == "" {
		return nil
	}

	t, err := time.Parse(jiraTimeLayout, s)
	if err != nil {
		return err
	}

	jt.Time = t
	return nil
}
