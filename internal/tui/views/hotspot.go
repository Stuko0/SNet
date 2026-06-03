package views

import (
	"fmt"
	"github.com/Stuko0/SNet/internal/network"
	"github.com/Stuko0/SNet/internal/tui/theme"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Estados del hotspot
type hotspotState int

const (
	hotspotIdle      hotspotState = iota
	hotspotScanning               // obteniendo estado actual
	hotspotStarting               // iniciando hotspot
	hotspotStopping               // deteniendo hotspot
	hotspotDone
	hotspotError
)

// HotspotModel controla la creación y gestión del hotspot
type HotspotModel struct {
	state      hotspotState
	status     *network.HotspotConfig
	spinner    spinner.Model

	// Campos del formulario (solo visibles si hotspot inactivo)
	ssidInput     textinput.Model
	passwordInput textinput.Model
	band          string // "bg" o "a"
	focusField    int    // 0=SSID, 1=Password, 2=Band, 3=Start

	err       error
	toast     string
	toastErr  error
}

func NewHotspot() HotspotModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(theme.ColorPrimary)

	ssid := textinput.New()
	ssid.Placeholder = "Mi Hotspot"
	ssid.CharLimit = 32
	ssid.Width = 30
	ssid.Focus()

	pwd := textinput.New()
	pwd.Placeholder = "Mínimo 8 caracteres"
	pwd.EchoMode = textinput.EchoPassword
	pwd.EchoCharacter = '●'
	pwd.CharLimit = 64
	pwd.Width = 30

	return HotspotModel{
		state:         hotspotScanning,
		spinner:       s,
		ssidInput:     ssid,
		passwordInput: pwd,
		band:          "bg",
		focusField:    0,
	}
}

func (m HotspotModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, fetchHotspotStatus)
}

// ─── Mensajes ────────────────────────────────────────────────

type hotspotStatusMsg struct {
	status *network.HotspotConfig
	err    error
}

type hotspotActionResultMsg struct {
	action string // "start", "stop"
	err    error
}

func fetchHotspotStatus() tea.Msg {
	status, err := network.HotspotStatus()
	return hotspotStatusMsg{status: status, err: err}
}

func startHotspotAction(cfg network.HotspotConfig) tea.Msg {
	err := network.HotspotStart(cfg)
	return hotspotActionResultMsg{action: "start", err: err}
}

func stopHotspotAction() tea.Msg {
	err := network.HotspotStop()
	return hotspotActionResultMsg{action: "stop", err: err}
}

// ─── Update ──────────────────────────────────────────────────

func (m HotspotModel) Update(msg tea.Msg) (HotspotModel, tea.Cmd) {
	switch msg := msg.(type) {

	case hotspotStatusMsg:
		m.state = hotspotIdle
		if msg.err != nil {
			m.err = msg.err
		}
		if msg.status != nil {
			m.status = msg.status
			// Si está activo, rellenar campos con datos actuales
			if m.status.Active {
				m.ssidInput.SetValue(m.status.SSID)
				if m.status.Password != "" {
					m.passwordInput.SetValue(m.status.Password)
				}
				if m.status.Band != "" {
					m.band = m.status.Band
				}
			}
		}
		return m, nil

	case hotspotActionResultMsg:
		if msg.err != nil {
			m.state = hotspotError
			m.toastErr = msg.err
			m.toast = fmt.Sprintf("✗ Error: %s", msg.err.Error())
		} else {
			m.state = hotspotDone
			m.toastErr = nil
			if msg.action == "start" {
				m.toast = "✓ Hotspot iniciado"
				// Refresh status
				return m, fetchHotspotStatus
			} else {
				m.toast = "✓ Hotspot detenido"
				m.status = &network.HotspotConfig{Active: false}
			}
		}
		return m, nil

	case refreshMsg:
		m.state = hotspotScanning
		m.toast = ""
		return m, tea.Batch(m.spinner.Tick, fetchHotspotStatus)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		if m.state == hotspotDone || m.state == hotspotError {
			m.state = hotspotIdle
			m.toast = ""
			m.toastErr = nil
			return m, nil
		}

		if m.state == hotspotScanning {
			return m, nil
		}

		return m.handleKey(msg)
	}

	return m, nil
}

