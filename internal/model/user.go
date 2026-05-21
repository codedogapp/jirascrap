package model

// User represents a Jira user for @mention autocomplete.
type User struct {
	AccountID   string
	DisplayName string
}
