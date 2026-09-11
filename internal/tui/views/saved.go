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
	// keyByRow mapea cada índice de fila de la tabla al nombre real de la
	// conexión (la columna visible va truncada, no sirve para operar sobre nmcli).
	keyByRow []string
	// feedback del panel de contraseña (copiar al portapapeles)
	pwdMsg    string
	pwdMsgErr bool

	// búsqueda y orden: con 34 conexiones la lista era inmanejable sin filtrar.
	filtro filtroLista
	orden  ordenLista
}

func NewSaved() SavedModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(theme.ColorPrimary)
	return SavedModel{
		state:   savedLoading,
		spinner: s,
		filtro:  nuevoFiltro("Buscar conexión..."),
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
	pwd    string // contraseña recuperada (action == "password")
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
	return connActionMsg{action: "password", name: name, pwd: pwd, err: nil}
}

func (m SavedModel) Update(msg tea.Msg) (SavedModel, tea.Cmd) {
	switch msg := msg.(type) {

	case connListMsg:
		m.state = savedIdle
		if msg.err != nil {
			m.err = msg.err
			m.toast = msgError(msg.err)
			m.state = savedError
			break
		}
		m.conns = msg.conns
		// Se reaplica el filtro/orden vigente para que refrescar no pierda la
		// búsqueda que el usuario tenía activa.
		m = m.rebuildTable()
		m.err = nil
		return m, nil

	case connActionMsg:
		switch msg.action {
		case "connect":
			m.state = savedDone
			if msg.err != nil {
				m.toast = fmt.Sprintf("Error al conectar %s: %s", msg.name, msgError(msg.err))
				m.toastErr = msg.err
			} else {
				m.toast = fmt.Sprintf("✓ Conectado a %s", msg.name)
				m.toastErr = nil

				return m, fetchConnections
			}
		case "delete":
			m.state = savedDone
			if msg.err != nil {
				m.toast = fmt.Sprintf("Error al eliminar %s: %s", msg.name, msgError(msg.err))
				m.toastErr = msg.err
			} else {
				m.toast = fmt.Sprintf("✓ Eliminada: %s", msg.name)
				m.toastErr = nil

				return m, fetchConnections
			}
		case "password":
			m.state = savedShowingPwd
			if msg.err != nil {
				m.toast = msgError(msg.err)
				m.toastErr = msg.err
				m.password = ""
				return m, nil
			}
			m.password = msg.pwd
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
			switch msg.String() {
			case "esc", "ctrl+c":
				m.state = savedIdle
				m.password = ""
				m.pwdMsg = ""
				m.pwdMsgErr = false
			case "c":
				// No romper el panel: se deja la contraseña en el portapapeles
				// y se confirma en el hint.
				if m.password != "" {
					if err := clipboardCopy(m.password); err != nil {
						m.pwdMsg = "✗ No se pudo copiar: " + err.Error()
						m.pwdMsgErr = true
					} else {
						m.pwdMsg = "✓ Contraseña copiada al portapapeles"
						m.pwdMsgErr = false
					}
				}
			}
			return m, nil
		}
		if m.state == savedIdle {
			return m.handleIdleKey(msg)
		}
	}

	return m, nil
}

