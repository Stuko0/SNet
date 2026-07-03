package views

import (
	"fmt"
	"github.com/Stuko0/SNet/internal/network"
	"github.com/Stuko0/SNet/internal/tui/theme"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Estados del gestor VPN
type vpnState int

const (
	vpnLoading vpnState = iota
	vpnIdle
	vpnConnecting
	vpnDisconnecting
	vpnAddType   // seleccionar tipo de VPN a añadir
	vpnAddConfig // configurar nueva VPN
	vpnDone
	vpnError
)

// VPNListModel gestiona las conexiones VPN
type VPNListModel struct {
	state    vpnState
	vpns     []network.VPNConnection
	table    table.Model
	spinner  spinner.Model
	toast    string
	toastErr error
	err      error

	// Estado para añadir VPN
	addType     string // "openvpn", "wireguard", "sstp"
	addField    int
	addName     textinput.Model
	addServer   textinput.Model
	addPort     textinput.Model
	addUser     textinput.Model
	addPassword textinput.Model
	addIface    textinput.Model
	addConfig   textinput.Model // ruta a archivo config (WireGuard)
}

func NewVPNList() VPNListModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(theme.ColorPrimary)

	return VPNListModel{
		state:       vpnLoading,
		spinner:     s,
		addName:     newVPNInput("Nombre de la conexión VPN"),
		addServer:   newVPNInput("Servidor (ej: vpn.example.com)"),
		addPort:     newVPNInput("Puerto (ej: 1194)"),
		addUser:     newVPNInput("Usuario"),
		addPassword: newVPNPassword("Contraseña"),
		addIface:    newVPNInput("Interfaz (ej: wg0)"),
		addConfig:   newVPNInput("Ruta al archivo .conf (opcional)"),
	}
}

func newVPNInput(placeholder string) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = 128
	ti.Width = 40
	return ti
}

func newVPNPassword(placeholder string) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '●'
	ti.CharLimit = 128
	ti.Width = 40
	return ti
}

func (m VPNListModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, fetchVPNs)
}

type vpnListMsg struct {
	vpns []network.VPNConnection
	err  error
}

type vpnActionMsg struct {
	action string
	err    error
}

func fetchVPNs() tea.Msg {
	vpns, err := network.GetVPNs()
	return vpnListMsg{vpns: vpns, err: err}
}

func connectVPN(name string) tea.Msg {
	err := network.ConnectionUp(name)
	return vpnActionMsg{action: "connect", err: err}
}

func disconnectVPN(name string) tea.Msg {
	err := network.ConnectionDown(name)
	return vpnActionMsg{action: "disconnect", err: err}
}

func addVPN(name, vpnType, server, port, user, password string) tea.Msg {
	var err error
	switch vpnType {
	case "openvpn":
		err = network.AddOpenVPNConnection(name, server, port, user, password)
	case "wireguard":
		err = network.AddWireGuardConnection(name, "wg0", "")
	case "sstp":
		err = network.AddSSTPConnection(name, server, user, password)
	default:
		err = fmt.Errorf("tipo VPN no soportado: %s", vpnType)
	}
	return vpnActionMsg{action: "add", err: err}
}

func (m VPNListModel) Update(msg tea.Msg) (VPNListModel, tea.Cmd) {
	switch msg := msg.(type) {

	case vpnListMsg:
		m.state = vpnIdle
		if msg.err != nil {
			m.err = msg.err
			m.toast = "Error: " + msg.err.Error()
			m.state = vpnError
			break
		}
		m.vpns = msg.vpns
		m.table = buildVPNTable(msg.vpns)
		m.err = nil
		return m, nil

	case vpnActionMsg:
		m.state = vpnDone
		if msg.err != nil {
			m.toast = "✗ Error: " + msg.err.Error()
			m.toastErr = msg.err
		} else {
			switch msg.action {
			case "connect":
				m.toast = "✓ VPN conectada"
			case "disconnect":
				m.toast = "✓ VPN desconectada"
			case "add":
				m.toast = "✓ VPN añadida"
			}
			m.toastErr = nil
			return m, fetchVPNs
		}
		return m, nil

	case RefreshMsg:
		m.state = vpnLoading
		m.toast = ""
		return m, tea.Batch(m.spinner.Tick, fetchVPNs)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		if m.state == vpnDone || m.state == vpnError {
			m.state = vpnIdle
			m.toast = ""
			m.toastErr = nil
			return m, nil
		}
		if m.state == vpnAddType {
			return m.handleAddTypeKey(msg)
		}
		if m.state == vpnAddConfig {
			return m.handleAddConfigKey(msg)
		}
		if m.state == vpnIdle || m.state == vpnConnecting {
			return m.handleIdleKey(msg)
		}
	}

	return m, nil
}

