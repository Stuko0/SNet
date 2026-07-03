package views

import (
	"fmt"
	"github.com/Stuko0/SNet/internal/network"
	"github.com/Stuko0/SNet/internal/tui/theme"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Estados internos de la vista de conexiones guardadas
type savedState int

const (
	savedLoading savedState = iota
	savedIdle
	savedConnecting
	savedConfirmDelete
	savedShowingPwd
	savedDone
	savedError
)

// SavedModel gestiona las conexiones guardadas de NetworkManager
type SavedModel struct {
	state    savedState
	conns    []network.Connection
	table    table.Model
	spinner  spinner.Model
	toast    string
	toastErr error
	err      error
	password string
}

func NewSaved() SavedModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(theme.ColorPrimary)
	return SavedModel{
		state:   savedLoading,
		spinner: s,
	}
}

func (m SavedModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, fetchConnections)
}

type connListMsg struct {
	conns []network.Connection
	err   error
}

type connActionMsg struct {
	action string // "connect", "delete", "password"
	name   string
	err    error
}

func fetchConnections() tea.Msg {
	conns, err := network.GetConnections()
	return connListMsg{conns: conns, err: err}
}

func connectConnection(name string) tea.Msg {
	err := network.ConnectionUp(name)
	return connActionMsg{action: "connect", name: name, err: err}
}

func deleteConnection(name string) tea.Msg {
	err := network.DeleteConnection(name)
	return connActionMsg{action: "delete", name: name, err: err}
}

func fetchPassword(name string) tea.Msg {
	pwd, err := network.GetConnectionPassword(name)
	if err != nil || pwd == "" {
		return connActionMsg{action: "password", name: name, err: fmt.Errorf("sin contraseña o no es WiFi")}
	}
	return connActionMsg{action: "password", name: name, err: nil}
}

func (m SavedModel) Update(msg tea.Msg) (SavedModel, tea.Cmd) {
	switch msg := msg.(type) {

	case connListMsg:
		m.state = savedIdle
		if msg.err != nil {
			m.err = msg.err
			m.toast = "Error: " + msg.err.Error()
			m.state = savedError
			break
		}
		m.conns = msg.conns
		m.table = buildConnTable(msg.conns)
		m.err = nil
		return m, nil

	case connActionMsg:
		switch msg.action {
		case "connect":
			m.state = savedDone
			if msg.err != nil {
				m.toast = fmt.Sprintf("✗ Error al conectar %s: %s", msg.name, msg.err.Error())
				m.toastErr = msg.err
			} else {
				m.toast = fmt.Sprintf("✓ Conectado a %s", msg.name)
				m.toastErr = nil

				return m, fetchConnections
			}
		case "delete":
			m.state = savedDone
			if msg.err != nil {
				m.toast = fmt.Sprintf("✗ Error al eliminar %s: %s", msg.name, msg.err.Error())
				m.toastErr = msg.err
			} else {
				m.toast = fmt.Sprintf("✓ Eliminada: %s", msg.name)
				m.toastErr = nil

				return m, fetchConnections
			}
		case "password":
			m.state = savedShowingPwd
			if msg.err != nil {
				m.toast = msg.err.Error()
				m.toastErr = msg.err
				m.password = ""
				return m, nil
			}
		}
		return m, nil

	case RefreshMsg:
		m.state = savedLoading
		m.toast = ""
		m.toastErr = nil
		return m, tea.Batch(m.spinner.Tick, fetchConnections)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		if m.state == savedDone || m.state == savedError {
			m.state = savedIdle
			m.toast = ""
			m.toastErr = nil
			return m, nil
		}
		if m.state == savedConfirmDelete {
			return m.handleDeleteConfirm(msg)
		}
		if m.state == savedShowingPwd {

			m.state = savedIdle
			m.password = ""
			return m, nil
		}
		if m.state == savedIdle {
			return m.handleIdleKey(msg)
		}
	}

	return m, nil
}

func (m SavedModel) handleDeleteConfirm(msg tea.KeyMsg) (SavedModel, tea.Cmd) {
	switch msg.String() {
	case "enter", "y":
		name := m.getSelectedName()
		m.state = savedLoading
		return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
			return deleteConnection(name)
		})
	case "esc", "n", "q":
		m.state = savedIdle
		return m, nil
	default:
		return m, nil
	}
}

func (m SavedModel) handleIdleKey(msg tea.KeyMsg) (SavedModel, tea.Cmd) {
	sel := m.getSelectedName()

	switch msg.String() {
	case "enter":
		if sel == "" {
			return m, nil
		}
		m.state = savedConnecting
		return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
			return connectConnection(sel)
		})

	case "d":
		if sel == "" {
			return m, nil
		}
		m.state = savedConfirmDelete
		return m, nil

	case "e":
		if sel == "" {
			return m, nil
		}
		connType := m.getSelectedType()
		return m, func() tea.Msg {
			return EditConnectionMsg{Name: sel, Type: connType}
		}

	case "p":
		if sel == "" {
			return m, nil
		}
		m.state = savedLoading
		return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
			return fetchPassword(sel)
		})

	case "r":
		m.state = savedLoading
		m.toast = ""
		return m, tea.Batch(m.spinner.Tick, fetchConnections)

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

