package model

// Todo represents a per-ticket checklist item.
type Todo struct {
	ID    int // database ID (0 for unsaved)
	Title string
	Done  bool
}
