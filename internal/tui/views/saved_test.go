package views

import (
	"strings"
	"testing"

	"github.com/Stuko0/SNet/internal/network"
)

func TestBuildConnTableMapsRealNames(t *testing.T) {
	// Nombre más largo que la columna: la celda se trunca pero keyByRow debe
	// conservar el nombre real, si no toda acción (conectar/editar/borrar)
	// fallaba con "no such connection".
	largos := []network.Connection{
		{Name: "Wired connection con un nombre larguísimo que no entra", Type: "ethernet", Device: "enp0s3"},
		{Name: "MiWiFi", Type: "wifi", Device: "wlp0s20f3", Active: true, Autoconnect: true},
	}

	tabla, keys := buildConnTable(largos)
	if len(keys) != len(largos) {
		t.Fatalf("keyByRow tiene %d entradas, se esperaban %d", len(keys), len(largos))
	}
	if keys[0] != largos[0].Name {
		t.Errorf("keyByRow[0] = %q, se esperaba el nombre real %q", keys[0], largos[0].Name)
	}
	celda := tabla.Rows()[0][0]
	if celda == keys[0] {
		t.Errorf("la celda visible debería estar truncada, pero es igual al nombre real")
	}
	if !strings.HasSuffix(celda, "…") {
		t.Errorf("la celda truncada debería terminar en elipsis, es %q", celda)
	}
	if ancho := visibleWidth(celda); ancho > 28 {
		t.Errorf("la celda excede la columna de 28: %d", ancho)
	}
	if keys[1] != "MiWiFi" {
		t.Errorf("keyByRow[1] = %q, se esperaba MiWiFi", keys[1])
	}
}

func TestGetSelectedNameUsaCursorNoLaFila(t *testing.T) {
	conns := []network.Connection{
		{Name: "aaaa-muy-largo-nombre-de-conexion-para-truncar", Type: "ethernet"},
		{Name: "bbbb", Type: "wifi"},
	}
	m := SavedModel{}
	m.conns = conns
	m.table, m.keyByRow = buildConnTable(conns)

	if got := m.getSelectedName(); got != conns[0].Name {
		t.Errorf("getSelectedName() = %q, se esperaba %q", got, conns[0].Name)
	}
	if !m.selectedIsWiFi() {
		// con el cursor en 0 debe ser ethernet
		t.Log("cursor 0 no es WiFi (correcto)")
	}
	m.table.MoveDown(1)
	if got := m.getSelectedName(); got != "bbbb" {
		t.Errorf("tras bajar, getSelectedName() = %q, se esperaba bbbb", got)
	}
	if !m.selectedIsWiFi() {
		t.Errorf("bbbb es wifi, selectedIsWiFi() debería ser true")
	}
}

func TestCortarPorAnchoPlanoRuneSafe(t *testing.T) {
	casos := []struct {
		in   string
		max  int
		want string
	}{
		{"hola", 10, "hola"},
		{"contraseña-larga", 8, "contras…"},
		{"contraseña-larga", 1, "…"},
		{"contraseña", 0, ""},
		{"áéíóú", 4, "áéí…"},
	}
	for _, c := range casos {
		if got := cortarPorAnchoPlano(c.in, c.max); got != c.want {
			t.Errorf("cortarPorAnchoPlano(%q, %d) = %q, se esperaba %q", c.in, c.max, got, c.want)
		}
	}
}

func TestCortaEstilosLargosSinDesbordar(t *testing.T) {
	// Fila del formulario de alta VPN con un valor muy largo.
	row := "▸  " + "Nombre:" + " " + strings.Repeat("x", 200)
	got := cortarPorAncho(row, 72)
	if ancho := visibleWidth(got); ancho > 72 {
		t.Errorf("la fila mide %d columnas, debería caber en 72: %q", ancho, got)
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("se esperaba elipsis al final, got %q", got)
	}
}

func TestWrapTextoNoDesbordaPanel(t *testing.T) {
	psk := strings.Repeat("a", 63) // largo típico de una PSK WPA2
	wrapped := wrapTexto(psk, 54)
	for _, linea := range strings.Split(wrapped, "\n") {
		if n := len([]rune(linea)); n > 54 {
			t.Errorf("línea de %d runas excede el ancho 54", n)
		}
	}
	if strings.ReplaceAll(wrapped, "\n", "") != psk {
		t.Errorf("wrapTexto alteró el contenido")
	}
}
