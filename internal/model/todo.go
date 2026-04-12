package model

type Todo struct {
	Title string
	Done  bool
}

func (t Todo) FilterValue() string { return t.Title }