func (m HotspotModel) handleKey(msg tea.KeyMsg) (HotspotModel, tea.Cmd) {
	switch msg.String() {
	case "tab", "down":
		m.focusField = (m.focusField + 1) % 4
		m.updateFocus()
		return m, nil

	case "shift+tab", "up":
		m.focusField = (m.focusField - 1 + 4) % 4
		m.updateFocus()
		return m, nil

	case "enter":
		return m.handleEnter()

	case "left":
		if m.focusField == 2 {
			m.band = "bg"
			return m, nil
		}
		return m, nil

	case "right":
		if m.focusField == 2 {
			m.band = "a"
			return m, nil
		}
		return m, nil

	case "esc":
		return m, nil

	default:
		return m.updateFocusedField(msg)
	}
}

func (m HotspotModel) handleEnter() (HotspotModel, tea.Cmd) {
	// Si el hotspot ya está activo, Enter detiene
	if m.status != nil && m.status.Active {
		m.state = hotspotStopping
		return m, tea.Batch(m.spinner.Tick, stopHotspotAction)
	}

	// Si no está activo, validar e iniciar
	switch m.focusField {
	case 0: // SSID → password
		m.focusField = 1
		m.updateFocus()
		return m, nil
	case 1: // Password → band
		m.focusField = 2
		m.updateFocus()
		return m, nil
	case 2: // Band → start
		m.focusField = 3
		m.updateFocus()
		return m, nil
	case 3: // Start
		return m.doStart()
	default:
		return m, nil
	}
}

func (m HotspotModel) doStart() (HotspotModel, tea.Cmd) {
	// Validar SSID
	ssid := m.ssidInput.Value()
	if ssid == "" {
		m.toast = "✗ El SSID no puede estar vacío"
		m.toastErr = fmt.Errorf("SSID vacío")
		m.state = hotspotError
		return m, nil
	}

	// Validar password (mín 8 chars)
	pwd := m.passwordInput.Value()
	if pwd == "" {
		m.toast = "✗ La contraseña es obligatoria"
		m.toastErr = fmt.Errorf("password vacía")
		m.state = hotspotError
		return m, nil
	}
	if len(pwd) < 8 {
		m.toast = "✗ La contraseña debe tener al menos 8 caracteres"
		m.toastErr = fmt.Errorf("password muy corta")
		m.state = hotspotError
		return m, nil
	}

	iface := network.GetHotspotIface()
	if iface == "" {
		m.toast = "✗ No se encontró interfaz Wi-Fi"
		m.toastErr = fmt.Errorf("sin interfaz Wi-Fi")
		m.state = hotspotError
		return m, nil
	}

	cfg := network.HotspotConfig{
		SSID:     ssid,
		Password: pwd,
		Band:     m.band,
		Iface:    iface,
	}

	m.state = hotspotStarting
	return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
		return startHotspotAction(cfg)
	})
}

func (m *HotspotModel) updateFocus() {
	m.ssidInput.Blur()
	m.passwordInput.Blur()
	switch m.focusField {
	case 0:
		m.ssidInput.Focus()
	case 1:
		m.passwordInput.Focus()
	}
}

func (m HotspotModel) updateFocusedField(msg tea.KeyMsg) (HotspotModel, tea.Cmd) {
	switch m.focusField {
	case 0:
		var cmd tea.Cmd
		m.ssidInput, cmd = m.ssidInput.Update(msg)
		return m, cmd
	case 1:
		var cmd tea.Cmd
		m.passwordInput, cmd = m.passwordInput.Update(msg)
		return m, cmd
	default:
		return m, nil
	}
}

// ─── View ─────────────────────────────────────────────────────

func (m HotspotModel) View() string {
	if m.state == hotspotScanning {
		return theme.CardStyle.Render(
			theme.CardTitleStyle.Render("🔥 Hotspot") + "\n\n"+
				m.spinner.View()+" Verificando estado...",
		)
	}

	if m.state == hotspotStarting {
		return theme.CardStyle.Render(
			theme.CardTitleStyle.Render("🔥 Hotspot")+"\n\n"+
				m.spinner.View()+" Iniciando hotspot...",
		)
	}

	if m.state == hotspotStopping {
		return theme.CardStyle.Render(
			theme.CardTitleStyle.Render("🔥 Hotspot")+"\n\n"+
				m.spinner.View()+" Deteniendo hotspot...",
		)
	}

	return theme.CardStyle.Render(m.renderContent())
}

