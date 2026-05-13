package store

import (
	"fmt"
	"time"
)

// parseTime parses an RFC3339 timestamp, returning an error on failure.
func parseTime(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse time %q: %w", s, err)
	}
	return t, nil
}
