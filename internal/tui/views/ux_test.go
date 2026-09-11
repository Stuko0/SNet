package views

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Stuko0/SNet/internal/network"
)

// TestEscribirPasswordConRNoDisparaRefresh reproduce el bug reportado: al
// escribir la contraseña de una red, la 'r' se interpretaba como "refrescar" y
// la vista saltaba a estado de escaneo perdiendo la contraseña.
func TestEscribirPasswordConRNoDisparaRefresh(t *testing.T) {
	m := NewWifiList()
	m.state = wifiIdle
	m.networks = []network.WiFiNetwork{{SSID: "Casa", Security: "WPA2"}}
	m.table = buildTable(m.networks)

	// Enter sobre una red con contraseña -> estado de password
	var cmd tea.Cmd
	m, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Log("enter lanzó comando de conexión directa (red conocida)")
	}
	if m.state != wifiPassword {
		t.Fatalf("estado = %v, se esperaba wifiPassword", m.state)
	}

	// Escribir "roberto" (tiene 'r') letra por letra.
	for _, r := range "roberto" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	if m.state != wifiPassword {
		t.Fatalf("la 'r' cambió el estado a %v; debería seguir en wifiPassword", m.state)
	}
	if got := m.password.Value(); got != "roberto" {
		t.Fatalf("contraseña capturada = %q, se esperaba roberto", got)
	}
}

func TestPasswordEscVuelveAIdle(t *testing.T) {
	m := NewWifiList()
	m.state = wifiPassword
	m.password.Focus()
	m.password.SetValue("secreto")

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.state != wifiIdle {
		t.Fatalf("estado = %v, se esperaba wifiIdle", m.state)
	}
	if m.password.Value() != "" {
		t.Errorf("la contraseña debería limpiarse al cancelar")
	}
}

// TestPanelPasswordSobreviveTeclas reproduce el segundo bug: el panel de
// contraseña se cerraba con CUALQUIER tecla, incluida la 'y' que el usuario
// escribe al copiar resaltando con el mouse.
func TestPanelPasswordSobreviveTeclas(t *testing.T) {
	m := NewSaved()
	m.state = savedIdle
	conns := []network.Connection{{Name: "Casa", Type: "wifi"}}
	m.conns = conns
	m.table, m.keyByRow = buildConnTable(conns)

	// simular llegada de la contraseña
	m, _ = m.Update(connActionMsg{action: "password", name: "Casa", pwd: strings.Repeat("a", 63)})
	if m.state != savedShowingPwd {
		t.Fatalf("estado = %v, se esperaba savedShowingPwd", m.state)
	}

	// escribir 'y' (copiar con mouse) y 'r' no deben cerrar el panel
	for _, r := range []rune{'y', 'r', 'q'} {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		if m.state != savedShowingPwd {
			t.Fatalf("la tecla %q cerró el panel (estado %v)", string(r), m.state)
		}
	}

	// Esc sí cierra
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.state != savedIdle {
		t.Fatalf("Esc debería cerrar el panel, estado = %v", m.state)
	}
}

// TestGuardadasStateVacioNoRompe verifica que la tarjeta sin conexiones se
// renderiza sin desbordar el ancho útil ni colapsar el layout.
//
// Nota: el ancho se mide DESPUÉS de un tea.WindowSizeMsg, como en producción.
// Sin tamaño, table.Model usa su ancho interno por defecto y el test mide una
// tabla que nunca llega a la pantalla (falso positivo de 180+ columnas).
func TestGuardadasStateVacioNoRompe(t *testing.T) {
	m := NewSaved()
	m.state = savedIdle
	m.table, m.keyByRow = buildConnTable(nil)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	v := m.View()
	if v == "" {
		t.Fatal("View() vacío")
	}
	if !strings.Contains(v, "No hay conexiones guardadas") {
		t.Error("falta el mensaje de estado vacío")
	}
	// Ancho útil del panel: 80 - 4 de margen del AppStyle. Se permite 1 col de
	// holgura en la línea de borde.
	for _, linea := range strings.Split(v, "\n") {
		if w := visibleWidth(linea); w > innerWidth+1 {
			t.Errorf("línea de %d columnas desborda el panel de %d: %q", w, innerWidth, linea)
		}
	}
}

// TestPanelPasswordNoDesborda comprueba que una PSK larga no rompe el borde.
func TestPanelPasswordNoDesborda(t *testing.T) {
	m := NewSaved()
	m.state = savedIdle
	conns := []network.Connection{{Name: "UnaRedConNombreBastanteLargoParaProbar", Type: "wifi"}}
	m.conns = conns
	m.table, m.keyByRow = buildConnTable(conns)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = m.Update(connActionMsg{action: "password", pwd: strings.Repeat("a", 63)})

	for _, linea := range strings.Split(m.View(), "\n") {
		if w := visibleWidth(linea); w > innerWidth+1 {
			t.Errorf("línea de %d columnas desborda el panel de %d: %q", w, innerWidth, linea)
		}
	}
}

// TestAltaVPNAvanzaConEnterYConfirma: Enter ya no debe crear la VPN desde el
// primer campo.
func TestAltaVPNAvanzaConEnterYConfirma(t *testing.T) {
	m := NewVPNList()
	m.state = vpnAddType
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	if m.state != vpnAddConfig {
		t.Fatalf("estado = %v, se esperaba vpnAddConfig", m.state)
	}

	m.addName.SetValue("oficina")
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state == vpnDone {
		t.Fatal("Enter en el primer campo creó la VPN sin confirmar")
	}
	if m.addField != 1 {
		t.Errorf("Enter debería avanzar al campo 1, está en %d", m.addField)
	}

	// llegar al último campo y confirmar
	for m.addField < m.addFieldCount()-1 {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != vpnConfirmAdd {
		t.Fatalf("estado = %v, se esperaba vpnConfirmAdd", m.state)
	}
	if !m.IsAdding() {
		t.Error("IsAdding() debería ser true durante la confirmación")
	}

	// Esc vuelve a editar sin perder los datos
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.state != vpnAddConfig {
		t.Fatalf("Esc debería volver a vpnAddConfig, estado = %v", m.state)
	}
	if m.addName.Value() != "oficina" {
		t.Errorf("Esc perdió los datos del formulario: %q", m.addName.Value())
	}
}

func TestHotspotTabAcotadoConHotspotActivo(t *testing.T) {
	m := NewHotspot()
	m.state = hotspotIdle
	m.status = &network.HotspotConfig{Active: true, SSID: "MiAP"}
	if got := m.focusFieldCount(); got != 2 {
		t.Fatalf("focusFieldCount() = %d, se esperaba 2 con hotspot activo", got)
	}

	// Con hotspot activo Tab no mueve el foco (el formulario no se dibuja).
	m.focusField = 1
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyTab})
	if m.focusField != 1 {
		t.Errorf("con hotspot activo Tab no debe mover el foco, está en %d", m.focusField)
	}

	// Sin hotspot, Tab vuelve a navegar (2 = toggle de banda).
	m.status = &network.HotspotConfig{Active: false}
	m.focusField = 1
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyTab})
	if m.focusField != 2 {
		t.Errorf("sin hotspot Tab debería avanzar a 2, está en %d", m.focusField)
	}

	// Y desde el último campo vuelve a 0.
	m.focusField = 3
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyTab})
	if m.focusField != 0 {
		t.Errorf("Tab desde el último campo debería volver a 0, está en %d", m.focusField)
	}
}
