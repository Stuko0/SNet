package views

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Stuko0/SNet/internal/network"
)

// Estos tests demuestran el objetivo de la fase: ejercitar las vistas SIN nmcli
// ni red real. Antes los tests de UI dependían de la máquina donde corrían.

// conMock instala un MockClient durante el test y restaura el real al terminar.
func conMock(t *testing.T) *network.MockClient {
	t.Helper()
	mock := &network.MockClient{}
	t.Cleanup(network.SetClient(mock))
	return mock
}

// TestVistaSavedCargaDesdeElMock: la lista se puebla desde datos inyectados, sin
// tocar la red.
func TestVistaSavedCargaDesdeElMock(t *testing.T) {
	mock := conMock(t)
	mock.Connections = []network.Connection{
		{Name: "Casa", Type: "wifi", Device: "wlan0", Active: true},
		{Name: "Oficina", Type: "ethernet", Device: "enp0s3"},
	}

	m := NewSaved()
	msg := fetchConnections()
	listMsg, ok := msg.(connListMsg)
	if !ok {
		t.Fatalf("se esperaba connListMsg, se obtuvo %T", msg)
	}
	if listMsg.err != nil {
		t.Fatalf("error inesperado: %v", listMsg.err)
	}

	m, _ = m.Update(listMsg)
	if got := len(m.conns); got != 2 {
		t.Fatalf("se cargaron %d conexiones, se esperaban 2", got)
	}
	if got := m.conexionesVisibles(); got != 2 {
		t.Errorf("filas visibles = %d, se esperaban 2", got)
	}
	if !contiene(m.View(), "Casa") {
		t.Error("la vista no muestra la conexión del mock")
	}
}

// TestVistaSavedErrorDelMockSeTraduce: un error del backend llega traducido.
func TestVistaSavedErrorDelMockSeTraduce(t *testing.T) {
	mock := conMock(t)
	mock.ConnsErr = &network.NmcliError{
		Args:   []string{"connection", "show"},
		Stderr: "Error: Not authorized to control networking.",
	}

	listMsg, ok := fetchConnections().(connListMsg)
	if !ok {
		t.Fatal("se esperaba connListMsg")
	}

	m := NewSaved()
	m, _ = m.Update(listMsg)
	if m.state != savedError {
		t.Errorf("estado = %v, se esperaba savedError", m.state)
	}
	if !contiene(m.toast, "Sin permisos") {
		t.Errorf("el error no se tradujo: %q", m.toast)
	}
}

// TestVistaDashboardConMock: el panel refleja el estado inyectado.
func TestVistaDashboardConMock(t *testing.T) {
	mock := conMock(t)
	mock.ActiveState = &network.NetworkState{
		Connectivity:   network.ConnectivityFull,
		ActiveSSID:     "Casa",
		ActiveDevice:   "wlan0",
		ActiveType:     "wifi",
		SignalStrength: 80,
		IPAddress:      "192.168.1.50/24",
	}
	mock.Connectivity = network.ConnectivityFull

	m := NewDashboard()
	msg := fetchState()
	sm, ok := msg.(stateMsg)
	if !ok {
		t.Fatalf("se esperaba stateMsg, se obtuvo %T", msg)
	}
	m, _ = m.Update(sm)

	if m.loading {
		t.Error("debería dejar de cargar")
	}
	v := m.View()
	for _, want := range []string{"Casa", "Conectado"} {
		if !contiene(v, want) {
			t.Errorf("la vista no contiene %q", want)
		}
	}
}

// TestVistaDashboardErrorConMock.
func TestVistaDashboardErrorConMock(t *testing.T) {
	mock := conMock(t)
	mock.StateErr = errors.New("nmcli no responde")

	m := NewDashboard()
	m, _ = m.Update(fetchState())
	if m.err == nil {
		t.Fatal("debería registrar el error")
	}
	if !contiene(m.View(), "nmcli no responde") {
		t.Error("la vista no muestra el error")
	}
}

