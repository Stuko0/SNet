package network

import (
	"errors"
	"sync"
)

// MockClient es un Client en memoria para tests.
//
// Permite testear las vistas sin tocar nmcli ni la red real, que es lo que
// hacía que los tests de UI dependieran de la máquina donde corrían.
//
// Todos los campos de resultado y los errores son configurables; las llamadas
// se registran para poder verificar qué pidió la UI.
type MockClient struct {
	mu sync.Mutex

	// Resultados a devolver.
	Connectivity  ConnectivityStatus
	ActiveState   *NetworkState
	StateErr      error
	Networks      []WiFiNetwork
	ScanErr       error
	Connections   []Connection
	ConnsErr      error
	VPNs          []VPNConnection
	VPNsErr       error
	Password      string
	PasswordErr   error
	Settings      map[string]string
	SettingsErr   error
	Hotspot       *HotspotConfig
	HotspotErr    error
	HotspotIface  string
	GeneralStatus string

	// Errores de las operaciones de escritura.
	ConnectErr error
	DeleteErr  error
	ModifyErr  error
	UpErr      error
	DownErr    error
	AddWiFiErr error
	AddVPNErr  error

	// Llamadas registradas, en orden.
	Llamadas []string
}

// nuevaLlamada registra una llamada (protegida por mutex).
func (m *MockClient) anotar(s string) {
	m.mu.Lock()
	m.Llamadas = append(m.Llamadas, s)
	m.mu.Unlock()
}

// LlamadasCon devuelve las llamadas que contienen el texto dado.
func (m *MockClient) LlamadasCon(sub string) []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []string
	for _, l := range m.Llamadas {
		if containsStr(l, sub) {
			out = append(out, l)
		}
	}
	return out
}

// UltimaLlamada devuelve la última llamada registrada, o "".
func (m *MockClient) UltimaLlamada() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.Llamadas) == 0 {
		return ""
	}
	return m.Llamadas[len(m.Llamadas)-1]
}

// Reset limpia las llamadas registradas (no los resultados configurados).
func (m *MockClient) Reset() {
	m.mu.Lock()
	m.Llamadas = nil
	m.mu.Unlock()
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func (m *MockClient) GetGeneralStatus() string {
	m.anotar("general status")
	return m.GeneralStatus
}

func (m *MockClient) GetConnectivity() ConnectivityStatus {
	m.anotar("connectivity")
	return m.Connectivity
}

func (m *MockClient) GetActiveConnection() (*NetworkState, error) {
	m.anotar("active connection")
	if m.StateErr != nil {
		return nil, m.StateErr
	}
	if m.ActiveState == nil {
		return &NetworkState{}, nil
	}
	return m.ActiveState, nil
}

func (m *MockClient) ScanWiFi(rescan bool) ([]WiFiNetwork, error) {
	m.anotar("scan wifi")
	if m.ScanErr != nil {
		return nil, m.ScanErr
	}
	return m.Networks, nil
}

func (m *MockClient) GetConnections() ([]Connection, error) {
	m.anotar("get connections")
	if m.ConnsErr != nil {
		return nil, m.ConnsErr
	}
	return m.Connections, nil
}

func (m *MockClient) GetVPNs() ([]VPNConnection, error) {
	m.anotar("get vpns")
	if m.VPNsErr != nil {
		return nil, m.VPNsErr
	}
	return m.VPNs, nil
}

func (m *MockClient) ConnectToWiFi(ssid, password string) error {
	// La contraseña NO se registra: los tests verifican que no se filtre.
	m.anotar("connect wifi " + ssid)
	return m.ConnectErr
}

func (m *MockClient) Disconnect(device string) error {
	m.anotar("disconnect " + device)
	return m.DownErr
}

func (m *MockClient) DeleteConnection(name string) error {
	m.anotar("delete " + name)
	return m.DeleteErr
}

func (m *MockClient) GetConnectionPassword(name string) (string, error) {
	m.anotar("password " + name)
	if m.PasswordErr != nil {
		return "", m.PasswordErr
	}
	if m.Password == "" {
		return "", errors.New("sin contraseña")
	}
	return m.Password, nil
}

func (m *MockClient) GetConnectionSettings(name string, settings ...string) (map[string]string, error) {
	m.anotar("settings " + name)
	if m.SettingsErr != nil {
		return nil, m.SettingsErr
	}
	return m.Settings, nil
}

func (m *MockClient) ModifyConnection(name, setting, value string) error {
	m.anotar("modify " + name + " " + setting)
	return m.ModifyErr
}

func (m *MockClient) ConnectionUp(name string) error {
	m.anotar("up " + name)
	return m.UpErr
}

func (m *MockClient) ConnectionDown(name string) error {
	m.anotar("down " + name)
	return m.DownErr
}

func (m *MockClient) AddWiFiConnection(ssid, password string) error {
	m.anotar("add wifi " + ssid)
	return m.AddWiFiErr
}

func (m *MockClient) HotspotStart(cfg HotspotConfig) error {
	m.anotar("hotspot start " + cfg.SSID)
	return m.HotspotErr
}

func (m *MockClient) HotspotStop() error {
	m.anotar("hotspot stop")
	return m.HotspotErr
}

func (m *MockClient) HotspotStatus() (*HotspotConfig, error) {
	m.anotar("hotspot status")
	if m.HotspotErr != nil {
		return nil, m.HotspotErr
	}
	if m.Hotspot == nil {
		return &HotspotConfig{Active: false}, nil
	}
	return m.Hotspot, nil
}

func (m *MockClient) GetHotspotIface() string {
	m.anotar("hotspot iface")
	return m.HotspotIface
}

func (m *MockClient) AddOpenVPNConnection(name, remote, port, username, password string) error {
	m.anotar("add openvpn " + name)
	return m.AddVPNErr
}

func (m *MockClient) AddWireGuardConnection(name, iface, configFile string) error {
	m.anotar("add wireguard " + name)
	return m.AddVPNErr
}

func (m *MockClient) AddSSTPConnection(name, server, username, password string) error {
	m.anotar("add sstp " + name)
	return m.AddVPNErr
}

// Compilar falla si MockClient deja de cumplir la interfaz.
var _ Client = (*MockClient)(nil)
