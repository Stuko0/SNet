package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap define todas las teclas de la aplicación
type KeyMap struct {
	Quit    key.Binding
	Help    key.Binding
	Refresh key.Binding
	Tab     key.Binding
	ShiftTab key.Binding
	Enter   key.Binding
	Escape  key.Binding
	Up      key.Binding
	Down    key.Binding
	Delete  key.Binding
	Edit    key.Binding
	New     key.Binding
	Back    key.Binding
}

var Keys = KeyMap{
	Quit: key.NewBinding(
		key.WithKeys("ctrl+q", "q"),
		key.WithHelp("Ctrl+q", "Salir"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "Ayuda"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "Refrescar"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("Tab", "Siguiente pestaña"),
	),
	ShiftTab: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("Shift+Tab", "Pestaña anterior"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("Enter", "Seleccionar"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("Esc", "Volver"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "Arriba"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "Abajo"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "Eliminar"),
	),
	Edit: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "Editar"),
	),
	New: key.NewBinding(
		key.WithKeys("ctrl+n"),
		key.WithHelp("Ctrl+n", "Nuevo"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc", "backspace"),
		key.WithHelp("Esc", "Cancelar"),
	),
}

// FullHelp returns all keybindings for the help screen
func (k KeyMap) FullHelp() []key.Binding {
	return []key.Binding{
		k.Tab, k.ShiftTab, k.Enter, k.Escape,
		k.Up, k.Down, k.Refresh,
		k.Delete, k.Edit, k.New,
		k.Help, k.Quit,
	}
}
