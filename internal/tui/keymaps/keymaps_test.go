package keymaps

import (
	"testing"

	"charm.land/bubbles/v2/key"
)

func TestNoKeyBindingConflicts(t *testing.T) {
	km := DefaultKeyMap

	bindings := []struct {
		name    string
		binding key.Binding
	}{
		{"ForceQuit", km.ForceQuit},
		{"Quit", km.Quit},
		{"GoBack", km.GoBack},
		{"GoHome", km.GoHome},
		{"Select", km.Select},
		{"ToggleTagging", km.ToggleTagging},
		{"ToggleTodo", km.ToggleTodo},
		{"ToggleStatus", km.ToggleStatus},
		{"Refresh", km.Refresh},
		{"ToggleHelp", km.ToggleHelp},
		{"OpenInBrowser", km.OpenInBrowser},
		{"SendToCopilot", km.SendToCopilot},
		{"AddComment", km.AddComment},
	}

	seen := make(map[string]string)
	for _, b := range bindings {
		for _, k := range b.binding.Keys() {
			if prev, ok := seen[k]; ok {
				t.Errorf("key %q bound to both %s and %s", k, prev, b.name)
			} else {
				seen[k] = b.name
			}
		}
	}

	if t.Failed() {
		t.Logf("total bindings checked: %d", len(bindings))
	}
}
