package views

import (
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
)

// ordenLista representa el criterio de orden activo.
type ordenLista int

const (
	ordenNatural ordenLista = iota // como lo devuelve nmcli
	ordenNombre
	ordenTipo
	ordenEstado // activas primero
)

func (o ordenLista) String() string {
	switch o {
	case ordenNombre:
		return "nombre"
	case ordenTipo:
		return "tipo"
	case ordenEstado:
		return "estado"
	default:
		return "natural"
	}
}

// siguiente cicla al próximo criterio.
func (o ordenLista) siguiente() ordenLista {
	return (o + 1) % 4
}

// filtroLista es el estado de búsqueda de una lista: el input de texto más la
// consulta ya aplicada.
//
// Existe para no duplicar la lógica entre Guardadas y Wi-Fi: las listas con
// muchas entradas (34 conexiones en una máquina real) eran imposibles de
// navegar sin filtrar.
type filtroLista struct {
	activo bool
	input  textinput.Model
}

func nuevoFiltro(placeholder string) filtroLista {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = 64
	ti.Width = 30
	return filtroLista{input: ti}
}

// abrir activa el modo búsqueda y enfoca el input.
func (f *filtroLista) abrir() {
	f.activo = true
	f.input.Focus()
}

// cerrar desactiva el modo búsqueda y limpia el texto.
func (f *filtroLista) cerrar() {
	f.activo = false
	f.input.SetValue("")
	f.input.Blur()
}

// consulta devuelve el texto buscado, normalizado (minúsculas, sin espacios).
func (f filtroLista) consulta() string {
	return strings.ToLower(strings.TrimSpace(f.input.Value()))
}

// vacio informa si no hay nada que filtrar.
func (f filtroLista) vacio() bool { return f.consulta() == "" }

// coincide informa si alguno de los campos contiene la consulta. Con la
// consulta vacía todo coincide.
func (f filtroLista) coincide(campos ...string) bool {
	q := f.consulta()
	if q == "" {
		return true
	}
	for _, c := range campos {
		if strings.Contains(strings.ToLower(c), q) {
			return true
		}
	}
	return false
}

// filtrarYOrdenar aplica el filtro y el orden a una lista genérica.
//
// `campos` extrae los textos comparables de un elemento, `ordenar` aplica el
// criterio (recibe el índice del criterio para poder elegir la clave).
func filtrarYOrdenar[T any](
	items []T,
	f filtroLista,
	orden ordenLista,
	campos func(T) []string,
	comparar func(a, b T, orden ordenLista) bool,
) []T {
	out := make([]T, 0, len(items))
	for _, it := range items {
		if f.coincide(campos(it)...) {
			out = append(out, it)
		}
	}
	if orden != ordenNatural && comparar != nil {
		sort.SliceStable(out, func(i, j int) bool {
			return comparar(out[i], out[j], orden)
		})
	}
	return out
}

// indicadorFiltro devuelve el texto a mostrar en el encabezado de la tabla
// cuando hay un filtro activo, o "" si no hay nada que informar.
func indicadorFiltro(f filtroLista, mostrados, total int) string {
	if f.vacio() {
		return ""
	}
	return " · " + itoa(mostrados) + "/" + itoa(total) + " coinciden"
}

// itoa evita importar strconv sólo para esto.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
