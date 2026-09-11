package views

import (
	"strings"

	"github.com/Stuko0/SNet/internal/network"
	"github.com/Stuko0/SNet/internal/tui/theme"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Estados del editor
type editorState int

const (
	editorLoading editorState = iota
	editorEditing
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
	spinner  spinner.Model
	toast    string
	toastErr error
}

// editorSettingsMsg transporta las propiedades cargadas desde nmcli
type editorSettingsMsg struct {
	values map[string]string
	err    error
}

// settingsToLoad devuelve las propiedades de nmcli que se deben precargar según el tipo
func settingsToLoad(connType string) []string {
	settings := []string{"connection.id"}
	switch connType {
	case "wifi":
		settings = append(settings, "802-11-wireless.ssid", "802-11-wireless-security.psk")
	case "ethernet":
		settings = append(settings, "802-3-ethernet.mtu")
	case "vpn", "openvpn", "sstp":
		settings = append(settings, "vpn.user-name", "vpn.secrets")
	case "wireguard":
		settings = append(settings, "connection.interface-name")
	}
	// Se incluye el método (auto/manual) además de los valores: sin él no se
	// puede distinguir "sin IP fija" de "IP pendiente de cargar".
	settings = append(settings,
		"ipv4.method", "ipv4.addresses", "ipv4.gateway", "ipv4.dns",
		"ipv6.method", "ipv6.addresses", "ipv6.gateway", "ipv6.dns",
		"ipv4.routes", "connection.autoconnect",
	)
	return settings
}

// NewEditor crea un editor para una conexión específica
func NewEditor(connName, connType string) EditorModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(theme.ColorPrimary)

	return EditorModel{
		state:    editorLoading,
		connName: connName,
		connType: connType,
		fields:   buildFields(connType),
		spinner:  s,
	}
}

// LoadCmd returns a command that fetches current connection settings from nmcli.
func (m EditorModel) LoadCmd() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			settings := settingsToLoad(m.connType)
			vals, err := network.GetConnectionSettings(m.connName, settings...)
			if err != nil {
				// Si falla la carga, igual mostramos el editor vacío
				return editorSettingsMsg{values: nil, err: err}
			}
			return editorSettingsMsg{values: vals}
		},
	)
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
	case "vpn", "openvpn":
		fields = append(fields,
			Field{
				Label:   "Servidor",
				Setting: "vpn.data",
				Input:   newInput("Servidor (ej: vpn.example.com)", false),
			},
			Field{
				Label:   "Puerto",
				Setting: "vpn.data",
				Input:   newInput("Puerto (ej: 1194)", false),
			},
			Field{
				Label:   "Usuario",
				Setting: "vpn.user-name",
				Input:   newInput("Usuario VPN", false),
			},
			Field{
				Label:   "Contraseña",
				Setting: "vpn.secrets",
				Input:   newInput("Contraseña VPN", true),
			},
		)
	case "wireguard":
		fields = append(fields,
			Field{
				Label:   "Interfaz",
				Setting: "connection.interface-name",
				Input:   newInput("Interfaz (ej: wg0)", false),
			},
		)
	case "sstp":
		fields = append(fields,
			Field{
				Label:   "Servidor",
				Setting: "vpn.data",
				Input:   newInput("Servidor (ej: vpn.example.com)", false),
			},
			Field{
				Label:   "Usuario",
				Setting: "vpn.user-name",
				Input:   newInput("Usuario SSTP", false),
			},
			Field{
				Label:   "Contraseña",
				Setting: "vpn.secrets",
				Input:   newInput("Contraseña SSTP", true),
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
		// IPv6: sin esto no se podía configurar una red v6, que es la brecha
		// que separa un gestor "útil" de uno "completo".
		Field{
			Label:   "IPv6 (manual)",
			Setting: "ipv6.addresses",
			Input:   newInput("IPv6/Máscara (ej: 2001:db8::5/64)", false),
		},
		Field{
			Label:   "IPv6 Gateway",
			Setting: "ipv6.gateway",
			Input:   newInput("Gateway v6 (ej: fe80::1)", false),
		},
		Field{
			Label:   "IPv6 DNS",
			Setting: "ipv6.dns",
			Input:   newInput("DNS v6 (ej: 2606:4700:4700::1111)", false),
		},
		Field{
			// "Rutas" y no "Rutas estáticas": LabelStyle mide 14 columnas y un
			// label más largo se parte en dos líneas, descuadrando el form.
			Label:   "Rutas",
			Setting: "ipv4.routes",
			Input:   newInput("Rutas (ej: 10.0.0.0/8 192.168.1.254)", false),
		},
		Field{
			Label:   "Autoconnect",
			Setting: "connection.autoconnect",
			Input:   newInput("yes/no", false),
		},
	)

	return fields
}