// handleFiltroKey procesa el teclado mientras el buscador está abierto.
//
// Enter y Esc cierran el buscador conservando/limpiando el filtro: mantener el
// filtro al cerrar permite navegar y conectar sobre los resultados filtrados.
func (m SavedModel) handleFiltroKey(msg tea.KeyMsg) (SavedModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Cerrar el input pero conservar el filtro aplicado.
		m.filtro.activo = false
		m.filtro.input.Blur()
		m = m.rebuildTable()
		return m, nil

	case "esc":
		// Cancelar y limpiar el filtro.
		m.filtro.cerrar()
		m = m.rebuildTable()
		return m, nil

	case "ctrl+c":
		return m, nil

	case "up":
		m.table.MoveUp(1)
		return m, nil

	case "down":
		m.table.MoveDown(1)
		return m, nil

	default:
		// Escribir re-filtra en vivo: con listas largas conviene ver el
		// resultado mientras se tipea.
		antes := m.filtro.consulta()
		var cmd tea.Cmd
		m.filtro.input, cmd = m.filtro.input.Update(msg)
		if m.filtro.consulta() != antes {
			m = m.rebuildTable()
		}
		return m, cmd
	}
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
	// Con el buscador abierto, todo el teclado va al input (incluidas las
	// letras que en idle son atajos, como d/e/p/r).
	if m.filtro.activo {
		return m.handleFiltroKey(msg)
	}

	sel := m.getSelectedName()

	switch msg.String() {
	case "/":
		m.filtro.abrir()
		return m, nil

	case "s":
		m.orden = m.orden.siguiente()
		m = m.rebuildTable()
		return m, nil
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

	case "esc":
		// Esc limpia un filtro que quedó aplicado tras cerrar el buscador con
		// Enter; si no hay filtro, no hace nada (no debe cerrar la app).
		if !m.filtro.vacio() {
			m.filtro.cerrar()
			m = m.rebuildTable()
		}
		return m, nil

	default:
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		return m, cmd
	}
}

func (m SavedModel) getSelectedName() string {
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.keyByRow) {
		return ""
	}
	return m.keyByRow[idx]
}

// rebuildTable reaplica filtro y orden sobre las conexiones cargadas.
// Se llama tras cada cambio de filtro/orden, no en cada tecla de escritura de
// la tabla (eso reconstruía el modelo y perdía la selección).
func (m SavedModel) rebuildTable() SavedModel {
	visibles := filtrarYOrdenar(
		m.conns,
		m.filtro,
		m.orden,
		func(c network.Connection) []string { return []string{c.Name, c.Type, c.Device} },
		compararConexiones,
	)
	m.table, m.keyByRow = buildConnTable(visibles)
	return m
}

// compararConexiones ordena según el criterio activo.
func compararConexiones(a, b network.Connection, orden ordenLista) bool {
	switch orden {
	case ordenNombre:
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	case ordenTipo:
		if a.Type != b.Type {
			return a.Type < b.Type
		}
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	case ordenEstado:
		// Activas primero; dentro de cada grupo, por nombre.
		if a.Active != b.Active {
			return a.Active
		}
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	}
	return false
}

// conexionesVisibles es la cantidad de filas tras filtrar.
func (m SavedModel) conexionesVisibles() int { return len(m.keyByRow) }

func (m SavedModel) getSelectedType() string {
	selName := m.getSelectedName()
	for _, c := range m.conns {
		if c.Name == selName {
			return c.Type
		}
	}
	return ""
}

func (m SavedModel) selectedIsWiFi() bool {
	selName := m.getSelectedName()
	for _, c := range m.conns {
		if c.Name == selName {
			return c.Type == "wifi" || c.Type == "802-11-wireless"
		}
	}
	return false
}

type EditConnectionMsg struct {
	Name string
	Type string
}

// savedCardWidth es el ancho interior de la tarjeta de Guardadas. Todas las
// ramas de View() deben usar el MISMO ancho: si los estados "Cargando..."
// (que miden su ancho natural, ~29 cols) no lo fijan, la tarjeta salta de
// tamaño al llegar los datos y la estructura se percibe rota.
const savedCardWidth = 72

// savedCard envuelve un contenido en la tarjeta con el ancho uniforme.
func savedCard(contenido string) string {
	return theme.CardStyle.
		Width(savedCardWidth).
		Render(contenido)
}

