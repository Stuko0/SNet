package views

import (
	"fmt"
	"github.com/Stuko0/SNet/internal/network"
	"github.com/Stuko0/SNet/internal/tui/theme"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Estados internos de la vista Wi-Fi
type wifiState int

const (
	wifiLoading    wifiState = iota // escaneando
	wifiIdle                        // mostrando tabla
	wifiPassword                    // pidiendo contraseña
	wifiConnecting                  // conectando a red
	wifiDone                        // toast de éxito
	wifiError                       // toast de error
)

// WifiListModel es la vista de escaneo y conexión Wi-Fi
type WifiListModel struct {
	state        wifiState
	networks     []network.WiFiNetwork
	table        table.Model
	spinner      spinner.Model
	password     textinput.Model
	toast        string
	toastErr     error
	err          error
	showPassword bool
}

func NewWifiList() WifiListModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(theme.ColorPrimary)

	ti := textinput.New()
	ti.Placeholder = "Contraseña de la red"
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '●'
	ti.Focus()
	ti.CharLimit = 128
	ti.Width = 40

	return WifiListModel{
		state:    wifiLoading,
		spinner:  s,
		password: ti,
	}
}

// Init arranca el escaneo inicial
func (m WifiListModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, scanNetworks)
}

type scanResultMsg struct {
	networks []network.WiFiNetwork
	err      error
}

type connectResultMsg struct {
	ssid string
	err  error
}

// scanNetworks ejecuta el escaneo en segundo plano
func scanNetworks() tea.Msg {
	networks, err := network.ScanWiFi(true)
	return scanResultMsg{networks: networks, err: err}
}

func connectToNetwork(ssid, password string) tea.Msg {
	err := network.ConnectToWiFi(ssid, password)
	return connectResultMsg{ssid: ssid, err: err}
}

func (m WifiListModel) Update(msg tea.Msg) (WifiListModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case scanResultMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = wifiError
			m.toast = "Error al escanear: " + msg.err.Error()
			break
		}
		m.networks = msg.networks
		m.state = wifiIdle
		m.table = buildTable(msg.networks)
		m.err = nil
		return m, nil

	case connectResultMsg:
		m.state = wifiDone
		if msg.err != nil {
			m.toast = fmt.Sprintf("✗ Error al conectar a %s: %s", msg.ssid, msg.err.Error())
			m.toastErr = msg.err
		} else {
			m.toast = fmt.Sprintf("✓ Conectado a %s", msg.ssid)
			m.toastErr = nil
		}
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case RefreshMsg:
		m.state = wifiLoading
		m.toast = ""
		m.toastErr = nil
		m.err = nil
		cmds = append(cmds, m.spinner.Tick, scanNetworks)
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:

		if m.state == wifiDone || m.state == wifiError {

			m.state = wifiIdle
			m.toast = ""
			m.toastErr = nil
			m.err = nil
			return m, nil
		}

		if m.state == wifiPassword {
			return m.handlePasswordKey(msg)
		}

		if m.state == wifiIdle {
			return m.handleIdleKey(msg)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m WifiListModel) handlePasswordKey(msg tea.KeyMsg) (WifiListModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		pwd := m.password.Value()
		ssid := m.getSelectedSSID()
		if ssid == "" {
			m.state = wifiIdle
			return m, nil
		}
		m.state = wifiConnecting
		m.password.SetValue("")
		m.password.Blur()
		return m, tea.Batch(
			m.spinner.Tick,
			func() tea.Msg { return connectToNetwork(ssid, pwd) },
		)

	case "esc":
		m.state = wifiIdle
		m.password.SetValue("")
		m.password.Blur()
		return m, nil

	case "ctrl+t":

		m.showPassword = !m.showPassword
		if m.showPassword {
			m.password.EchoMode = textinput.EchoNormal
		} else {
			m.password.EchoMode = textinput.EchoPassword
		}
		return m, nil

	default:
		var cmd tea.Cmd
		m.password, cmd = m.password.Update(msg)
		return m, cmd
	}
}

func (m WifiListModel) handleIdleKey(msg tea.KeyMsg) (WifiListModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		ssid := m.getSelectedSSID()
		if ssid == "" {
			return m, nil
		}
		return m.initiateConnection(ssid)

	case "r":

		m.state = wifiLoading
		m.toast = ""
		return m, tea.Batch(m.spinner.Tick, scanNetworks)

	case "up", "k":
		m.table.MoveUp(1)
		return m, nil

	case "down", "j":
		m.table.MoveDown(1)
		return m, nil

	case "esc":

		return m, nil

	default:
		// Dejar que la tabla maneje otras teclas (PgUp, PgDown, Home, End)
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		return m, cmd
	}
}

// initiateConnection decide si pide contraseña o conecta directo
func (m WifiListModel) initiateConnection(ssid string) (WifiListModel, tea.Cmd) {

	for _, n := range m.networks {
		if n.SSID == ssid {
			if n.Known || n.Security == "" || n.Security == "Open" || n.Security == "--" {

				m.state = wifiConnecting
				return m, tea.Batch(
					m.spinner.Tick,
					func() tea.Msg { return connectToNetwork(ssid, "") },
				)
			}

			m.state = wifiPassword
			m.password = textinput.New()
			m.password.Placeholder = "Contraseña de " + ssid
			m.password.EchoMode = textinput.EchoPassword
			m.password.EchoCharacter = '●'
			m.password.Focus()
			m.password.CharLimit = 128
			m.password.Width = 40
			m.showPassword = false
			return m, nil
		}
	}

	m.toast = fmt.Sprintf("Red %s no encontrada", ssid)
	m.state = wifiError
	return m, nil
}