func (m SavedModel) getSelectedName() string {
	if len(m.table.Rows()) == 0 {
		return ""
	}
	row := m.table.SelectedRow()
	if len(row) == 0 {
		return ""
	}
	return row[0]
}

func (m SavedModel) getSelectedType() string {
	selName := m.getSelectedName()
	for _, c := range m.conns {
		if c.Name == selName {
			return c.Type
		}
	}
	return ""
}

type EditConnectionMsg struct {
	Name string
	Type string
}

func (m SavedModel) View() string {
	if m.state == savedLoading {
		label := "Cargando conexiones..."
		if m.password == "" && m.state == savedLoading {
			label = "Cargando conexiones..."
		}
		return theme.CardStyle.Render(
			theme.CardTitleStyle.Render("󰆓 Conexiones Guardadas") + "\n\n" +
				m.spinner.View() + " " + label,
		)
	}

	if m.state == savedConnecting {
		return theme.CardStyle.Render(
			theme.CardTitleStyle.Render("󰆓 Conexiones Guardadas") + "\n\n" +
				m.spinner.View() + " Conectando a " + m.getSelectedName() + "...",
		)
	}

	if m.state == savedConfirmDelete {
		return m.renderDeleteConfirm()
	}

	if m.state == savedShowingPwd {
		if m.password != "" {
			return m.renderTableView()
		}

		return m.renderTableView()
	}

	if m.state == savedDone || m.state == savedError {
		view := m.renderTableView()
		toast := m.renderToast()
		return lipgloss.JoinVertical(lipgloss.Top, view, toast)
	}

	return m.renderTableView()
}

func (m SavedModel) renderTableView() string {
	title := theme.CardTitleStyle.Render("󰆓 Conexiones Guardadas")
	stats := fmt.Sprintf("  %d conexiones    ", len(m.conns))

	var body string
	if len(m.conns) == 0 {
		body = "\n  No hay conexiones guardadas."
	} else {
		body = theme.TableStyle.Render(m.table.View())
	}

	help := lipgloss.NewStyle().Foreground(theme.ColorSubtle).Render(
		"  ↑/↓: Navegar  Enter: Conectar  d: Eliminar  p: Ver contraseña  r: Refrescar  ?: Ayuda",
	)

	return theme.CardStyle.Render(
		title + "\n" +
			stats + "\n" +
			body + "\n" +
			help,
	)
}

func (m SavedModel) renderDeleteConfirm() string {
	sel := m.getSelectedName()
	confirmBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColorDanger).
		Padding(1, 2).
		Width(50).
		Render(
			lipgloss.JoinVertical(lipgloss.Center,
				lipgloss.NewStyle().Foreground(theme.ColorDanger).Bold(true).Render("🗑 Eliminar conexión"),
				"",
				theme.ValueStyle.Render("¿Eliminar \""+sel+"\" de forma permanente?"),
				"",
				lipgloss.NewStyle().Foreground(theme.ColorSubtle).Render("  Enter/y: Confirmar  Esc/n: Cancelar"),
			),
		)

	tableView := m.renderTableView()
	return lipgloss.JoinVertical(lipgloss.Top,
		tableView,
		"",
		confirmBox,
	)
}

func (m SavedModel) renderToast() string {
	var style lipgloss.Style
	icon := "✓"
	if m.toastErr != nil {
		style = theme.ErrorStyle
		icon = "✗"
	} else {
		style = theme.SuccessStyle
	}
	return theme.ToastStyle.Render(style.Render(icon + " " + m.toast))
}

func buildConnTable(conns []network.Connection) table.Model {
	columns := []table.Column{
		{Title: "Nombre", Width: 28},
		{Title: "Tipo", Width: 10},
		{Title: "Dispositivo", Width: 14},
		{Title: "Auto", Width: 6},
		{Title: "", Width: 6},
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
	for _, c := range conns {
		status := ""
		if c.Active {
			status = "●"
		}
		auto := "✓"
		if !c.Autoconnect {
			auto = "✗"
		}
		rows = append(rows, table.Row{
			truncateString(c.Name, 26),
			connTypeIcon(c.Type),
			c.Device,
			auto,
			status,
		})
	}
	t.SetRows(rows)
	return t
}

func connTypeIcon(t string) string {
	switch t {
	case "wifi":
		return "󰤨 WiFi"
	case "ethernet":
		return "🔌 Eth"
	case "vpn", "openvpn":
		return "󰒄 VPN"
	case "wireguard":
		return "󰒄 WG"
	case "bridge":
		return "🔗 Br"
	default:
		if len(t) > 8 {
			return t[:8]
		}
		return t
	}
}

func truncateString(s string, max int) string {
	if len(s) > max {
		return s[:max-1] + "…"
	}
	if len(s) < max {
		return s + strings.Repeat(" ", max-len(s))
	}
	return s
}
