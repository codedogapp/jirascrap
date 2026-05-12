package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/codedogapp/jirascrap/internal/tui/views"
)

// popupKeyHandler handles a key press for a specific popup.
type popupKeyHandler func(tea.KeyPressMsg) (tea.Model, tea.Cmd)

// PopupManager centralizes popup visibility checks, key routing, and overlay rendering.
type PopupManager struct {
	tag    *views.TagModel
	todo   *views.TodoModel
	status *views.StatusModel
	debug  *views.DebugModel
	toast  *views.ToastModel

	// Key handlers set by AppModel to avoid circular dependency.
	onTagKey    popupKeyHandler
	onTodoKey   popupKeyHandler
	onStatusKey popupKeyHandler
}

func newPopupManager(
	tag *views.TagModel,
	todo *views.TodoModel,
	status *views.StatusModel,
	debug *views.DebugModel,
	toast *views.ToastModel,
) *PopupManager {
	return &PopupManager{
		tag:    tag,
		todo:   todo,
		status: status,
		debug:  debug,
		toast:  toast,
	}
}

// SetKeyHandlers wires up the popup key handlers. Called once after AppModel construction.
func (p *PopupManager) SetKeyHandlers(tag, todo, status popupKeyHandler) {
	p.onTagKey = tag
	p.onTodoKey = todo
	p.onStatusKey = status
}

// IsActive returns true if any modal popup is visible (excludes debug/toast which are non-blocking).
func (p *PopupManager) IsActive() bool {
	return p.tag.IsVisible() || p.todo.IsVisible() || p.status.IsVisible()
}

// RouteKeyPress dispatches a key to the active popup if one is visible.
// Returns (handled, cmd). If handled is false, no popup consumed the key.
func (p *PopupManager) RouteKeyPress(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	if p.tag.IsVisible() {
		_, cmd := p.onTagKey(msg)
		return true, cmd
	}
	if p.todo.IsVisible() {
		_, cmd := p.onTodoKey(msg)
		return true, cmd
	}
	if p.status.IsVisible() {
		_, cmd := p.onStatusKey(msg)
		return true, cmd
	}
	return false, nil
}

// RouteMsg dispatches non-key messages (cursor blink, etc.) to popups that need them.
// Returns (handled, cmd).
func (p *PopupManager) RouteMsg(msg tea.Msg) (bool, tea.Cmd) {
	if p.tag.IsVisible() {
		return true, p.tag.UpdateMsg(msg)
	}
	if p.todo.IsVisible() && p.todo.IsAdding() {
		return true, p.todo.UpdateMsg(msg)
	}
	return false, nil
}

// Layers returns all non-nil overlay layers for compositor rendering.
func (p *PopupManager) Layers() []*lipgloss.Layer {
	var layers []*lipgloss.Layer

	if l := p.todo.View(); l != nil {
		layers = append(layers, l)
	}
	if l := p.tag.View(); l != nil {
		layers = append(layers, l)
	}
	if l := p.status.View(); l != nil {
		layers = append(layers, l)
	}
	if l := p.debug.View(); l != nil {
		layers = append(layers, l)
	}
	if l := p.toast.View(); l != nil {
		layers = append(layers, l)
	}

	return layers
}

// SetSize updates all popups with new terminal dimensions.
func (p *PopupManager) SetSize(contentWidth, contentHeight, termWidth, termHeight int) {
	p.tag.SetSize(contentWidth, contentHeight)
	p.todo.SetSize(contentWidth, contentHeight)
	p.status.SetSize(contentWidth, contentHeight)
	p.debug.SetSize(termWidth, termHeight)
	p.toast.SetSize(termWidth, termHeight)
}