func (m SavedModel) View() string {
	if m.state == savedLoading {
		label := "Cargando conexiones..."
		return savedCard(
			theme.CardTitleStyle.Render("󰆓 Conexiones Guardadas") + "\n\n" +
				m.spinner.View() + " " + label,
		)
	}

	if m.state == savedConnecting {
		return savedCard(
			theme.CardTitleStyle.Render("󰆓 Conexiones Guardadas") + "\n\n" +
				m.spinner.View() + " Conectando a " + m.getSelectedName() + "...",
		)
	}

	if m.state == savedConfirmDelete {
		return m.renderDeleteConfirm()
	}

	if m.state == savedShowingPwd {
		view := m.renderTableView()
		// JoinVertical alinea al ancho del bloque más ancho y, si se fuerza con
		// Width(), el redondeo del borde parte la línea en dos. MaxWidth acota
		// sin estirar.
		pwd := m.renderPasswordBox()
		bloque := lipgloss.JoinVertical(lipgloss.Left, view, "", pwd)
		return lipgloss.NewStyle().MaxWidth(innerWidth).Render(bloque)
	}
	if m.state == savedDone || m.state == savedError {
		view := m.renderTableView()
		toast := m.renderToast()
		return lipgloss.JoinVertical(lipgloss.Top, view, toast)
	}

	return m.renderTableView()
}

const (
	// ancho útil del panel de contenido: 80 cols - 4 de margen del AppStyle.
	innerWidth = 76
	// ancho interior del panel de contraseña.
	pwdBoxWidth = 60
	// suma de las columnas de la tabla de conexiones (usada también para
	// reservar el alto del estado vacío y mantener el mismo ancho).
	savedTableWidth = 28 + 10 + 14 + 6 + 6
)

func (m SavedModel) renderPasswordBox() string {
	nombre := m.getSelectedName()
	if len([]rune(nombre)) > pwdBoxWidth-6 {
		nombre = cortarPorAnchoPlano(nombre, pwdBoxWidth-6)
	}

	header := lipgloss.NewStyle().Foreground(theme.ColorPrimary).Bold(true).
		Render("🔑 Contraseña de " + nombre)

	var body string
	if m.password == "" {
		body = theme.WarningStyle.Render("Sin contraseña guardada para esta conexión")
	} else {
		// Se parte en varias líneas para que encaje dentro del panel (una PSK
		// de 63+ chars desbordaba el borde).
		body = theme.ValueStyle.Render(wrapTexto(m.password, pwdBoxWidth-10))
	}

	hint := theme.OutputHintStyle.Render("  p: Ver contraseña   c: Copiar   Esc: Cerrar")
	if m.pwdMsg != "" {
		estilo := theme.SuccessStyle
		if m.pwdMsgErr {
			estilo = theme.ErrorStyle
		}
		hint = estilo.Render("  " + m.pwdMsg)
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColorPrimary).
		Padding(1, 3).
		Width(pwdBoxWidth).
		Render(
			lipgloss.JoinVertical(lipgloss.Left,
				header,
				"",
				body,
				"",
				hint,
			),
		)
}

