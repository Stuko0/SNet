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

	// wifiEnabled refleja el estado de la radio Wi-Fi. Es nil cuando todavía no
	// se consultó, para poder distinguir "apagado" de "no sé".
	wifiEnabled *bool

	// toast de la última acción (desconectar / radio), con su error si falló.
	toast    string
	toastErr error
	// busy mientras hay una acción en curso, para ignorar teclas repetidas.
	busy bool
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
		fetchWiFiRadio,
	)
}

// Messages
type stateMsg struct {
	state *network.NetworkState
	err   error
}

// wifiRadioMsg transporta el estado de la radio Wi-Fi.
type wifiRadioMsg struct {
	enabled bool
	err     error
}

// dashboardActionMsg es el resultado de desconectar o de togglear la radio.
type dashboardActionMsg struct {
	accion string // "disconnect", "radio-on", "radio-off"
	nombre string // dispositivo o interfaz afectada
	err    error
}

func fetchState() tea.Msg {
	state, err := network.GetActiveConnection()
	return stateMsg{state: state, err: err}
}

func fetchWiFiRadio() tea.Msg {
	return wifiRadioMsg{enabled: network.IsWiFiEnabled()}
}

// disconnectDevice baja la conexión activa del dispositivo indicado.
func disconnectDevice(device string) tea.Msg {
	err := network.Disconnect(device)
	return dashboardActionMsg{accion: "disconnect", nombre: device, err: err}
}

// toggleWiFiRadio apaga o enciende la radio Wi-Fi.
func toggleWiFiRadio(habilitar bool) tea.Msg {
	err := network.RadioToggleWiFi(habilitar)
	accion := "radio-off"
	if habilitar {
		accion = "radio-on"
	}
	return dashboardActionMsg{accion: accion, err: err}
}

type RefreshMsg struct{}

func RefreshCmd() tea.Msg {
	return RefreshMsg{}
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

	case wifiRadioMsg:
		habilitado := msg.enabled
		m.wifiEnabled = &habilitado
		return m, nil

	case dashboardActionMsg:
		m.busy = false
		if msg.err != nil {
			m.toast = msgError(msg.err)
			m.toastErr = msg.err
			return m, nil
		}
		m.toastErr = nil
		switch msg.accion {
		case "disconnect":
			m.toast = "Desconectado de " + msg.nombre
			// Refrescar para que el panel refleje la desconexión.
			m.loading = true
			return m, fetchState
		case "radio-on":
			m.toast = "Wi-Fi activado"
		case "radio-off":
			m.toast = "Wi-Fi desactivado"
		}
		return m, tea.Batch(fetchState, fetchWiFiRadio)

	case RefreshMsg:
		m.loading = true
		m.toast = ""
		m.toastErr = nil
		return m, tea.Batch(fetchState, fetchWiFiRadio)

	case tea.KeyMsg:
		return m.handleKey(msg)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	default:
		return m, nil
	}
}

// handleKey procesa las acciones propias del dashboard: desconectar la conexión
// activa y encender/apagar la radio Wi-Fi.
//
// Antes estas dos operaciones existían en la capa network pero no tenían forma
// de invocarse desde la TUI, así que el usuario sólo podía desconectarse desde
// nmcli.
func (m DashboardModel) handleKey(msg tea.KeyMsg) (DashboardModel, tea.Cmd) {
	if m.loading || m.busy {
		return m, nil
	}

	switch msg.String() {
	case "x":
		// Desconectar el dispositivo activo.
		device := m.dispositivoDesconectable()
		if device == "" {
			m.toast = "No hay una conexión activa para desconectar"
			m.toastErr = fmt.Errorf("sin conexión activa")
			return m, nil
		}
		m.busy = true
		m.toast = ""
		m.toastErr = nil
		return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
			return disconnectDevice(device)
		})

	case "w":
		// Toggle de la radio Wi-Fi.
		if m.wifiEnabled == nil {
			// Todavía no sabemos el estado: consultarlo primero.
			return m, fetchWiFiRadio
		}
		if m.state == nil || m.state.ActiveType == "wifi" {
			// Apagar la radio estando conectado por Wi-Fi corta la conexión:
			// se avisa en vez de hacerlo a ciegas.
			if *m.wifiEnabled {
				m.toast = "Estás conectado por Wi-Fi. Desconectate primero (x) para apagar la radio."
				m.toastErr = fmt.Errorf("wifi activo")
				return m, nil
			}
		}
		m.busy = true
		m.toast = ""
		m.toastErr = nil
		habilitar := !*m.wifiEnabled
		return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
			return toggleWiFiRadio(habilitar)
		})
	}

	return m, nil
}

// dispositivoDesconectable devuelve el dispositivo de la conexión activa, o ""
// si no hay nada que desconectar.
func (m DashboardModel) dispositivoDesconectable() string {
	if m.state == nil {
		return ""
	}
	return m.state.ActiveDevice
}

func (m DashboardModel) View() string {
	if m.loading {
		return theme.CardStyle.Render(
			theme.CardTitleStyle.Render("󰣺 Estado de Red") + "\n\n" +
				m.spinner.View() + " Obteniendo estado...",
		)
	}

	if m.err != nil {
		return theme.CardStyle.Render(
			theme.CardTitleStyle.Render("󰣺 Estado de Red") + "\n\n" +
				theme.ErrorStyle.Render("Error al obtener estado: "+msgError(m.err)),
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

	if state.Connectivity == network.ConnectivityFull || state.Connectivity == network.ConnectivityLimited || state.ActiveSSID != "" {
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

	content += "\n" + theme.LabelStyle.Render("VPN activas:") + " "
	if len(state.ActiveVPNs) > 0 {
		var vpnList []string
		for _, vpn := range state.ActiveVPNs {
			vpnList = append(vpnList, theme.SuccessStyle.Render("● "+vpn))
		}
		content += strings.Join(vpnList, ", ") + "\n"
	} else {
		content += theme.ValueStyle.Render("❌ Ninguna") + "\n"
	}

	content += "\n" + theme.LabelStyle.Render("Dispositivo:") + " " +
		theme.ValueStyle.Render(state.ActiveDevice) + "\n"

	// Estado de la radio Wi-Fi: antes no se mostraba ni se podía cambiar.
	if m.wifiEnabled != nil {
		estado := theme.SuccessStyle.Render("activado")
		if !*m.wifiEnabled {
			estado = lipgloss.NewStyle().Foreground(theme.ColorMuted).Render("desactivado")
		}
		content += theme.LabelStyle.Render("Radio Wi-Fi:") + " " + estado + "\n"
	}

	// Atajos propios de esta vista.
	content += "\n" + theme.OutputHintStyle.Render(
		"  x: Desconectar   w: Encender/apagar Wi-Fi   r: Refrescar")

	var toast string
	if m.toast != "" {
		estilo := theme.SuccessStyle
		icono := "✓"
		if m.toastErr != nil {
			estilo = theme.ErrorStyle
			icono = "✗"
		}
		toast = "\n" + theme.ToastStyle.Render(
			estilo.Render(icono+" "+wrapTexto(m.toast, innerWidth-6)))
	}

	return theme.CardStyle.Render(
		theme.CardTitleStyle.Render("󰣺 Estado de Red") + "\n" +
			content + toast,
	)
}

func deviceIcon(t string) string {
	switch t {
	case "wifi":
		return "󰤨"
	case "ethernet":
		return "🔌"
	case "tun":
		return "󰒄"
	case "bridge":
		return "🔗"
	default:
		return "🔧"
	}
}
