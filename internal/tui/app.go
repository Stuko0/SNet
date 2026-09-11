package tui

import (
	"strings"

	"github.com/Stuko0/SNet/internal/config"
	"github.com/Stuko0/SNet/internal/tui/theme"
	"github.com/Stuko0/SNet/internal/tui/views"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const (
	// appPadding es el padding horizontal de theme.AppStyle (0, 1).
	appPadding = 1
	// maxContentWidth acota la columna de contenido: en terminales anchas
	// (150+ columnas) las tarjetas se centran en vez de estirarse o quedar
	// pegadas a la izquierda.
	maxContentWidth = 100
	// minContentWidth es el ancho mínimo con el que el layout sigue siendo
	// legible (el diseño asume ~80 columnas).
	minContentWidth = 76
)

// Model es el modelo principal de la aplicación
type Model struct {
	width     int
	height    int
	ready     bool
	activeTab int

	// cfg es la configuración persistente (pestaña activa al cerrar, etc.).
	// Antes el paquete config existía pero nadie lo llamaba: la pestaña se
	// perdía en cada arranque.
	cfg *config.Config

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
	// La config se carga acá y no puede fallar: LoadConfig cae a defaults ante
	// archivo ausente o corrupto, para que una config rota no impida arrancar.
	cfg, err := config.LoadConfig()
	if err != nil || cfg == nil {
		cfg = config.Default()
	}

	return Model{
		cfg:       cfg,
		activeTab: cfg.LastTab,
		dashboard: views.NewDashboard(),
		wifiList:  views.NewWifiList(),
		saved:     views.NewSaved(),
		vpnList:   views.NewVPNList(),
		hotspot:   views.NewHotspot(),
	}
}

// saveConfig persiste el estado actual. Los errores se ignoran a propósito: no
// poder guardar preferencias no debe romper la sesión ni mostrar un error.
func (m Model) saveConfig() {
	if m.cfg == nil {
		return
	}
	m.cfg.LastTab = m.activeTab
	_ = config.SaveConfig(m.cfg)
}

// SaveConfigHook devuelve una función que persiste el estado actual sin
// necesitar el modelo. main() la usa al recibir SIGTERM/SIGINT: en ese camino
// Bubble Tea no ejecuta Update, así que la pestaña se perdería.
func (m Model) SaveConfigHook() func() {
	return func() { m.saveConfig() }
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
	// ==========================================================
	// Editor overlay: intercept ALL messages while active
	// (spinner ticks, settings load results, key presses, etc.)
	// ==========================================================
	if m.editor != nil {
		return m.updateEditor(msg)
	}

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
				// Guardar antes de salir: es el momento en que la pestaña
				// activa queda registrada para el próximo arranque.
				m.saveConfig()
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
			if m.helpBlocked() {
				// La vista activa está mostrando un formulario/modal: la '?'
				// debe llegarle (o ignorarse), no abrir el overlay de ayuda.
				break
			}
			m.showHelp = true
			return m, nil

		case keyMatches(msg, Keys.Tab):
			m.activeTab = (m.activeTab + 1) % len(theme.TabTitles)
			m.saveConfig()
			return m, func() tea.Msg { return views.RefreshMsg{} }

		case keyMatches(msg, Keys.ShiftTab):
			m.activeTab = (m.activeTab - 1 + len(theme.TabTitles)) % len(theme.TabTitles)
			m.saveConfig()
			return m, func() tea.Msg { return views.RefreshMsg{} }

		case keyMatches(msg, Keys.Refresh):
			return m, func() tea.Msg {
				return views.RefreshMsg{}
			}
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
							m.saveConfig()
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
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		m.wifiList, cmd = m.wifiList.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		m.saved, cmd = m.saved.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		m.vpnList, cmd = m.vpnList.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		m.hotspot, cmd = m.hotspot.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// helpBlocked indica si la vista activa está en un sub-estado (formulario,
// modal, input) en el que '?' debe llegarle a ella en vez de abrir la ayuda
// global, que tapaba la pantalla y descartaba lo que se estaba escribiendo.
func (m Model) helpBlocked() bool {
	switch m.activeTab {
	case 1: // Wi-Fi: pidiendo contraseña
		return m.wifiList.IsPasswordState()
	case 3: // VPN: asistente de alta
		return m.vpnList.IsAdding()
	}
	return false
}

// updateEditor maneja TODOS los mensajes mientras el editor overlay está activo
func (m Model) updateEditor(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Escape sin cambios → cerrar editor
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMatches(keyMsg, Keys.Escape) && !m.editor.IsDone() {
			m.editor = nil
			return m, nil
		}
		// Quit desde el editor
		if keyMatches(keyMsg, Keys.Quit) {
			m.quitting = true
			return m, nil
		}
	}

	// Pasar el mensaje al editor
	var cmd tea.Cmd
	updatedEditor, cmd := m.editor.Update(msg)
	m.editor = &updatedEditor

	// Si el editor terminó (save ok/error) y fue la última tecla Escape → cerrar
	if m.editor.IsDone() {
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMatches(keyMsg, Keys.Escape) {
			m.editor = nil
			return m, func() tea.Msg { return views.RefreshMsg{} }
		}
		// cualquier otra tecla: volver a editing (descartar toast)
	}

	return m, cmd
}

func (m Model) View() string {
	if !m.ready {
		return lipgloss.NewStyle().Width(80).Height(24).
			Align(lipgloss.Center, lipgloss.Center).
			Render("Inicializando SNet...")
	}

	tabRow := renderTabs(m.activeTab)

	// Columna de contenido: se acota a maxContentWidth y se CENTRA en la
	// terminal. Sin esto, en una ventana ancha (kitty 1000px ≈ 150 cols) cada
	// tarjeta quedaba pegada al borde izquierdo con un ancho fijo (Estado 29,
	// Hotspot 30, Guardadas 84) y la vista parecía rota.
	contentWidth := m.width - 2*appPadding
	if contentWidth > maxContentWidth {
		contentWidth = maxContentWidth
	}
	if contentWidth < minContentWidth {
		contentWidth = minContentWidth
	}
	// La fila de pestañas mide su ancho natural y se alinea a la izquierda; si
	// la columna es más angosta que ella, el centrado la desplaza y el tab row
	// queda desalineado respecto a la tarjeta.
	if tabAncho := lipgloss.Width(renderTabs(0)); tabAncho > contentWidth {
		contentWidth = tabAncho
	}

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

	// Toda tarjeta se renderiza dentro de la misma columna: así el borde
	// izquierdo y el derecho alinean entre tabs y entre estados de carga.
	content = lipgloss.NewStyle().
		Width(contentWidth).
		MaxWidth(contentWidth).
		Render(content)

	// Header alineado a la misma columna que el contenido.
	header := m.renderHeader(contentWidth)
	footer := renderFooter(m.quitting, m.showHelp, m.activeTab, contentWidth)

	// Cada línea se normaliza al ancho de la columna: JoinVertical alinea a la
	// IZQUIERDA, así que todos los bloques deben medir exactamente
	// contentWidth o el centrado posterior los desplaza y el layout queda
	// desalineado.
	normalizar := func(s string) string {
		lineas := strings.Split(s, "\n")
		for i, l := range lineas {
			if w := lipgloss.Width(l); w < contentWidth {
				lineas[i] = l + strings.Repeat(" ", contentWidth-w)
			}
		}
		return strings.Join(lineas, "\n")
	}

	inner := lipgloss.JoinVertical(lipgloss.Left,
		normalizar(header),
		normalizar(tabRow),
		normalizar(content),
		normalizar(footer),
	)

	// En terminales anchas, la columna completa se centra: si no, todo queda
	// pegado al borde izquierdo y la vista parece rota (reportado en kitty a
	// ~150 columnas).
	if m.width > contentWidth+2*appPadding {
		inner = lipgloss.Place(m.width-2*appPadding, 0,
			lipgloss.Center, lipgloss.Top, inner)
	}

	if m.editor != nil {
		return lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			m.editor.View(),
		)
	}

	if m.showHelp {
		return renderHelp(m.width, m.height)
	}

	if m.quitting {
		return lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			m.renderQuitConfirm(),
		)
	}

	return theme.AppStyle.Render(inner)
}