func (m SavedModel) renderTableView() string {
	title := theme.CardTitleStyle.Render("󰆓 Conexiones Guardadas")

	// El contador refleja el filtro activo y el criterio de orden.
	stats := fmt.Sprintf("  %d conexiones", len(m.conns))
	if !m.filtro.vacio() {
		stats += fmt.Sprintf(" · %d de %d", m.conexionesVisibles(), len(m.conns))
	}
	// El orden se muestra aparte para no pegarlo al conteo ("34 conexionesorden").
	if ind := indicadorOrden(m.orden); ind != "" {
		stats += " · " + ind
	}
	stats += "    "

	// Buscador abierto: se muestra el input debajo del título.
	var buscador string
	if m.filtro.activo {
		buscador = "\n  " + theme.CardTitleStyle.Render("Buscar: ") + m.filtro.input.View()
	}

	var body string
	if len(m.conns) == 0 {
		// Ojo: theme.LabelStyle tiene Width(14)+Align(Right), pensado para
		// etiquetas de campo. Usarlo acá estiraba la tarjeta a ~174 columnas,
		// así que el mensaje usa estilos sin ancho fijo.
		mensaje := lipgloss.NewStyle().Foreground(theme.ColorText).
			Render("No hay conexiones guardadas") + "\n" +
			theme.OutputHintStyle.Render("Presiona 'r' para refrescar")
		body = "\n" + theme.TableStyle.Render(
			lipgloss.Place(savedTableWidth, 3, lipgloss.Center, lipgloss.Center, mensaje))
	} else if m.conexionesVisibles() == 0 {
		// Hay conexiones pero ninguna coincide: no confundir con "no hay nada".
		mensaje := lipgloss.NewStyle().Foreground(theme.ColorWarning).
			Render("Ninguna conexión coincide con la búsqueda") + "\n" +
			theme.OutputHintStyle.Render("Esc para limpiar el filtro")
		body = "\n" + theme.TableStyle.Render(
			lipgloss.Place(savedTableWidth, 3, lipgloss.Center, lipgloss.Center, mensaje))
	} else {
		body = "\n" + theme.TableStyle.Render(m.table.View())
	}

	helpText := "  ↑/↓: Navegar  Enter: Conectar  /: Buscar  s: Ordenar  e: Editar  d: Eliminar  p: Contraseña  r: Refrescar"
	if m.filtro.activo {
		helpText = "  Escribí para filtrar   Enter: Mantener filtro   Esc: Limpiar   ↑/↓: Navegar"
	}
	if m.state == savedShowingPwd {
		helpText = "  ↑/↓: Navegar  p: Ver contraseña  c: Copiar  Esc: Cerrar el panel"
	}
	if m.state == savedConfirmDelete {
		helpText = "  Enter/y: Confirmar  Esc/n: Cancelar"
	}
	// Ojo con la concatenación: `body` ya termina en "\n" (viene de la tabla o
	// del estado vacío) pero el `help` debe ir en su propia línea. Si la línea
	// de atajos excede el ancho de la tarjeta, envolverla en vez de estirar el
	// borde (medido: 98 columnas de atajos => tarjeta de 174 en un panel de 76).
	help := theme.OutputHintStyle.Render(wrapTexto(helpText, savedCardWidth-6))

	return savedCard(
		title + buscador + "\n" +
			stats + "\n" +
			body + "\n" +
			help,
	)
}

// indicadorOrden describe el criterio activo sólo cuando no es el natural.
func indicadorOrden(o ordenLista) string {
	if o == ordenNatural {
		return ""
	}
	return "orden: " + o.String()
}

func (m SavedModel) renderDeleteConfirm() string {
	sel := m.getSelectedName()
	if len([]rune(sel)) > 40 {
		sel = cortarPorAnchoPlano(sel, 40)
	}
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
				theme.OutputHintStyle.Render("  Enter/y: Confirmar  Esc/n: Cancelar"),
			),
		)

	// La barra de ayuda va arriba (la tabla real debajo) para que no cambie de
	// posición al abrir/cerrar diálogos.
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

func buildConnTable(conns []network.Connection) (table.Model, []string) {
	// El ancho total debe caber en savedCardWidth (72) menos el padding de
	// CardStyle. table.Model añade ~5 columnas de separadores y padding, así
	// que las columnas suman 60: 60+5=65 < 72-2*2=68. Antes sumaban 64 y la
	// tabla desbordaba, partiendo la línea separadora y dejando el indicador
	// de activo (●) en una fila propia.
	columns := []table.Column{
		{Title: "Nombre", Width: 26},
		{Title: "Tipo", Width: 9},
		{Title: "Dispositivo", Width: 13},
		{Title: "Auto", Width: 5},
		{Title: "", Width: 5},
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
	var keys []string
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
			cortarPorAnchoPlano(c.Name, 26),
			connTypeIcon(c.Type),
			cortarPorAnchoPlano(c.Device, 13),
			auto,
			status,
		})
		keys = append(keys, c.Name)
	}
	t.SetRows(rows)
	return t, keys
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
