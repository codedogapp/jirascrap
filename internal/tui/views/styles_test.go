package views

import (
	"image/color"
	"testing"
)

func TestTrimTags(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"basic", "a, b, c", []string{"a", "b", "c"}},
		{"single", "tag", []string{"tag"}},
		{"extra spaces", "  a  ,  b  ", []string{"a", "b"}},
		{"trailing comma", "a, b, ", []string{"a", "b"}},
		{"empty string", "", nil},
		{"only commas", ", , , ", nil},
		{"no spaces", "a,b,c", []string{"a", "b", "c"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trimTags(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("len: got %d (%v), want %d (%v)", len(got), got, len(tt.want), tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("index %d: got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestStatusColor(t *testing.T) {
	tests := []struct {
		status string
		want   color.Color
	}{
		{"In Progress", blue},
		{"in review", blue},
		{"development", blue},
		{"Done", green},
		{"Closed", green},
		{"Resolved", green},
		{"Blocked", red},
		{"On Hold", red},
		{"To Do", grey},
		{"Backlog", grey},
		{"Unknown Status", grey},
		{"", grey},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := statusColor(tt.status)
			if got != tt.want {
				t.Errorf("statusColor(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestPriorityColor(t *testing.T) {
	tests := []struct {
		priority string
		want     color.Color
	}{
		{"Highest", priorityColors["Highest"]},
		{"High", priorityColors["High"]},
		{"Medium", priorityColors["Medium"]},
		{"Low", priorityColors["Low"]},
		{"Lowest", priorityColors["Lowest"]},
		{"Unknown", grey},
		{"", grey},
	}
	for _, tt := range tests {
		t.Run(tt.priority, func(t *testing.T) {
			got := priorityColor(tt.priority)
			if got != tt.want {
				t.Errorf("priorityColor(%q) = %v, want %v", tt.priority, got, tt.want)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		s    string
		subs []string
		want bool
	}{
		{"hello world", []string{"world"}, true},
		{"hello world", []string{"foo", "world"}, true},
		{"hello world", []string{"foo", "bar"}, false},
		{"", []string{"a"}, false},
		{"abc", []string{}, false},
	}
	for _, tt := range tests {
		got := contains(tt.s, tt.subs...)
		if got != tt.want {
			t.Errorf("contains(%q, %v) = %v, want %v", tt.s, tt.subs, got, tt.want)
		}
	}
}