func (m HotspotModel) renderContent() string {
	title := theme.CardTitleStyle.Render("🔥 Hotspot")

	// ── Si el hotspot está activo → panel de control ──
	if m.status != nil && m.status.Active {
		statusBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.ColorSecondary).
			Padding(1, 2).
			Width(50).
			Render(
				lipgloss.JoinVertical(lipgloss.Left,
					theme.SuccessStyle.Render("●  Hotspot activo"),
					"",
					theme.LabelStyle.Render("SSID:")+"  "+theme.ValueStyle.Render(m.status.SSID),
					theme.LabelStyle.Render("Banda:")+"  "+theme.ValueStyle.Render(formatBand(m.status.Band)),
					theme.LabelStyle.Render("Interfaz:")+"  "+theme.ValueStyle.Render(m.status.Iface),
					theme.LabelStyle.Render("Clientes:")+"  "+theme.ValueStyle.Render(fmt.Sprintf("%d", m.status.Clients)),
				),
			)

		actions := lipgloss.NewStyle().Foreground(theme.ColorSubtle).Render(
			"  Enter: Detener hotspot   r: Refrescar   Tab: Navegar",
		)

		return title + "\n\n" + statusBox + "\n\n" + actions
	}

	// ── Si está inactivo → formulario de creación ──
	configBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColorBorder).
		Padding(1, 2).
		Width(54)

	var fields []string

	// SSID
	ssidLabel := "  "
	if m.focusField == 0 {
		ssidLabel = lipgloss.NewStyle().Foreground(theme.ColorPrimary).Render("▸ ")
	}
	fields = append(fields, ssidLabel+theme.LabelStyle.Render("SSID:")+" "+m.ssidInput.View())

	// Password
	pwdLabel := "  "
	if m.focusField == 1 {
		pwdLabel = lipgloss.NewStyle().Foreground(theme.ColorPrimary).Render("▸ ")
	}
	fields = append(fields, pwdLabel+theme.LabelStyle.Render("Password:")+" "+m.passwordInput.View())

	// Banda
	bandLabel := "  "
	if m.focusField == 2 {
		bandLabel = lipgloss.NewStyle().Foreground(theme.ColorPrimary).Render("▸ ")
	}
	bandWidget := theme.LabelStyle.Render("Banda:") + "  "
	if m.band == "bg" {
		bandWidget += theme.ActiveTabStyle.Render(" 2.4GHz ") + "  " + theme.TabStyle.Render(" 5GHz ")
	} else {
		bandWidget += theme.TabStyle.Render(" 2.4GHz ") + "  " + theme.ActiveTabStyle.Render(" 5GHz ")
	}
	fields = append(fields, bandLabel+bandWidget)

	// Botón Start
	startLabel := "  "
	if m.focusField == 3 {
		startLabel = lipgloss.NewStyle().Foreground(theme.ColorPrimary).Render("▸ ")
	}
	startBtn := lipgloss.NewStyle().
		Background(theme.ColorSecondary).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 3).
		Render("🔥 Iniciar Hotspot")
	if m.focusField == 3 {
		startBtn = lipgloss.NewStyle().
			Background(theme.ColorPrimary).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 3).
			Render("🔥 Iniciar Hotspot")
	}
	fields = append(fields, startLabel+startBtn)

	form := configBox.Render(lipgloss.JoinVertical(lipgloss.Left, fields...))

	help := lipgloss.NewStyle().Foreground(theme.ColorSubtle).Render(
		"  Tab/↓: Navegar  ←/→: Cambiar banda  Enter en Start: Iniciar  r: Refrescar",
	)

	var toast string
	if m.state == hotspotDone || m.state == hotspotError {
		style := theme.SuccessStyle
		icon := "✓"
		if m.toastErr != nil {
			style = theme.ErrorStyle
			icon = "✗"
		}
		toast = "\n" + theme.ToastStyle.Render(style.Render(icon+" "+m.toast))
	}

	return title + "\n\n" + form + "\n\n" + help + toast
}

func formatBand(b string) string {
	switch b {
	case "bg":
		return "2.4 GHz"
	case "a":
		return "5 GHz"
	default:
		return b
	}
}