// setFieldValues rellena los campos del formulario con los valores cargados.
// Los settings vpn.data y vpn.secrets necesitan parseo especial.
func setFieldValues(fields []Field, values map[string]string) {
	if values == nil {
		return
	}
	for i, f := range fields {
		val, ok := values[f.Setting]
		if !ok || val == "" {
			continue
		}

		// Parseo especial para vpn.data (formato "key1=val1,key2=val2,...")
		if f.Setting == "vpn.data" {
			// Intentar extraer el valor correspondiente según la etiqueta del campo
			val = extractVPNDataValue(val, f.Label)
			if val == "" {
				continue
			}
		}

		// vpn.secrets puede venir como "password=valor"
		if f.Setting == "vpn.secrets" {
			if strings.HasPrefix(val, "password=") {
				val = strings.TrimPrefix(val, "password=")
			}
		}

		fields[i].Input.SetValue(val)
	}
}

// extractVPNDataValue parsea "remote=host,port=1194,connection-type=password" y extrae el valor según label
func extractVPNDataValue(data, label string) string {
	if data == "" {
		return ""
	}
	pairs := strings.Split(data, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])
		switch label {
		case "Servidor":
			if key == "remote" || key == "gateway" {
				return val
			}
		case "Puerto":
			if key == "port" {
				return val
			}
		}
	}
	return ""
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

// saveConnection aplica los campos del formulario a la conexión.
//
// Los campos se guardan uno por uno, pero algunos dependen de otro: nmcli
// rechaza `ipv6.addresses` si el método sigue en `auto`, y lo mismo con IPv4.
// Por eso primero se resuelven los métodos y después el resto.
func saveConnection(name string, fields []Field) tea.Msg {
	// Índice por setting para poder consultar dependencias.
	valores := make(map[string]string, len(fields))
	for _, f := range fields {
		valores[f.Setting] = f.Input.Value()
	}

	// Si hay direcciones IPv4/IPv6 manuales, el método debe ser `manual`
	// (si no nmcli falla), y si se limpian vuelve a `auto`.
	dependencias := []struct {
		addrs  string
		metodo string
	}{
		{"ipv4.addresses", "ipv4.method"},
		{"ipv6.addresses", "ipv6.method"},
	}

	for _, d := range dependencias {
		if valores[d.addrs] == "" {
			continue
		}
		method := "manual"
		if err := network.ModifyConnection(name, d.metodo, method); err != nil {
			return editorSaveMsg{err: err}
		}
	}

	for _, f := range fields {
		val := f.Input.Value()
		if val == "" {
			continue
		}
		// Para vpn.secrets hay que empaquetar como "password=valor"
		setting := f.Setting
		if setting == "vpn.secrets" {
			val = "password=" + val
		}
		if err := network.ModifyConnection(name, setting, val); err != nil {
			return editorSaveMsg{err: err}
		}
	}
	return editorSaveMsg{}
}

func (m EditorModel) Update(msg tea.Msg) (EditorModel, tea.Cmd) {
	switch msg := msg.(type) {

	case editorSettingsMsg:
		m.state = editorEditing
		if msg.values != nil {
			setFieldValues(m.fields, msg.values)
		}
		if len(m.fields) > 0 {
			m.fields[0].Input.Focus()
		}
		return m, nil

	case editorSaveMsg:
		if msg.err != nil {
			m.state = editorError
			m.toast = "Error al guardar: " + msgError(msg.err)
			m.toastErr = msg.err
		} else {
			m.state = editorDone
			// Sin el "✓": renderToast/View ya anteponen el icono según el
			// resultado, y si no quedaba duplicado ("✓ ✓ Cambios guardados").
			m.toast = "Cambios guardados en " + m.connName
			m.toastErr = nil
		}
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

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
	if m.state == editorLoading {
		return theme.CardStyle.Render(
			theme.CardTitleStyle.Render("󰆓 Cargando: "+m.connName) + "\n\n" +
				m.spinner.View() + " Cargando configuración actual...",
		)
	}

	if m.state == editorSaving {
		return theme.CardStyle.Render(
			theme.CardTitleStyle.Render("󰆓 Editando: "+m.connName) + "\n\n" +
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

// Init se usa para arrancar la carga asíncrona al integrarse en el modelo ppal
func (m EditorModel) Init() tea.Cmd { return nil }

// Helpers públicos para cerrar el editor
func (m EditorModel) IsDone() bool     { return m.state == editorDone || m.state == editorError }
func (m EditorModel) ConnName() string { return m.connName }