// TestVistaWifiEscaneoConMock: la tabla se arma desde las redes inyectadas,
// incluidas las que tienen ':' en el SSID.
func TestVistaWifiEscaneoConMock(t *testing.T) {
	mock := conMock(t)
	mock.Networks = []network.WiFiNetwork{
		{SSID: "Casa:5G", BSSID: "AA:BB:CC:DD:EE:FF", Security: "WPA2", Signal: 90, Freq: "5180 MHz"},
		{SSID: "Vecina", Security: "WPA2", Signal: 40, Freq: "2412 MHz"},
	}

	m := NewWifiList()
	msg := scanNetworks()
	sr, ok := msg.(scanResultMsg)
	if !ok {
		t.Fatalf("se esperaba scanResultMsg, se obtuvo %T", msg)
	}
	if sr.err != nil {
		t.Fatalf("error inesperado: %v", sr.err)
	}
	m, _ = m.Update(sr)

	if m.state != wifiIdle {
		t.Errorf("estado = %v, se esperaba wifiIdle", m.state)
	}
	if !contiene(m.View(), "Casa:5G") {
		t.Error("la vista no muestra el SSID con ':'")
	}
}

// TestVistaVPNConMock: la lista de VPN se puebla desde el mock.
func TestVistaVPNConMock(t *testing.T) {
	mock := conMock(t)
	mock.VPNs = []network.VPNConnection{
		{Name: "oficina", Type: "openvpn", Active: true},
	}

	m := NewVPNList()
	m, _ = m.Update(fetchVPNs())
	if got := len(m.vpns); got != 1 {
		t.Fatalf("se cargaron %d VPNs, se esperaba 1", got)
	}
	if got := m.getSelectedName(); got != "oficina" {
		t.Errorf("getSelectedName() = %q, se esperaba oficina", got)
	}
}

// TestVistaHotspotConMock: el panel refleja un hotspot activo inyectado.
func TestVistaHotspotConMock(t *testing.T) {
	mock := conMock(t)
	mock.Hotspot = &network.HotspotConfig{
		Active: true, SSID: "MiAP", Band: "bg", Iface: "wlan0", Clients: 3,
	}

	m := NewHotspot()
	m, _ = m.Update(fetchHotspotStatus())
	if m.status == nil || !m.status.Active {
		t.Fatal("debería quedar activo")
	}
	v := m.View()
	for _, want := range []string{"MiAP", "3"} {
		if !contiene(v, want) {
			t.Errorf("la vista no contiene %q", want)
		}
	}
}

// TestDesconectarUsaElMock: la acción 'x' del dashboard llega al cliente.
func TestDesconectarUsaElMock(t *testing.T) {
	mock := conMock(t)
	mock.ActiveState = &network.NetworkState{
		ActiveDevice: "wlan0", ActiveType: "wifi", Connectivity: network.ConnectivityFull,
	}

	m := NewDashboard()
	m.loading = false
	m.state = mock.ActiveState

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if cmd == nil {
		t.Fatal("'x' no disparó comando")
	}
	// Ejecutar el batch para que efectivamente llame al cliente.
	if batch, ok := cmd().(tea.BatchMsg); ok {
		for _, c := range batch {
			if c != nil {
				c()
			}
		}
	}

	if got := mock.UltimaLlamada(); got != "disconnect wlan0" {
		t.Errorf("última llamada = %q, se esperaba 'disconnect wlan0'", got)
	}
}

// TestRadioUsaElMock: el toggle de radio Wi-Fi llega al hook inyectado.
func TestRadioUsaElMock(t *testing.T) {
	estado := true
	var pedidos []bool
	restaurar := network.SetRadioHooks(
		func() bool { return estado },
		func(enable bool) error { pedidos = append(pedidos, enable); estado = enable; return nil },
		func() string { return "wlan0" },
	)
	t.Cleanup(restaurar)

	// La radio debe consultarse a través del hook.
	if !network.IsWiFiEnabled() {
		t.Error("IsWiFiEnabled debería reflejar el hook")
	}
	if err := network.RadioToggleWiFi(false); err != nil {
		t.Fatalf("toggle falló: %v", err)
	}
	if len(pedidos) != 1 || pedidos[0] != false {
		t.Errorf("pedidos = %v, se esperaba [false]", pedidos)
	}
	if network.IsWiFiEnabled() {
		t.Error("el estado debería haber cambiado a apagado")
	}
}

// TestSetClientRestaura: el cliente original vuelve tras el test, para no
// contaminar a los demás.
func TestSetClientRestaura(t *testing.T) {
	original := network.Cliente()

	mock := &network.MockClient{}
	restaurar := network.SetClient(mock)
	if network.Cliente() != network.Client(mock) {
		t.Error("SetClient no instaló el mock")
	}
	restaurar()

	if network.Cliente() != original {
		t.Error("no se restauró el cliente original")
	}
}