func (m VPNListModel) handleIdleKey(msg tea.KeyMsg) (VPNListModel, tea.Cmd) {
	sel := m.getSelectedName()

	switch msg.String() {
	case "enter":
		if sel == "" {
			return m, nil
		}

		if m.isActive(sel) {
			m.state = vpnDisconnecting
			return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
				return disconnectVPN(sel)
			})
		}
		m.state = vpnConnecting
		return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
			return connectVPN(sel)
		})

	case "ctrl+n":
		m.state = vpnAddType
		return m, nil

	case "r":
		m.state = vpnLoading
		return m, tea.Batch(m.spinner.Tick, fetchVPNs)

	case "up", "k":
		m.table.MoveUp(1)
		return m, nil

	case "down", "j":
		m.table.MoveDown(1)
		return m, nil

	default:
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		return m, cmd
	}
}

func (m VPNListModel) handleAddTypeKey(msg tea.KeyMsg) (VPNListModel, tea.Cmd) {
	switch msg.String() {
	case "enter":

		m.addType = "openvpn"
		m.state = vpnAddConfig
		m.addField = 0
		m.addName.Focus()
		m.resetAddFields()
		return m, nil

	case "o":
		m.addType = "openvpn"
		m.state = vpnAddConfig
		m.addField = 0
		m.addName.Focus()
		m.resetAddFields()
		return m, nil

	case "w":
		m.addType = "wireguard"
		m.state = vpnAddConfig
		m.addField = 0
		m.addName.Focus()
		m.resetAddFields()
		return m, nil

	case "s":
		m.addType = "sstp"
		m.state = vpnAddConfig
		m.addField = 0
		m.addName.Focus()
		m.resetAddFields()
		return m, nil

	case "esc":
		m.state = vpnIdle
		return m, nil

	default:
		return m, nil
	}
}

func (m VPNListModel) handleAddConfigKey(msg tea.KeyMsg) (VPNListModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = vpnIdle
		return m, nil

	case "enter":
		return m.doAddVPN()

	case "tab", "down":
		m.addField = (m.addField + 1) % m.addFieldCount()
		m.updateAddFocus()
		return m, nil

	case "shift+tab", "up":
		m.addField = (m.addField - 1 + m.addFieldCount()) % m.addFieldCount()
		m.updateAddFocus()
		return m, nil

	default:
		return m.updateAddField(msg)
	}
}

func (m VPNListModel) addFieldCount() int {
	switch m.addType {
	case "openvpn":
		return 5
	case "wireguard":
		return 3
	case "sstp":
		return 4
	default:
		return 2
	}
}

func (m *VPNListModel) resetAddFields() {
	m.addName.SetValue("")
	m.addServer.SetValue("")
	m.addPort.SetValue("1194")
	m.addUser.SetValue("")
	m.addPassword.SetValue("")
	m.addIface.SetValue("wg0")
	m.addConfig.SetValue("")
}

