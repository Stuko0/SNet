package views

import (
	"fmt"
	"github.com/Stuko0/SNet/internal/network"
	"github.com/Stuko0/SNet/internal/tui/theme"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DashboardModel es el modelo de la vista de estado
type DashboardModel struct {
	state   *network.NetworkState
	err     error
	loading bool
	spinner spinner.Model
}

func NewDashboard() DashboardModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(theme.ColorPrimary)
	return DashboardModel{
		spinner: s,
		loading: true,
	}
}

func (m DashboardModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		fetchState,
	)
}

// Messages
type stateMsg struct {
	state *network.NetworkState
	err   error
}

func fetchState() tea.Msg {
	state, err := network.GetActiveConnection()
	return stateMsg{state: state, err: err}
}

type refreshMsg struct{}

func RefreshCmd() tea.Msg {
	return refreshMsg{}
}

func (m DashboardModel) Update(msg tea.Msg) (DashboardModel, tea.Cmd) {
	switch msg := msg.(type) {
	case stateMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.state = msg.state
			m.err = nil
		}
		return m, nil

	case refreshMsg:
		m.loading = true
		return m, fetchState

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	default:
		return m, nil
	}
}

func (m DashboardModel) View() string {
	if m.loading {
		return theme.CardStyle.Render(
			theme.CardTitleStyle.Render("📡 Estado de Red") + "\n\n" +
				m.spinner.View() + " Obteniendo estado...",
		)
	}

	if m.err != nil {
		return theme.CardStyle.Render(
			theme.CardTitleStyle.Render("📡 Estado de Red") + "\n\n" +
				theme.ErrorStyle.Render("✗ Error al obtener estado: "+m.err.Error()),
		)
	}

	state := m.state
	if state == nil {
		state = &network.NetworkState{}
	}

	// Estado de conectividad con icono
	var statusIcon, statusText string
	switch state.Connectivity {
	case network.ConnectivityFull:
		statusIcon = theme.StatusOnline.Render("●")
		statusText = "Conectado"
	case network.ConnectivityLimited:
		statusIcon = theme.StatusLimited.Render("●")
		statusText = "Limitado"
	case network.ConnectivityNone:
		statusIcon = theme.StatusOffline.Render("●")
		statusText = "Desconectado"
	default:
		statusIcon = "○"
		statusText = "Desconocido"
	}

	// Barras de señal Wi-Fi
	var signalBars string
	if state.ActiveType == "wifi" {
		bars := ""
		for i := 0; i < 8; i++ {
			threshold := (i + 1) * 12
			if state.SignalStrength >= threshold {
				bars += "█"
			} else {
				bars += "░"
			}
		}
		signalBars = fmt.Sprintf(" %s (%d%%)", bars, state.SignalStrength)
	}

	ipDisplay := state.IPAddress
	if idx := strings.Index(ipDisplay, "/"); idx >= 0 {
		ipDisplay = ipDisplay[:idx]
	}

	// Panel de estado principal
	var content string
	content += fmt.Sprintf("%s  %s\n", statusIcon, theme.ValueStyle.Render(statusText))
	content += "\n"

	if state.Connectivity == network.ConnectivityFull || state.Connectivity == network.ConnectivityLimited {
		content += theme.LabelStyle.Render("Red activa:") + " " +
			theme.ValueStyle.Render(state.ActiveSSID) + signalBars + "\n"
		content += theme.LabelStyle.Render("Tipo:") + " " +
			theme.ValueStyle.Render(deviceIcon(state.ActiveType)+" "+state.ActiveType) + "\n"
		if state.Speed != "" {
			content += theme.LabelStyle.Render("Velocidad:") + " " +
				theme.ValueStyle.Render(state.Speed) + "\n"
		}
		content += theme.LabelStyle.Render("IP local:") + " " +
			theme.ValueStyle.Render(ipDisplay) + "\n"
		if state.Gateway != "" {
			content += theme.LabelStyle.Render("Gateway:") + " " +
				theme.ValueStyle.Render(state.Gateway) + "\n"
		}
		if len(state.DNSServers) > 0 {
			content += theme.LabelStyle.Render("DNS:") + " " +
				theme.ValueStyle.Render(strings.Join(state.DNSServers, ", ")) + "\n"
		}
	} else {
		content += theme.WarningStyle.Render("  No hay conexión activa") + "\n"
	}

	content += "\n" + theme.LabelStyle.Render("VPN activa:") + " "
	if state.IsVPNActive {
		content += theme.SuccessStyle.Render("● "+state.VPNName) + "\n"
	} else {
		content += theme.ValueStyle.Render("❌ Ninguna") + "\n"
	}

	content += "\n" + theme.LabelStyle.Render("Dispositivo:") + " " +
		theme.ValueStyle.Render(state.ActiveDevice)

	return theme.CardStyle.Render(
		theme.CardTitleStyle.Render("📡 Estado de Red") + "\n" +
			content,
	)
}

func deviceIcon(t string) string {
	switch t {
	case "wifi":
		return "📶"
	case "ethernet":
		return "🔌"
	case "tun":
		return "🔒"
	case "bridge":
		return "🔗"
	default:
		return "🔧"
	}
}
