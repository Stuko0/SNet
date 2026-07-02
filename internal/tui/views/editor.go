package views

import (
	"github.com/Stuko0/SNet/internal/network"
	"github.com/Stuko0/SNet/internal/tui/theme"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Estados del editor
type editorState int

const (
	editorEditing editorState = iota
	editorSaving
	editorDone
	editorError
)

// Field representa un campo del formulario de edición
type Field struct {
	Label   string
	Setting string // nombre del setting nmcli (ej: "802-11-wireless-security.psk")
	Input   textinput.Model
}

// EditorModel es un formulario genérico para editar conexiones
type EditorModel struct {
	state    editorState
	connName string
	connType string
	fields   []Field
	focusIdx int
	// spinner   tea.Model // reusamos el spinner
	toast    string
	toastErr error
}

// NewEditor crea un editor para una conexión específica
func NewEditor(connName, connType string) EditorModel {
	fields := buildFields(connType)

	if len(fields) > 0 {
		fields[0].Input.Focus()
	}

	return EditorModel{
		state:    editorEditing,
		connName: connName,
		connType: connType,
		fields:   fields,
		focusIdx: 0,
	}
}

// buildFields genera los campos según el tipo de conexión
func buildFields(connType string) []Field {
	fields := []Field{
		{
			Label:   "Nombre",
			Setting: "connection.id",
			Input:   newInput("Nombre de la conexión", false),
		},
	}

	switch connType {
	case "wifi":
		fields = append(fields,
			Field{
				Label:   "SSID",
				Setting: "802-11-wireless.ssid",
				Input:   newInput("SSID de la red", false),
			},
			Field{
				Label:   "Contraseña",
				Setting: "802-11-wireless-security.psk",
				Input:   newInput("Contraseña WiFi", true),
			},
		)
	case "ethernet":
		fields = append(fields,
			Field{
				Label:   "MTU",
				Setting: "802-3-ethernet.mtu",
				Input:   newInput("MTU (ej: 1500)", false),
			},
		)
	}

	fields = append(fields,
		Field{
			Label:   "IPv4 (manual)",
			Setting: "ipv4.addresses",
			Input:   newInput("IP/Máscara (ej: 192.168.1.100/24)", false),
		},
		Field{
			Label:   "Gateway",
			Setting: "ipv4.gateway",
			Input:   newInput("Gateway (ej: 192.168.1.1)", false),
		},
		Field{
			Label:   "DNS",
			Setting: "ipv4.dns",
			Input:   newInput("DNS (ej: 1.1.1.1,8.8.8.8)", false),
		},
		Field{
			Label:   "Autoconnect",
			Setting: "connection.autoconnect",
			Input:   newInput("yes/no", false),
		},
	)

	return fields
}

func newInput(placeholder string, password bool) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = 128
	ti.Width = 40
	if password {
		ti.EchoMode = textinput.EchoPassword
		ti.EchoCharacter = '●'
	}
	return ti
}

type editorSaveMsg struct {
	err error
}

func saveConnection(name string, fields []Field) tea.Msg {
	for _, f := range fields {
		if f.Input.Value() == "" {
			continue
		}
		err := network.ModifyConnection(name, f.Setting, f.Input.Value())
		if err != nil {
			return editorSaveMsg{err: err}
		}
	}
	return editorSaveMsg{}
}

func (m EditorModel) Update(msg tea.Msg) (EditorModel, tea.Cmd) {
	switch msg := msg.(type) {

	case editorSaveMsg:
		if msg.err != nil {
			m.state = editorError
			m.toast = "Error al guardar: " + msg.err.Error()
			m.toastErr = msg.err
		} else {
			m.state = editorDone
			m.toast = "✓ Cambios guardados en " + m.connName
			m.toastErr = nil
		}
		return m, nil

	case tea.KeyMsg:

		if m.state == editorDone || m.state == editorError {
			m.state = editorEditing
			m.toast = ""
			return m, nil
		}

		if m.state != editorEditing {
			return m, nil
		}

		switch msg.String() {
		case "tab", "down":
			m.focusIdx = (m.focusIdx + 1) % len(m.fields)
			m.updateFocus()
			return m, nil

		case "shift+tab", "up":
			m.focusIdx = (m.focusIdx - 1 + len(m.fields)) % len(m.fields)
			m.updateFocus()
			return m, nil

		case "enter":

			if m.focusIdx == len(m.fields)-1 {
				m.state = editorSaving
				return m, func() tea.Msg {
					return saveConnection(m.connName, m.fields)
				}
			}

			m.focusIdx = (m.focusIdx + 1) % len(m.fields)
			m.updateFocus()
			return m, nil

		default:
			field := &m.fields[m.focusIdx]
			var cmd tea.Cmd
			field.Input, cmd = field.Input.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m *EditorModel) updateFocus() {
	for i := range m.fields {
		if i == m.focusIdx {
			m.fields[i].Input.Focus()
		} else {
			m.fields[i].Input.Blur()
		}
	}
}

func (m EditorModel) View() string {
	if m.state == editorSaving {
		return theme.CardStyle.Render(
			theme.CardTitleStyle.Render("💾 Editando: "+m.connName) + "\n\n" +
				"Guardando cambios...",
		)
	}

	title := theme.CardTitleStyle.Render("✏️  Editando: " + m.connName)

	var fieldsView []string
	for i, f := range m.fields {
		label := theme.LabelStyle.Render(f.Label + ":")
		input := f.Input.View()

		cursor := " "
		if i == m.focusIdx {
			cursor = lipgloss.NewStyle().Foreground(theme.ColorPrimary).Render("▸")
		}

		fieldsView = append(fieldsView, cursor+" "+label+" "+input)
	}

	help := lipgloss.NewStyle().Foreground(theme.ColorSubtle).Render(
		"  Tab/↓: Siguiente  Shift+Tab/↑: Anterior  Enter en último campo: Guardar  Esc: Volver",
	)

	var toast string
	if m.state == editorDone || m.state == editorError {
		style := theme.SuccessStyle
		icon := "✓"
		if m.toastErr != nil {
			style = theme.ErrorStyle
			icon = "✗"
		}
		toast = "\n" + theme.ToastStyle.Render(style.Render(icon+" "+m.toast))
	}

	return theme.CardStyle.Render(
		title + "\n\n" +
			lipgloss.JoinVertical(lipgloss.Left, fieldsView...) + "\n\n" +
			help +
			toast,
	)
}

// No necesitamos Init — el editor se usa como sub-modelo
func (m EditorModel) Init() tea.Cmd { return nil }

// Helpers públicos para cerrar el editor
func (m EditorModel) IsDone() bool     { return m.state == editorDone || m.state == editorError }
func (m EditorModel) ConnName() string { return m.connName }
