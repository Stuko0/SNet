package network

import "fmt"

// Client defines the interface for NetworkManager operations.
// Implementations can wrap nmcli (production) or be mocked (tests).
type Client interface {
	GetConnectivity() ConnectivityStatus
	GetActiveConnection() (*NetworkState, error)
	ScanWiFi(rescan bool) ([]WiFiNetwork, error)
	GetConnections() ([]Connection, error)
	GetVPNs() ([]VPNConnection, error)
	ConnectToWiFi(ssid, password string) error
	Disconnect(device string) error
	DeleteConnection(name string) error
	GetConnectionPassword(name string) (string, error)
	ModifyConnection(name, setting, value string) error
	ConnectionUp(name string) error
	ConnectionDown(name string) error
	AddWiFiConnection(ssid, password string) error
	HotspotStart(cfg HotspotConfig) error
	HotspotStop() error
	HotspotStatus() (*HotspotConfig, error)
	GetHotspotIface() string
}

// Ensure nmcliClient implements Client.
var _ Client = (*NmcliClient)(nil)

// NmcliClient is the production implementation backed by nmcli.
type NmcliClient struct{}

func NewClient() *NmcliClient { return &NmcliClient{} }

// NetworkState representa el estado actual del sistema de red
type NetworkState struct {
	Connectivity  ConnectivityStatus
	ActiveSSID    string
	ActiveDevice  string
	ActiveType    string // wifi, ethernet, tun, bridge
	SignalStrength int    // 0-100
	IPAddress     string
	Gateway       string
	DNSServers    []string
	Speed         string // ej: "866.7 MBit/s"
	IsVPNActive   bool
	VPNName       string
}

type ConnectivityStatus int

const (
	ConnectivityUnknown ConnectivityStatus = iota
	ConnectivityNone
	ConnectivityPortal
	ConnectivityLimited
	ConnectivityFull
)

func (c ConnectivityStatus) String() string {
	switch c {
	case ConnectivityNone:
		return "none"
	case ConnectivityPortal:
		return "portal"
	case ConnectivityLimited:
		return "limited"
	case ConnectivityFull:
		return "full"
	default:
		return "unknown"
	}
}

// WiFiNetwork representa una red Wi-Fi visible en el escaneo
type WiFiNetwork struct {
	SSID     string
	BSSID    string
	Security string // WPA2, WPA3, WEP, Open, etc
	Signal   int    // 0-100
	Freq     string // 2.4GHz, 5GHz, 6GHz
	Channel  int
	Known    bool // si ya existe una conexión guardada
	Active   bool // si es la red actualmente conectada
}

// Connection representa una conexión guardada en NetworkManager
type Connection struct {
	Name       string
	UUID       string
	Type       string // wifi, ethernet, vpn, bridge, tun
	Device     string
	Autoconnect bool
	Active     bool
}

// VPNConnection representa una conexión VPN
type VPNConnection struct {
	Name       string
	UUID       string
	Type       string // openvpn, wireguard, l2tp, sstp, etc
	Active     bool
	Autoconnect bool
}

// HotspotConfig representa la configuración de un hotspot
type HotspotConfig struct {
	SSID     string
	Password string
	Band     string // "a" (5GHz) o "bg" (2.4GHz)
	Iface    string
	Active   bool
	Clients  int
}

func (n WiFiNetwork) SignalBars() string {
	switch {
	case n.Signal >= 80:
		return "████████"
	case n.Signal >= 60:
		return "██████░░"
	case n.Signal >= 40:
		return "████░░░░"
	case n.Signal >= 20:
		return "██░░░░░░"
	default:
		return "█░░░░░░░"
	}
}

func (n WiFiNetwork) String() string {
	return fmt.Sprintf("%-25s %-10s %s %s", n.SSID, n.Security, n.SignalBars(), n.Freq)
}
