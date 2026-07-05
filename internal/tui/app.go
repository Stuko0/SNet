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

	editor *views.EditorModel

	// Overlays
	showHelp bool
	quitting bool
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

	case views.EditConnectionMsg:
		editor := views.NewEditor(msg.Name, msg.Type)
		loadCmd := editor.LoadCmd()
		m.editor = &editor
		return m, loadCmd

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
			return m, func() tea.Msg { return views.RefreshMsg{} }

		case keyMatches(msg, Keys.ShiftTab):
			m.activeTab = (m.activeTab - 1 + len(theme.TabTitles)) % len(theme.TabTitles)
			return m, func() tea.Msg { return views.RefreshMsg{} }

		case keyMatches(msg, Keys.Refresh):
			return m, func() tea.Msg {
				return views.RefreshMsg{}
			}
		}

		if m.editor != nil {
			if keyMatches(msg, Keys.Escape) && !m.editor.IsDone() {
				m.editor = nil
				return m, nil
			}
			var cmd tea.Cmd
			updatedEditor, cmd := m.editor.Update(msg)
			m.editor = &updatedEditor
			if m.editor.IsDone() && keyMatches(msg, Keys.Escape) {
				m.editor = nil
				return m, func() tea.Msg { return views.RefreshMsg{} } // Actualizado a RefreshMsg
			}
			return m, cmd
		}

	case tea.MouseMsg:
		if msg.Action == tea.MouseActionRelease && msg.Button == tea.MouseButtonLeft {
			if msg.Y == 1 { // Click en la fila de pestañas (Y=1 debido al header en Y=0)
				x := 1 // Padding izquierdo de AppStyle
				for i, t := range theme.TabTitles {
					var w int
					if i == m.activeTab {
						w = lipgloss.Width(theme.ActiveTabStyle.Render(t))
					} else {
						w = lipgloss.Width(theme.TabStyle.Render(t))
					}
					if msg.X >= x && msg.X < x+w {
						if m.activeTab != i {
							m.activeTab = i
							return m, func() tea.Msg { return views.RefreshMsg{} }
						}
						break
					}
					x += w
				}
			}
		}
	}

	var cmds []tea.Cmd

	// Mensajes locales: solo a la vista activa
	isLocal := false
	switch msg.(type) {
	case tea.KeyMsg, views.RefreshMsg:
		isLocal = true
	}

	if isLocal {
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
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	} else {
		// Mensajes globales (resultados en segundo plano, spinners): a todas las vistas
		var cmd tea.Cmd
		m.dashboard, cmd = m.dashboard.Update(msg)
		if cmd != nil { cmds = append(cmds, cmd) }
		
		m.wifiList, cmd = m.wifiList.Update(msg)
		if cmd != nil { cmds = append(cmds, cmd) }
		
		m.saved, cmd = m.saved.Update(msg)
		if cmd != nil { cmds = append(cmds, cmd) }
		
		m.vpnList, cmd = m.vpnList.Update(msg)
		if cmd != nil { cmds = append(cmds, cmd) }
		
		m.hotspot, cmd = m.hotspot.Update(msg)
		if cmd != nil { cmds = append(cmds, cmd) }
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if !m.ready {
		return lipgloss.NewStyle().Width(80).Height(24).
			Align(lipgloss.Center, lipgloss.Center).
			Render("Inicializando SNet...")
	}

	header := lipgloss.JoinHorizontal(lipgloss.Center,
		theme.LogoStyle.Render("󰣺 SNet"),
		theme.TitleStyle.Render("v0.1.0"),
		lipgloss.NewStyle().Width(m.width-18).Render(""),
		theme.LabelStyle.Render("NetworkManager TUI"),
	)

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

	if m.editor != nil {
		editorView := lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			m.editor.View(),
		)
		return editorView
	}

	footer := renderFooter(m.quitting, m.showHelp, m.activeTab)

	if m.showHelp {
		helpView := renderHelp(m.width, m.height)
		return helpView
	}

	if m.quitting {
		quitView := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.ColorDanger).
			Padding(1, 2).
			Width(40).
			Render(
				lipgloss.JoinVertical(lipgloss.Center,
					lipgloss.NewStyle().Foreground(theme.ColorDanger).Bold(true).Render("¿Salir de SNet?"),
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
	case 0:
		keys = []struct{ key, desc string }{
			{"Tab", "Navegar"},
			{"r", "Refresh"},
			{"?", "Ayuda"},
			{"Ctrl+q", "Salir"},
		}
	case 1:
		keys = []struct{ key, desc string }{
			{"↑/↓", "Navegar"},
			{"Enter", "Conectar"},
			{"r", "Buscar"},
			{"Tab", "Siguiente"},
			{"?", "Ayuda"},
		}
	case 2:
		keys = []struct{ key, desc string }{
			{"↑/↓", "Navegar"},
			{"Enter", "Conectar"},
			{"d", "Eliminar"},
			{"e", "Editar"},
			{"p", "Password"},
			{"r", "Refrescar"},
			{"?", "Ayuda"},
		}
	case 3:
		keys = []struct{ key, desc string }{
			{"↑/↓", "Navegar"},
			{"Enter", "Conectar/Desconectar"},
			{"e", "Editar"},
			{"Ctrl+n", "Nueva VPN"},
			{"r", "Refrescar"},
			{"?", "Ayuda"},
		}
	case 4:
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
			lipgloss.NewStyle().Foreground(theme.ColorPrimary).Bold(true).Render("Ayuda de SNet"),
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
