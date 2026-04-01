package jira

import (
	"strings"
	"time"
)

type searchRequest struct {
	JQL        string   `json:"jql"`
	MaxResults int      `json:"maxResults,omitempty"`
	Expand     string   `json:"expand,omitempty"`
	Fields     []string `json:"fields,omitempty"`
}

type searchResponse struct {
	Issues []issue `json:"issues"`
}

type issue struct {
	Key            string         `json:"key"`
	Fields         issueField     `json:"fields"`
	RenderedFields renderedFields `json:"renderedFields"`
}

type renderedFields struct {
	Description string `json:"description"`
}

type issueField struct {
	Summary   string   `json:"summary"`
	Reporter  reporter `json:"reporter"`
	Status    status   `json:"status"`
	CreatedAt jiraTime `json:"created"`
	UpdatedAt jiraTime `json:"updated"`
}

type reporter struct {
	DisplayName string `json:"displayName"`
}

type status struct {
	Name string `json:"name"`
}

const jiraTimeLayout = "2006-01-02T15:04:05.000-0700"

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