func (m *VPNListModel) updateAddFocus() {
	m.addName.Blur()
	m.addServer.Blur()
	m.addPort.Blur()
	m.addUser.Blur()
	m.addPassword.Blur()
	m.addIface.Blur()
	m.addConfig.Blur()

	switch m.addField {
	case 0:
		m.addName.Focus()
	case 1:
		m.addServer.Focus()
	case 2:
		if m.addType == "openvpn" || m.addType == "sstp" {
			m.addPort.Focus()
		} else {
			m.addIface.Focus()
		}
	case 3:
		if m.addType == "openvpn" {
			m.addUser.Focus()
		} else {
			m.addPassword.Focus()
		}
	case 4:
		_ = m.addConfig.Focus()
	}
}

func (m VPNListModel) updateAddField(msg tea.KeyMsg) (VPNListModel, tea.Cmd) {
	inputs := []*textinput.Model{
		&m.addName, &m.addServer, &m.addPort,
		&m.addUser, &m.addPassword, &m.addIface, &m.addConfig,
	}
	if m.addField >= 0 && m.addField < len(inputs) {
		var cmd tea.Cmd
		*inputs[m.addField], cmd = inputs[m.addField].Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m VPNListModel) doAddVPN() (VPNListModel, tea.Cmd) {
	name := m.addName.Value()
	if name == "" {
		m.toast = "✗ El nombre es obligatorio"
		m.toastErr = fmt.Errorf("nombre vacío")
		m.state = vpnError
		return m, nil
	}

	return m, func() tea.Msg {
		return addVPN(
			name,
			m.addType,
			m.addServer.Value(),
			m.addPort.Value(),
			m.addUser.Value(),
			m.addPassword.Value(),
		)
	}
}

func (m VPNListModel) getSelectedName() string {
	if len(m.table.Rows()) == 0 {
		return ""
	}
	row := m.table.SelectedRow()
	if len(row) == 0 {
		return ""
	}
	return row[0]
}

func (m VPNListModel) isActive(name string) bool {
	for _, v := range m.vpns {
		if v.Name == name && v.Active {
			return true
		}
	}
	return false
}

func (m VPNListModel) View() string {
	switch m.state {
	case vpnLoading:
		return theme.CardStyle.Render(
			theme.CardTitleStyle.Render("󰒄 VPN") + "\n\n" +
				m.spinner.View() + " Cargando VPNs...",
		)
	case vpnConnecting:
		return theme.CardStyle.Render(
			theme.CardTitleStyle.Render("󰒄 VPN") + "\n\n" +
				m.spinner.View() + " Conectando VPN...",
		)
	case vpnDisconnecting:
		return theme.CardStyle.Render(
			theme.CardTitleStyle.Render("󰒄 VPN") + "\n\n" +
				m.spinner.View() + " Desconectando VPN...",
		)
	case vpnAddType:
		return m.renderAddType()
	case vpnAddConfig:
		return m.renderAddConfig()
	default:
		view := m.renderTableView()
		if m.state == vpnDone || m.state == vpnError {
			return lipgloss.JoinVertical(lipgloss.Top, view, m.renderToast())
		}
		return view
	}
}

func (m VPNListModel) renderTableView() string {
	title := theme.CardTitleStyle.Render("󰒄 VPN")
	stats := fmt.Sprintf("  %d VPNs configuradas", len(m.vpns))

	var body string
	if len(m.vpns) == 0 {
		body = "\n  No hay VPNs configuradas.\n  Presiona Ctrl+n para añadir una."
	} else {
		body = theme.TableStyle.Render(m.table.View())
	}

	help := lipgloss.NewStyle().Foreground(theme.ColorSubtle).Render(
		"  ↑/↓: Navegar  Enter: Conectar/Desconectar  Ctrl+n: Nueva VPN  r: Refrescar",
	)

	return theme.CardStyle.Render(
		title + "\n" +
			stats + "\n" +
			body + "\n" +
			help,
	)
}

func (m VPNListModel) renderAddType() string {
	title := theme.CardTitleStyle.Render("󰒄 Nueva VPN")

	types := []struct {
		key  string
		name string
		desc string
	}{
		{"o", "OpenVPN", "Conexión estándar con servidor remoto"},
		{"w", "WireGuard", "VPN moderna con claves públicas/privadas"},
		{"s", "SSTP", "Secure Socket Tunneling Protocol"},
	}

	var items []string
	for _, t := range types {
		item := lipgloss.NewStyle().Foreground(theme.ColorPrimary).Bold(true).Render(t.key) +
			"  " + theme.ValueStyle.Render(t.name) +
			lipgloss.NewStyle().Foreground(theme.ColorSubtle).Render(" - "+t.desc)
		items = append(items, "  "+item)
	}

	help := lipgloss.NewStyle().Foreground(theme.ColorSubtle).Render(
		"  o/w/s: Seleccionar tipo  Esc: Cancelar",
	)

	return theme.CardStyle.Render(
		title + "\n\n" +
			lipgloss.JoinVertical(lipgloss.Left, items...) + "\n\n" +
			help,
	)
}

func (m VPNListModel) renderAddConfig() string {
	title := theme.CardTitleStyle.Render(
		fmt.Sprintf("󰒄 Configurar %s", m.addType),
	)

	var fields []string

	addField := func(label, cursor string, input textinput.Model) {
		fields = append(fields, cursor+"  "+theme.LabelStyle.Render(label)+" "+input.View())
	}

	cursor := func(idx int) string {
		if m.addField == idx {
			return lipgloss.NewStyle().Foreground(theme.ColorPrimary).Render("▸")
		}
		return " "
	}

	addField("Nombre:", cursor(0), m.addName)
	addField("Servidor:", cursor(1), m.addServer)

	switch m.addType {
	case "openvpn":
		addField("Puerto:", cursor(2), m.addPort)
		addField("Usuario:", cursor(3), m.addUser)
		addField("Password:", cursor(4), m.addPassword)
	case "wireguard":
		addField("Interfaz:", cursor(2), m.addIface)
		addField("Config:", cursor(3), m.addConfig)
	case "sstp":
		addField("Usuario:", cursor(2), m.addUser)
		addField("Password:", cursor(3), m.addPassword)
	}

	help := lipgloss.NewStyle().Foreground(theme.ColorSubtle).Render(
		"  Tab/↓: Siguiente  Enter: Guardar  Esc: Cancelar",
	)

	return theme.CardStyle.Render(
		title + "\n\n" +
			lipgloss.JoinVertical(lipgloss.Left, fields...) + "\n\n" +
			help,
	)
}

func (m VPNListModel) renderToast() string {
	style := theme.SuccessStyle
	icon := "✓"
	if m.toastErr != nil {
		style = theme.ErrorStyle
		icon = "✗"
	}
	return theme.ToastStyle.Render(style.Render(icon + " " + m.toast))
}

func buildVPNTable(vpns []network.VPNConnection) table.Model {
	columns := []table.Column{
		{Title: "Nombre", Width: 24},
		{Title: "Tipo", Width: 12},
		{Title: "Estado", Width: 10},
	}

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		Bold(true).
		Foreground(theme.ColorPrimary)
	s.Selected = s.Selected.
		Foreground(theme.ColorPrimary).
		Background(theme.ColorSurface).
		Bold(true)
	s.Cell = s.Cell.
		Foreground(theme.ColorText)

	t := table.New(
		table.WithColumns(columns),
		table.WithStyles(s),
		table.WithHeight(12),
	)

	var rows []table.Row
	for _, v := range vpns {
		status := "○ Inactiva"
		if v.Active {
			status = "● Activa"
		}
		rows = append(rows, table.Row{
			v.Name,
			vpnTypeLabel(v.Type),
			status,
		})
	}
	t.SetRows(rows)
	return t
}

func vpnTypeLabel(t string) string {
	switch t {
	case "vpn", "openvpn":
		return "OpenVPN"
	case "wireguard":
		return "WireGuard"
	case "sstp":
		return "SSTP"
	case "l2tp":
		return "L2TP"
	case "pptp":
		return "PPTP"
	default:
		return t
	}
}