func (m WifiListModel) getSelectedSSID() string {
	if len(m.table.Rows()) == 0 {
		return ""
	}
	row := m.table.SelectedRow()
	if len(row) == 0 {
		return ""
	}
	return row[0]
}

func (m WifiListModel) View() string {
	if m.state == wifiLoading && len(m.networks) == 0 {
		return theme.CardStyle.Render(
			theme.CardTitleStyle.Render("📶 Redes Wi-Fi") + "\n\n" +
				m.spinner.View() + " Escaneando redes...",
		)
	}

	if m.state == wifiConnecting {
		ssid := m.getSelectedSSID()
		return theme.CardStyle.Render(
			theme.CardTitleStyle.Render("📶 Redes Wi-Fi") + "\n\n" +
				m.spinner.View() + " Conectando a " + ssid + "...",
		)
	}

	if m.state == wifiPassword {
		return m.renderPasswordView()
	}

	if m.state == wifiDone || m.state == wifiError {
		view := m.renderTableView()
		toast := m.renderToast()
		return lipgloss.JoinVertical(lipgloss.Top, view, toast)
	}

	return m.renderTableView()
}

func (m WifiListModel) renderTableView() string {
	cardTitle := theme.CardTitleStyle.Render("📶 Redes Wi-Fi")

	stats := fmt.Sprintf("  %d redes encontradas    ", len(m.networks))

	var body string
	if len(m.networks) == 0 {
		body = "\n  No se encontraron redes Wi-Fi.\n  Presiona 'r' para buscar de nuevo."
	} else {
		body = theme.TableStyle.Render(m.table.View())
	}

	help := lipgloss.NewStyle().Foreground(theme.ColorSubtle).Render(
		"  ↑/↓: Navegar  Enter: Conectar  r: Buscar  ?: Ayuda",
	)

	return theme.CardStyle.Render(
		cardTitle + "\n" +
			stats + "\n" +
			body + "\n" +
			help,
	)
}

func (m WifiListModel) renderPasswordView() string {
	ssid := m.getSelectedSSID()
	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColorPrimary).
		Padding(1, 2).
		Width(50)

	toggleHint := lipgloss.NewStyle().Foreground(theme.ColorSubtle).Render(
		"  Ctrl+t: " + func() string {
			if m.showPassword {
				return "Ocultar"
			}
			return "Mostrar"
		}() + " contraseña",
	)

	input := m.password.View()

	if m.showPassword && m.password.Value() != "" {
		input += "\n" + lipgloss.NewStyle().Foreground(theme.ColorWarning).Render(
			"  Contraseña: "+m.password.Value(),
		)
	}

	content := lipgloss.JoinVertical(lipgloss.Center,
		lipgloss.NewStyle().Foreground(theme.ColorPrimary).Bold(true).Render(
			"🔑 Conectar a: "+ssid,
		),
		"",
		input,
		toggleHint,
		"",
		lipgloss.NewStyle().Foreground(theme.ColorSubtle).Render(
			"  Enter: Conectar  Esc: Cancelar",
		),
	)

	return theme.CardStyle.Render(
		lipgloss.JoinVertical(lipgloss.Center,
			inputStyle.Render(content),
		),
	)
}

func (m WifiListModel) renderToast() string {
	var style lipgloss.Style
	var icon string
	if m.toastErr != nil {
		style = theme.ErrorStyle
		icon = "✗"
	} else {
		style = theme.SuccessStyle
		icon = "✓"
	}
	return theme.ToastStyle.Render(
		style.Render(icon + " " + m.toast),
	)
}

func buildTable(networks []network.WiFiNetwork) table.Model {
	columns := []table.Column{
		{Title: "SSID", Width: 28},
		{Title: "Seguridad", Width: 10},
		{Title: "Señal", Width: 10},
		{Title: "Frec", Width: 8},
		{Title: "", Width: 4},
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
		table.WithHeight(15),
	)

	var rows []table.Row
	for _, n := range networks {
		signalBars := n.SignalBars()
		known := ""
		if n.Known {
			known = "★"
		}
		rows = append(rows, table.Row{
			truncateSSID(n.SSID, 26),
			formatSecurity(n.Security),
			signalBars + fmt.Sprintf(" %3d%%", n.Signal),
			formatFreq(n.Freq),
			known,
		})
	}
	t.SetRows(rows)

	return t
}

func truncateSSID(ssid string, maxLen int) string {
	if len(ssid) > maxLen {
		return ssid[:maxLen-1] + "…"
	}

	if len(ssid) < maxLen {
		return ssid + strings.Repeat(" ", maxLen-len(ssid))
	}
	return ssid
}

func formatSecurity(sec string) string {
	switch sec {
	case "":
		return "Open"
	case "--":
		return "Open"
	case "WPA1":
		return "WPA1"
	case "WPA2":
		return "WPA2"
	case "WPA3":
		return "WPA3"
	default:
		if strings.Contains(sec, "WPA2") && strings.Contains(sec, "WPA3") {
			return "WPA2/3"
		}
		return sec
	}
}

func formatFreq(freq string) string {
	if strings.HasPrefix(freq, "2.4") {
		return "2.4GHz"
	}
	if strings.HasPrefix(freq, "5") {
		return "5GHz"
	}
	if strings.HasPrefix(freq, "6") {
		return "6GHz"
	}
	return freq
}
