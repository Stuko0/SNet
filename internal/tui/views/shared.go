package views

import (
	"strings"

	"github.com/Stuko0/SNet/internal/network"
	"github.com/Stuko0/SNet/internal/tui/theme"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// visibleWidth devuelve el ancho visible real de un string ya renderizado
// (ignora secuencias ANSI). lipgloss.Width es +2 para strings que terminan en
// reset, por eso se usa ansi.PrintableRuneWidth sobre el texto sin escapes.
func visibleWidth(s string) int {
	return lipgloss.Width(s)
}

// cortarPorAncho recorta un string a max columnas visibles sin partir secuencias
// de escape. Si hubo que recortar, agrega una elipsis dentro del ancho pedido.
func cortarPorAncho(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if visibleWidth(s) <= max {
		return s
	}
	if max == 1 {
		return "…"
	}
	var b strings.Builder
	used := 0
	limit := max - 1 // reservar 1 columna para la elipsis
	for _, r := range s {
		w := lipgloss.Width(string(r))
		if used+w > limit {
			break
		}
		b.WriteRune(r)
		used += w
	}
	return b.String() + "…"
}

// cortarPorAnchoPlano recorta texto sin estilos (SSID, nombres, contraseñas).
func cortarPorAnchoPlano(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max == 1 {
		return "…"
	}
	return string(r[:max-1]) + "…"
}

// msgError traduce un error del backend a un mensaje para el usuario.
//
// La capa network ya devuelve mensajes en español y sin secretos; este wrapper
// existe para que las vistas tengan un único punto de entrada y no haya que
// recordar importar network en cada una.
func msgError(err error) string {
	if err == nil {
		return ""
	}
	return network.UserMessage(err)
}

// datosError devuelve true si hubo un timeout, para poder ofrecer "reintentar"
// en vez de un error genérico.
func esTimeout(err error) bool {
	if err == nil {
		return false
	}
	return network.UserMessage(err) == network.MsgTimeout
}

// nuevaFilaInput arma una fila de formulario con cursor, label alineado a la
// derecha (14 columnas, igual que theme.LabelStyle) y su textinput.
func nuevaFilaInput(focused bool, label string, input textinput.Model) string {
	cursor := "  "
	if focused {
		cursor = lipgloss.NewStyle().Foreground(theme.ColorPrimary).Render("▸ ")
	}
	texto := theme.LabelStyle.Render(label+":") + " " + input.View()
	return cortarPorAncho(cursor+texto, 72)
}

// wrapTexto parte texto sin estilos en líneas de a lo sumo `ancho` columnas,
// cortando por palabra cuando se puede. Las palabras más largas que el ancho
// (p. ej. una PSK de 64 caracteres sin espacios) se parten igual, porque no hay
// otra opción.
func wrapTexto(s string, ancho int) string {
	if ancho <= 1 {
		return s
	}

	var salida []string
	for _, parrafo := range strings.Split(s, "\n") {
		palabras := strings.Fields(parrafo)
		if len(palabras) == 0 {
			salida = append(salida, "")
			continue
		}
		linea := ""
		for _, p := range palabras {
			candidata := p
			if linea != "" {
				candidata = linea + " " + p
			}
			if len([]rune(candidata)) <= ancho {
				linea = candidata
				continue
			}
			if linea != "" {
				salida = append(salida, linea)
			}
			// Palabra sola demasiado larga: se parte en trozos.
			resto := []rune(p)
			for len(resto) > ancho {
				salida = append(salida, string(resto[:ancho]))
				resto = resto[ancho:]
			}
			linea = string(resto)
		}
		if linea != "" {
			salida = append(salida, linea)
		}
	}
	return strings.Join(salida, "\n")
}
