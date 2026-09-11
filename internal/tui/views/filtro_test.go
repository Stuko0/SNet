package views

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Stuko0/SNet/internal/network"
)

func connsDePrueba() []network.Connection {
	return []network.Connection{
		{Name: "Zeta", Type: "wifi", Device: "wlan0"},
		{Name: "alfa", Type: "ethernet", Device: "enp0s3"},
		{Name: "Beta:5G", Type: "wifi", Device: "wlan0", Active: true},
		{Name: "tailscale0", Type: "tun", Device: "tailscale0"},
	}
}

func savedListo() SavedModel {
	m := NewSaved()
	m.state = savedIdle
	m.conns = connsDePrueba()
	return m.rebuildTable()
}

func escribir(m SavedModel, texto string) SavedModel {
	for _, r := range texto {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return m
}

// TestFiltroSeAbreConSlash: con 34 conexiones hacía falta un buscador.
func TestFiltroSeAbreConSlash(t *testing.T) {
	m := savedListo()
	if m.filtro.activo {
		t.Fatal("el buscador no debería estar abierto al inicio")
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !m.filtro.activo {
		t.Error("'/' debería abrir el buscador")
	}
}

// TestFiltroReduceLasFilas: escribir filtra en vivo.
func TestFiltroReduceLasFilas(t *testing.T) {
	m := savedListo()
	total := m.conexionesVisibles()
	if total != 4 {
		t.Fatalf("sin filtro deberían verse 4, se ven %d", total)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = escribir(m, "wifi")

	// Zeta y Beta:5G son wifi; el filtro matchea por nombre/tipo/device.
	if got := m.conexionesVisibles(); got != 2 {
		t.Errorf("con 'wifi' se esperaban 2 coincidencias, hay %d (%v)", got, m.keyByRow)
	}
	for _, n := range m.keyByRow {
		if n != "Zeta" && n != "Beta:5G" {
			t.Errorf("aparece %q, que no debería coincidir con 'wifi'", n)
		}
	}
}

// TestFiltroEsInsensibleAMayusculas.
func TestFiltroEsInsensibleAMayusculas(t *testing.T) {
	m := savedListo()
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = escribir(m, "ALFA")
	if got := m.conexionesVisibles(); got != 1 {
		t.Errorf("'ALFA' debería matchear 'alfa', hay %d", got)
	}
	if len(m.keyByRow) > 0 && m.keyByRow[0] != "alfa" {
		t.Errorf("coincidencia = %q, se esperaba alfa", m.keyByRow[0])
	}
}

// TestFiltroMatcheaPorNombreYDispositivo: buscar "wlan0" debe encontrar por
// dispositivo, no sólo por nombre.
func TestFiltroMatcheaPorDispositivo(t *testing.T) {
	m := savedListo()
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = escribir(m, "enp0s3")
	if got := m.conexionesVisibles(); got != 1 {
		t.Errorf("'enp0s3' debería matchear 1, hay %d", got)
	}
}

// TestFiltroConDosPuntosNoRompe: el nombre real puede tener ':' (el escape de
// nmcli), y el filtro debe encontrarlo igual.
func TestFiltroConDosPuntosNoRompe(t *testing.T) {
	m := savedListo()
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = escribir(m, "beta:5g")
	if got := m.conexionesVisibles(); got != 1 {
		t.Fatalf("'beta:5g' debería matchear 1, hay %d (%v)", got, m.keyByRow)
	}
	if m.keyByRow[0] != "Beta:5G" {
		t.Errorf("coincidencia = %q, se esperaba Beta:5G", m.keyByRow[0])
	}
}

// TestFiltroEnterConservaEscLimpia.
func TestFiltroEnterConservaEscLimpia(t *testing.T) {
	m := savedListo()
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = escribir(m, "wifi")

	// Enter cierra el input pero conserva el filtro: se puede navegar y
	// conectar sobre lo filtrado.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.filtro.activo {
		t.Error("Enter debería cerrar el input")
	}
	if got := m.conexionesVisibles(); got != 2 {
		t.Errorf("Enter debería conservar el filtro (2), hay %d", got)
	}

	// Esc limpia todo.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.filtro.activo {
		t.Error("Esc debería cerrar el input")
	}
	if got := m.conexionesVisibles(); got != 4 {
		t.Errorf("Esc debería limpiar el filtro (4), hay %d", got)
	}
}

// TestFiltroAtrapaTeclasDeAtajo: con el buscador abierto, 'd'/'e'/'r' deben
// escribirse, no disparar acciones destructivas.
func TestFiltroAtrapaTeclasDeAtajo(t *testing.T) {
	m := savedListo()
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	for _, r := range []rune{'d', 'e', 'r'} {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	if m.state != savedIdle {
		t.Errorf("estado = %v; 'd' abrió el diálogo de borrado en vez de escribir", m.state)
	}
	if got := m.filtro.input.Value(); got != "der" {
		t.Errorf("el input tiene %q, se esperaba escribir 'der'", got)
	}
}

// TestOrdenCiclaYOrdenaPorNombre.
func TestOrdenCiclaYOrdenaPorNombre(t *testing.T) {
	m := savedListo()

	// Natural: el orden de entrada.
	if m.keyByRow[0] != "Zeta" {
		t.Errorf("orden natural debería empezar en Zeta, empieza en %q", m.keyByRow[0])
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if m.orden != ordenNombre {
		t.Fatalf("'s' debería pasar a ordenNombre, está en %v", m.orden)
	}
	// Insensible a mayúsculas: alfa, Beta:5G, tailscale0, Zeta
	esperado := []string{"alfa", "Beta:5G", "tailscale0", "Zeta"}
	for i, want := range esperado {
		if i < len(m.keyByRow) && m.keyByRow[i] != want {
			t.Errorf("ordenNombre[%d] = %q, se esperaba %q", i, m.keyByRow[i], want)
		}
	}
}

// TestOrdenEstadoPoneActivasPrimero.
func TestOrdenEstadoPoneActivasPrimero(t *testing.T) {
	m := savedListo()
	// natural -> nombre -> tipo -> estado
	for i := 0; i < 3; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	}
	if m.orden != ordenEstado {
		t.Fatalf("orden = %v, se esperaba ordenEstado", m.orden)
	}
	if len(m.keyByRow) == 0 || m.keyByRow[0] != "Beta:5G" {
		t.Errorf("la activa (Beta:5G) debería ir primero, lista: %v", m.keyByRow)
	}
}

// TestOrdenCiclaCompleto: tras 4 's' vuelve al natural.
func TestOrdenCiclaCompleto(t *testing.T) {
	m := savedListo()
	for i := 0; i < 4; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	}
	if m.orden != ordenNatural {
		t.Errorf("tras 4 cambios debería volver a natural, está en %v", m.orden)
	}
}

// TestRefrescoConservaElFiltro: recargar la lista no debe perder la búsqueda.
func TestRefrescoConservaElFiltro(t *testing.T) {
	m := savedListo()
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = escribir(m, "wifi")
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Llega la lista de nuevo desde el backend.
	m, _ = m.Update(connListMsg{conns: connsDePrueba()})

	if got := m.conexionesVisibles(); got != 2 {
		t.Errorf("el refresco perdió el filtro: hay %d, se esperaban 2", got)
	}
}

// TestFiltroSinCoincidenciasSeDistingueDeListaVacia: mostrar "ninguna coincide"
// en vez de "no hay conexiones" cuando sí hay pero el filtro no matchea.
func TestFiltroSinCoincidenciasSeDistingueDeListaVacia(t *testing.T) {
	m := savedListo()
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = escribir(m, "zzzz-no-existe")

	if got := m.conexionesVisibles(); got != 0 {
		t.Fatalf("se esperaban 0 coincidencias, hay %d", got)
	}
	v := m.View()
	if !contiene(v, "Ninguna conexión coincide") {
		t.Error("debería avisar que el filtro no matchea, no que no hay conexiones")
	}
	if contiene(v, "No hay conexiones guardadas") {
		t.Error("no debería decir que no hay conexiones si las hay, sólo filtradas")
	}
}

// TestIndicadorNoSePegaAlConteo: sin filtro pero con orden, el contador no
// debe quedar como "34 conexionesorden: tipo" (faltaba el separador).
func TestIndicadorNoSePegaAlConteo(t *testing.T) {
	m := savedListo()
	m.orden = ordenTipo

	v := m.View()
	if contiene(v, "conexionesorden") {
		t.Error("el orden quedó pegado al conteo sin separador")
	}
	if !contiene(v, "conexiones · orden: tipo") {
		t.Errorf("se esperaba 'conexiones · orden: tipo' en la vista")
	}
}

// TestIndicadorOrdenSoloSiNoEsNatural: con orden natural no se ensucia el
// encabezado.
func TestIndicadorOrdenSoloSiNoEsNatural(t *testing.T) {
	m := savedListo()
	if v := m.View(); contiene(v, "orden:") {
		t.Error("con orden natural no debería mostrarse el criterio")
	}
}

// TestGetSelectedNameUsaLaFilaFiltrada: tras filtrar, el cursor debe mapear al
// nombre real de la fila visible (no al índice original).
func TestGetSelectedNameUsaLaFilaFiltrada(t *testing.T) {
	m := savedListo()
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = escribir(m, "tailscale")
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if got := m.getSelectedName(); got != "tailscale0" {
		t.Errorf("getSelectedName() = %q, se esperaba tailscale0", got)
	}
}
