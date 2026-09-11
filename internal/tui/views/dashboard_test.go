package views

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Stuko0/SNet/internal/network"
)

// estadoListo devuelve un dashboard ya cargado con una conexión Wi-Fi activa.
func estadoListo() DashboardModel {
	m := NewDashboard()
	m.loading = false
	si := true
	m.wifiEnabled = &si
	m.state = &network.NetworkState{
		Connectivity: network.ConnectivityFull,
		ActiveSSID:   "Casa",
		ActiveDevice: "wlan0",
		ActiveType:   "wifi",
	}
	return m
}

// TestDashboardXDesconecta: la 'x' debe disparar Disconnect sobre el
// dispositivo activo. Antes esa acción no existía en la UI.
func TestDashboardXDesconecta(t *testing.T) {
	m := estadoListo()

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if cmd == nil {
		t.Fatal("'x' no disparó ningún comando")
	}
	if !updated.busy {
		t.Error("debería quedar en busy mientras corre la acción")
	}
	if actual := updated.dispositivoDesconectable(); actual != "wlan0" {
		t.Errorf("dispositivo = %q, se esperaba wlan0", actual)
	}

	// El comando debe producir el mensaje de acción correcto.
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("se esperaba tea.BatchMsg, se obtuvo %T", msg)
	}
	encontrado := false
	for _, c := range batch {
		if c == nil {
			continue
		}
		if am, ok := c().(dashboardActionMsg); ok {
			encontrado = true
			if am.accion != "disconnect" {
				t.Errorf("acción = %q, se esperaba disconnect", am.accion)
			}
			if am.nombre != "wlan0" {
				t.Errorf("nombre = %q, se esperaba wlan0", am.nombre)
			}
		}
	}
	if !encontrado {
		t.Error("el batch no incluyó la acción de desconectar")
	}
}

// TestDashboardXSinConexionAvisa: sin conexión activa no debe intentar nada.
func TestDashboardXSinConexionAvisa(t *testing.T) {
	m := NewDashboard()
	m.loading = false
	m.state = &network.NetworkState{Connectivity: network.ConnectivityNone}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if cmd != nil {
		t.Error("no debería lanzar comando sin conexión activa")
	}
	if updated.toast == "" {
		t.Error("debería avisar que no hay conexión para desconectar")
	}
	if updated.busy {
		t.Error("no debería quedar busy")
	}
}

// TestDashboardWToggleaRadio: la 'w' cambia el estado de la radio Wi-Fi.
func TestDashboardWToggleaRadio(t *testing.T) {
	m := NewDashboard()
	m.loading = false
	apagado := false
	m.wifiEnabled = &apagado
	// Sin conexión Wi-Fi activa: apagar la radio es seguro.
	m.state = &network.NetworkState{ActiveType: "ethernet"}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	if cmd == nil {
		t.Fatal("'w' no disparó ningún comando")
	}
	if !updated.busy {
		t.Error("debería quedar busy")
	}

	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("se esperaba tea.BatchMsg, se obtuvo %T", msg)
	}
	encontrado := false
	for _, c := range batch {
		if c == nil {
			continue
		}
		if am, ok := c().(dashboardActionMsg); ok {
			encontrado = true
			// Estaba apagado, así que debe encender.
			if am.accion != "radio-on" {
				t.Errorf("acción = %q, se esperaba radio-on", am.accion)
			}
		}
	}
	if !encontrado {
		t.Error("el batch no incluyó la acción de radio")
	}
}

// TestDashboardWNoApagaWiFiConectado: apagar la radio estando conectado por
// Wi-Fi cortaría la conexión, así que debe avisar en vez de hacerlo.
func TestDashboardWNoApagaWiFiConectado(t *testing.T) {
	m := estadoListo()

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	if cmd != nil {
		t.Error("no debería apagar la radio estando conectado por Wi-Fi")
	}
	if updated.toast == "" {
		t.Error("debería avisar por qué no se apaga")
	}
	if updated.busy {
		t.Error("no debería quedar busy")
	}
	if updated.wifiEnabled == nil || !*updated.wifiEnabled {
		t.Error("el estado de la radio no debería cambiar")
	}
}

// TestDashboardRadioMsgActualizaEstado: el mensaje de radio refresca el panel.
func TestDashboardRadioMsgActualizaEstado(t *testing.T) {
	m := NewDashboard()
	if m.wifiEnabled != nil {
		t.Fatal("arranca en nil (no consultado)")
	}
	m, _ = m.Update(wifiRadioMsg{enabled: false})
	if m.wifiEnabled == nil || *m.wifiEnabled {
		t.Error("debería quedar en false")
	}
	m, _ = m.Update(wifiRadioMsg{enabled: true})
	if m.wifiEnabled == nil || !*m.wifiEnabled {
		t.Error("debería quedar en true")
	}
}

// TestDashboardActionErrorMuestraMensaje: si la acción falla, el toast debe
// mostrarlo en español (vía msgError) y no quedar trabado en busy.
func TestDashboardActionErrorMuestraMensaje(t *testing.T) {
	m := estadoListo()
	m.busy = true

	err := &network.NmcliError{
		Args:   []string{"device", "disconnect", "wlan0"},
		Stderr: "Error: Not authorized to control networking.",
	}
	updated, _ := m.Update(dashboardActionMsg{accion: "disconnect", nombre: "wlan0", err: err})

	if updated.busy {
		t.Error("no debería quedar busy tras el error")
	}
	if updated.toastErr == nil {
		t.Error("debería registrar el error")
	}
	if updated.toast != network.MsgSinPermisos {
		t.Errorf("toast = %q, se esperaba el mensaje de permisos", updated.toast)
	}
}

// TestDashboardActionExitosaRefresca: al desconectar bien, debe refrescar el
// estado para que el panel deje de mostrar la red.
func TestDashboardActionExitosaRefresca(t *testing.T) {
	m := estadoListo()
	m.busy = true

	updated, cmd := m.Update(dashboardActionMsg{accion: "disconnect", nombre: "wlan0"})
	if updated.busy {
		t.Error("no debería quedar busy")
	}
	if updated.toast == "" {
		t.Error("debería confirmar la desconexión")
	}
	if !updated.loading {
		t.Error("debería volver a loading para refrescar el estado")
	}
	if cmd == nil {
		t.Error("debería disparar el refresco")
	}
}

// TestDashboardIgnoraTeclasMientrasCarga: no aceptar acciones si está cargando.
func TestDashboardIgnoraTeclasMientrasCarga(t *testing.T) {
	m := estadoListo()
	m.loading = true
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if cmd != nil {
		t.Error("no debería aceptar acciones mientras carga")
	}
}

// TestDashboardViewMuestraRadioYAtajos: el panel debe mostrar el estado de la
// radio y los atajos nuevos.
func TestDashboardViewMuestraRadioYAtajos(t *testing.T) {
	m := estadoListo()
	v := m.View()
	for _, want := range []string{"Radio Wi-Fi:", "activado", "x: Desconectar", "w:"} {
		if !contiene(v, want) {
			t.Errorf("la vista no contiene %q", want)
		}
	}

	// Apagada se muestra como desactivada.
	apagado := false
	m.wifiEnabled = &apagado
	if v := m.View(); !contiene(v, "desactivado") {
		t.Error("con la radio apagada debería decir 'desactivado'")
	}

	// Sin consultar, no se muestra la fila.
	m.wifiEnabled = nil
	if v := m.View(); contiene(v, "Radio Wi-Fi:") {
		t.Error("sin dato de radio no debería mostrarse la fila")
	}
}

func contiene(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