// renderHeader arma la barra superior con el logo a la izquierda y la etiqueta
// a la derecha, usando el mismo ancho que la columna de contenido.
func (m Model) renderHeader(ancho int) string {
	izq := lipgloss.JoinHorizontal(lipgloss.Center,
		theme.LogoStyle.Render("󰣺 SNet"),
		theme.TitleStyle.Render("v0.1.0"),
	)
	der := theme.LabelStyle.Render("NetworkManager TUI")
	// LabelStyle tiene Width(14), insuficiente para esta etiqueta: se usa un
	// estilo sin ancho fijo para no estirar el header.
	der = lipgloss.NewStyle().Foreground(theme.ColorSubtle).Render("NetworkManager TUI")

	espacio := ancho - lipgloss.Width(izq) - lipgloss.Width(der)
	if espacio < 1 {
		espacio = 1
	}
	return lipgloss.JoinHorizontal(lipgloss.Center,
		izq,
		lipgloss.NewStyle().Width(espacio).Render(""),
		der,
	)
}

func (m Model) renderQuitConfirm() string {
	return lipgloss.NewStyle().
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

func renderFooter(quitting bool, showHelp bool, activeTab int, ancho int) string {
	if showHelp {
		return theme.FooterStyle.Width(ancho).
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
			{"x", "Desconectar"},
			{"w", "Wi-Fi on/off"},
			{"r", "Refrescar"},
			{"Tab", "Navegar"},
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
			{"/", "Buscar"},
			{"s", "Ordenar"},
			{"e", "Editar"},
			{"d", "Eliminar"},
			{"p", "Contraseña"},
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

	// Se van agregando atajos mientras entren en el ancho disponible. Antes se
	// volcaban todos y FooterStyle.Width(ancho) los envolvía a una segunda
	// línea, dejando el footer desalineado respecto a la tarjeta.
	//
	// Se reserva 1 columna para que el ancho final nunca exceda `ancho`, y si
	// hubo que omitir alguno se indica con "…" al final.
	const sep = " "
	usado := 0
	var parts []string
	for _, k := range keys {
		bloque := theme.FooterKeyStyle.Render(" "+k.key+" ") +
			theme.FooterDescStyle.Render(" "+k.desc+" ")
		w := lipgloss.Width(bloque) + len(sep)
		if usado+w > ancho {
			break
		}
		parts = append(parts, bloque)
		usado += w
	}

	linea := lipgloss.JoinHorizontal(lipgloss.Center, parts...)
	if n := len(parts); n < len(keys) {
		linea += theme.FooterDescStyle.Render("…")
	}

	// Se recorta al ancho de la columna: sin esto el footer podía medir más que
	// la tarjeta y desbordar el borde de la terminal.
	linea = ansi.Truncate(linea, ancho, "…")
	if w := lipgloss.Width(linea); w < ancho {
		linea += strings.Repeat(" ", ancho-w)
	}
	return theme.FooterStyle.Render(linea)
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
			helpRow("/", "Buscar en la lista (Esc limpia)"),
			helpRow("s", "Cambiar el orden de la lista"),
			helpRow("x", "Desconectar la conexión activa (Estado)"),
			helpRow("w", "Encender/apagar la radio Wi-Fi (Estado)"),
			helpRow("e", "Editar conexión"),
			helpRow("d", "Eliminar conexión"),
			helpRow("p", "Ver contraseña guardada (Wi-Fi)"),
			helpRow("c", "Copiar la contraseña mostrada"),
			helpRow("Ctrl+n", "Nuevo (red / VPN / hotspot)"),
			"",
			"  Notas:",
			helpRow("Guardadas", "con 'e' se abre el editor de la conexión"),
			helpRow("VPN", "Enter avanza campo; en el último, crea (Ctrl+s siempre)"),
			helpRow("Hotspot", "←/→ cambia la banda; requiere 8+ caracteres"),
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
