package tui

import (
	"github.com/Stuko0/SNet/internal/tui/theme"
	"github.com/Stuko0/SNet/internal/tui/views"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model es el modelo principal de la aplicación
type Model struct {
	width     int
	height    int
	ready     bool
	activeTab int

	// Sub-modelos por vista
	dashboard views.DashboardModel
	wifiList  views.WifiListModel
	saved     views.SavedModel
	vpnList   views.VPNListModel
	hotspot   views.HotspotModel

	// Overlays
	showHelp   bool
	quitting   bool
}

func NewModel() Model {
	return Model{
		dashboard: views.NewDashboard(),
		wifiList:  views.NewWifiList(),
		saved:     views.NewSaved(),
		vpnList:   views.NewVPNList(),
		hotspot:   views.NewHotspot(),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.dashboard.Init(),
		m.wifiList.Init(),
		m.saved.Init(),
		m.vpnList.Init(),
		m.hotspot.Init(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		if m.showHelp {
			switch {
			case keyMatches(msg, Keys.Help), keyMatches(msg, Keys.Escape):
				m.showHelp = false
				return m, nil
			}
		}

		if m.quitting {
			switch {
			case keyMatches(msg, Keys.Quit):
				return m, tea.Quit
			default:
				m.quitting = false
				return m, nil
			}
		}

		switch {
		case keyMatches(msg, Keys.Quit):
			m.quitting = true
			return m, nil

		case keyMatches(msg, Keys.Help):
			m.showHelp = true
			return m, nil

		case keyMatches(msg, Keys.Tab):
			m.activeTab = (m.activeTab + 1) % len(theme.TabTitles)
			return m, nil

		case keyMatches(msg, Keys.ShiftTab):
			m.activeTab = (m.activeTab - 1 + len(theme.TabTitles)) % len(theme.TabTitles)
			return m, nil

		case keyMatches(msg, Keys.Refresh):
			return m, func() tea.Msg {
				return views.RefreshCmd()
			}
		}
	}

	// Pasar mensaje a la vista activa
	var cmd tea.Cmd
	switch m.activeTab {
	case 0:
		m.dashboard, cmd = m.dashboard.Update(msg)
	case 1:
		m.wifiList, cmd = m.wifiList.Update(msg)
	case 2:
		m.saved, cmd = m.saved.Update(msg)
	case 3:
		m.vpnList, cmd = m.vpnList.Update(msg)
	case 4:
		m.hotspot, cmd = m.hotspot.Update(msg)
	}

	return m, cmd
}

func (m Model) View() string {
	if !m.ready {
		return lipgloss.NewStyle().Width(80).Height(24).
			Align(lipgloss.Center, lipgloss.Center).
			Render("Inicializando nmtui...")
	}

	// Header
	header := lipgloss.JoinHorizontal(lipgloss.Center,
		theme.LogoStyle.Render("📡 nmtui"),
		theme.TitleStyle.Render("v0.1.0"),
		lipgloss.NewStyle().Width(m.width-18).Render(""),
		theme.LabelStyle.Render("NetworkManager TUI"),
	)

	// Tabs
	tabRow := renderTabs(m.activeTab)

	// Contenido
	var content string
	switch m.activeTab {
	case 0:
		content = m.dashboard.View()
	case 1:
		content = m.wifiList.View()
	case 2:
		content = m.saved.View()
	case 3:
		content = m.vpnList.View()
	case 4:
		content = m.hotspot.View()
	}

	contentWidth := m.width - 4
	if contentWidth < 40 {
		contentWidth = 40
	}
	content = lipgloss.NewStyle().Width(contentWidth).Render(content)

	// Footer
	footer := renderFooter(m.quitting, m.showHelp, m.activeTab)

	// Help overlay
	if m.showHelp {
		helpView := renderHelp(m.width, m.height)
		return helpView
	}

	// Quit confirmation
	if m.quitting {
		quitView := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.ColorDanger).
			Padding(1, 2).
			Width(40).
			Render(
				lipgloss.JoinVertical(lipgloss.Center,
					lipgloss.NewStyle().Foreground(theme.ColorDanger).Bold(true).Render("¿Salir de nmtui?"),
					"",
					"Presiona "+keyStyle("Ctrl+q")+" para confirmar",
					"o cualquier otra tecla para cancelar.",
				),
			)
		return lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			quitView,
		)
	}

	return theme.AppStyle.Render(
		lipgloss.JoinVertical(lipgloss.Top,
			header,
			tabRow,
			content,
			footer,
		),
	)
}

func renderTabs(active int) string {
	var tabs []string
	for i, title := range theme.TabTitles {
		if i == active {
			tabs = append(tabs, theme.ActiveTabStyle.Render(title))
		} else {
			tabs = append(tabs, theme.TabStyle.Render(title))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
}

func renderFooter(quitting bool, showHelp bool, activeTab int) string {
	if showHelp {
		return theme.FooterStyle.Width(100).
			Render("Presiona ? o Esc para cerrar la ayuda")
	}
	if quitting {
		return ""
	}

	// Footer keys según vista activa
	var keys []struct{ key, desc string }

	switch activeTab {
	case 0: // Dashboard
		keys = []struct{ key, desc string }{
			{"Tab", "Navegar"},
			{"r", "Refresh"},
			{"?", "Ayuda"},
			{"Ctrl+q", "Salir"},
		}
	case 1: // Wi-Fi
		keys = []struct{ key, desc string }{
			{"↑/↓", "Navegar"},
			{"Enter", "Conectar"},
			{"r", "Buscar"},
			{"Tab", "Siguiente"},
			{"?", "Ayuda"},
		}
	case 2: // Saved
		keys = []struct{ key, desc string }{
			{"↑/↓", "Navegar"},
			{"Enter", "Conectar"},
			{"d", "Eliminar"},
			{"p", "Password"},
			{"r", "Refrescar"},
			{"?", "Ayuda"},
		}
	case 4: // Hotspot
		keys = []struct{ key, desc string }{
			{"Tab/↓", "Navegar"},
			{"Enter", "Iniciar/Detener"},
			{"←/→", "Banda"},
			{"r", "Refrescar"},
			{"?", "Ayuda"},
		}
	default:
		keys = []struct{ key, desc string }{
			{"Tab", "Navegar"},
			{"r", "Refresh"},
			{"?", "Ayuda"},
			{"Ctrl+q", "Salir"},
		}
	}

	var parts []string
	for _, k := range keys {
		parts = append(parts,
			theme.FooterKeyStyle.Render(" "+k.key+" "),
			theme.FooterDescStyle.Render(" "+k.desc+" "),
		)
	}
	return theme.FooterStyle.Width(100).Render(
		lipgloss.JoinHorizontal(lipgloss.Center, parts...),
	)
}

func renderHelp(width, height int) string {
	helpContent := theme.HelpStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Foreground(theme.ColorPrimary).Bold(true).Render("Ayuda de nmtui"),
			"",
			"  Navegación:",
			helpRow("Tab / Shift+Tab", "Cambiar entre vistas"),
			helpRow("↑ / ↓ / k / j", "Navegar listas"),
			helpRow("Enter", "Seleccionar / conectar"),
			"",
			"  Acciones:",
			helpRow("r", "Refrescar estado / escanear"),
			helpRow("e", "Editar conexión"),
			helpRow("d", "Eliminar conexión"),
			helpRow("Ctrl+n", "Nuevo (red / VPN / hotspot)"),
			"",
			"  General:",
			helpRow("?", "Mostrar esta ayuda"),
			helpRow("Ctrl+q / q", "Salir"),
			"",
			lipgloss.NewStyle().Foreground(theme.ColorSubtle).Render("Presiona ? o Esc para cerrar"),
		),
	)
	return lipgloss.Place(width, height,
		lipgloss.Center, lipgloss.Center,
		helpContent,
	)
}

func helpRow(key, desc string) string {
	return "  " + theme.HelpKeyStyle.Render(key) + "  " + theme.HelpDescStyle.Render(desc)
}

func keyStyle(k string) string {
	return theme.HelpKeyStyle.Render(k)
}

func keyMatches(msg tea.KeyMsg, binding key.Binding) bool {
	return key.Matches(msg, binding)
}
