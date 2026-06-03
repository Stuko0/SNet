package network

import (
	"fmt"
	"os/exec"
	"strings"
)

// runCmd ejecuta nmcli y devuelve stdout
func runCmd(args ...string) (string, error) {
	// Forzamos LC_ALL=C para tener output en inglés siempre
	cmd := exec.Command("nmcli", args...)
	cmd.Env = append(cmd.Environ(), "LC_ALL=C")
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", &NmcliError{Args: args, ExitCode: exitErr.ExitCode(), Stderr: string(exitErr.Stderr)}
		}
		return "", &NmcliError{Args: args, Stderr: err.Error()}
	}
	return strings.TrimRight(string(out), "\n"), nil
}

// NmcliError representa un error de nmcli
type NmcliError struct {
	Args     []string
	ExitCode int
	Stderr   string
}

func (e *NmcliError) Error() string {
	return "nmcli " + strings.Join(e.Args, " ") + ": " + e.Stderr
}

// GetGeneralStatus retorna el estado general de NetworkManager
func (c *NmcliClient) GetGeneralStatus() string {
	out, err := runCmd("general", "status")
	if err != nil {
		return ""
	}
	return out
}

// GetConnectivity retorna el estado de conectividad
func (c *NmcliClient) GetConnectivity() ConnectivityStatus {
	out, err := runCmd("general", "connectivity")
	if err != nil {
		return ConnectivityUnknown
	}
	switch strings.TrimSpace(out) {
	case "full":
		return ConnectivityFull
	case "limited":
		return ConnectivityLimited
	case "portal":
		return ConnectivityPortal
	case "none":
		return ConnectivityNone
	default:
		return ConnectivityUnknown
	}
}

// GetActiveConnection obtiene detalles de la conexión activa
func (c *NmcliClient) GetActiveConnection() (*NetworkState, error) {
	state := &NetworkState{
		Connectivity: c.GetConnectivity(),
	}

	// nmcli -t -f DEVICE,TYPE,CONNECTION device status
	out, err := runCmd("-t", "-f", "DEVICE,TYPE,CONNECTION", "device", "status")
	if err != nil {
		return state, err
	}

	for _, line := range strings.Split(out, "\n") {
		parts := strings.Split(line, ":")
		if len(parts) < 3 {
			continue
		}
		device, iftype, conn := parts[0], parts[1], parts[2]
		if conn == "" || conn == "--" {
			continue
		}
		if iftype == "wifi" || iftype == "ethernet" {
			state.ActiveDevice = device
			state.ActiveType = iftype
			state.ActiveSSID = conn
		}
		if iftype == "tun" {
			state.IsVPNActive = true
			state.VPNName = conn
		}
	}

	// IP, gateway, DNS
	if state.ActiveDevice != "" {
		// nmcli -t -f IP4.ADDRESS device show <dev>
		devInfo, _ := runCmd("-t", "-f", "IP4.ADDRESS,IP4.GATEWAY,IP4.DNS", "device", "show", state.ActiveDevice)
		for _, line := range strings.Split(devInfo, "\n") {
			if strings.HasPrefix(line, "IP4.ADDRESS:") {
				state.IPAddress = strings.TrimPrefix(line, "IP4.ADDRESS:")
			}
			if strings.HasPrefix(line, "IP4.GATEWAY:") {
				state.Gateway = strings.TrimPrefix(line, "IP4.GATEWAY:")
			}
			if strings.HasPrefix(line, "IP4.DNS:") {
				dns := strings.TrimPrefix(line, "IP4.DNS:")
				if dns != "" {
					state.DNSServers = append(state.DNSServers, dns)
				}
			}
		}

		// Velocidad Wi-Fi
		if state.ActiveType == "wifi" {
			// nmcli -t -f GENERAL.SPEED device show <dev>
			speedOut, _ := runCmd("-t", "-f", "GENERAL.SPEED", "device", "show", state.ActiveDevice)
			if strings.HasPrefix(speedOut, "GENERAL.SPEED:") {
				state.Speed = strings.TrimPrefix(speedOut, "GENERAL.SPEED:")
			}

			// Señal
			signalOut, _ := runCmd("-t", "-f", "SIGNAL", "device", "wifi", "list", "--rescan", "no")
			for _, line := range strings.Split(signalOut, "\n") {
				// Formato: SSID:SEC:SIGNAL:FREQ:BSSID:MODE:CHAN
				parts := strings.Split(line, ":")
				if len(parts) >= 3 && parts[0] == state.ActiveSSID {
					fmt.Sscanf(parts[2], "%d", &state.SignalStrength)
					break
				}
			}
		}
	}

	return state, nil
}

// ScanWiFi escanea redes Wi-Fi y devuelve la lista
func (c *NmcliClient) ScanWiFi(rescan bool) ([]WiFiNetwork, error) {
	args := []string{"-t", "-f", "SSID,BSSID,SECURITY,SIGNAL,FREQ,CHAN,MODE", "device", "wifi", "list"}
	if rescan {
		args = append(args, "--rescan", "yes")
	} else {
		args = append(args, "--rescan", "no")
	}

	out, err := runCmd(args...)
	if err != nil {
		return nil, err
	}

	// Obtener conexiones conocidas para marcar las conocidas
	known := getKnownSSIDs()

	lines := strings.Split(out, "\n")
	seen := make(map[string]bool)
	var networks []WiFiNetwork

	for _, line := range lines {
		parts := strings.Split(line, ":")
		if len(parts) < 7 {
			continue
		}
		ssid := parts[0]
		if ssid == "" || seen[ssid] {
			continue
		}
		signal := 0
		fmt.Sscanf(parts[3], "%d", &signal)

		networks = append(networks, WiFiNetwork{
			SSID:     ssid,
			BSSID:    parts[1],
			Security: parts[2],
			Signal:   signal,
			Freq:     parts[4],
			Channel:  0,
			Known:    known[ssid],
		})
		seen[ssid] = true
	}

	return networks, nil
}

// getKnownSSIDs devuelve un set de SSIDs que ya tienen conexión guardada
func getKnownSSIDs() map[string]bool {
	out, err := runCmd("-t", "-f", "NAME,TYPE", "connection", "show")
	if err != nil {
		return nil
	}
	result := make(map[string]bool)
	for _, line := range strings.Split(out, "\n") {
		parts := strings.Split(line, ":")
		if len(parts) >= 2 && parts[1] == "wifi" {
			result[parts[0]] = true
		}
	}
	return result
}

// GetConnections lista todas las conexiones guardadas
func (c *NmcliClient) GetConnections() ([]Connection, error) {
	out, err := runCmd("-t", "-f", "NAME,UUID,TYPE,DEVICE,AUTOCONNECT", "connection", "show")
	if err != nil {
		return nil, err
	}

	// Obtener conexión activa para marcar
	activeConns := getActiveConnNames()

	var conns []Connection
	for _, line := range strings.Split(out, "\n") {
		parts := strings.Split(line, ":")
		if len(parts) < 5 {
			continue
		}
		conns = append(conns, Connection{
			Name:        parts[0],
			UUID:        parts[1],
			Type:        parts[2],
			Device:      parts[3],
			Autoconnect: parts[4] == "yes",
			Active:      activeConns[parts[0]],
		})
	}
	return conns, nil
}

func getActiveConnNames() map[string]bool {
	out, err := runCmd("-t", "-f", "NAME", "connection", "show", "--active")
	if err != nil {
		return nil
	}
	result := make(map[string]bool)
	for _, name := range strings.Split(out, "\n") {
		if name != "" {
			result[name] = true
		}
	}
	return result
}

// GetVPNs lista las conexiones VPN
func (c *NmcliClient) GetVPNs() ([]VPNConnection, error) {
	out, err := runCmd("-t", "-f", "NAME,UUID,TYPE,DEVICE,AUTOCONNECT", "connection", "show")
	if err != nil {
		return nil, err
	}

	activeConns := getActiveConnNames()

	var vpns []VPNConnection
	for _, line := range strings.Split(out, "\n") {
		parts := strings.Split(line, ":")
		if len(parts) < 5 {
			continue
		}
		if isVPNSection(parts[2]) {
			vpns = append(vpns, VPNConnection{
				Name:        parts[0],
				UUID:        parts[1],
				Type:        parts[2],
				Active:      activeConns[parts[0]],
				Autoconnect: parts[4] == "yes",
			})
		}
	}
	return vpns, nil
}

func isVPNSection(t string) bool {
	vpnTypes := []string{"vpn", "openvpn", "wireguard", "l2tp", "sstp", "pptp"}
	for _, vt := range vpnTypes {
		if t == vt {
			return true
		}
	}
	return false
}

// ConnectToWiFi se conecta a una red Wi-Fi
func (c *NmcliClient) ConnectToWiFi(ssid, password string) error {
	if password == "" {
		_, err := runCmd("device", "wifi", "connect", ssid)
		return err
	}
	_, err := runCmd("device", "wifi", "connect", ssid, "password", password)
	return err
}

// Disconnect desconecta un dispositivo
func (c *NmcliClient) Disconnect(device string) error {
	_, err := runCmd("device", "disconnect", device)
	return err
}

// DeleteConnection elimina una conexión guardada
func (c *NmcliClient) DeleteConnection(name string) error {
	_, err := runCmd("connection", "delete", name)
	return err
}

// ============================================================
// Standalone convenience functions (delegate to DefaultClient)
// ============================================================

// DefaultClient is the package-level client used by convenience functions.
var DefaultClient = &NmcliClient{}

func GetGeneralStatus() string                      { return DefaultClient.GetGeneralStatus() }
func GetConnectivity() ConnectivityStatus            { return DefaultClient.GetConnectivity() }
func GetActiveConnection() (*NetworkState, error)    { return DefaultClient.GetActiveConnection() }
func ScanWiFi(rescan bool) ([]WiFiNetwork, error)    { return DefaultClient.ScanWiFi(rescan) }
func GetConnections() ([]Connection, error)           { return DefaultClient.GetConnections() }
func GetVPNs() ([]VPNConnection, error)               { return DefaultClient.GetVPNs() }
func ConnectToWiFi(ssid, password string) error       { return DefaultClient.ConnectToWiFi(ssid, password) }
func Disconnect(device string) error                  { return DefaultClient.Disconnect(device) }
func DeleteConnection(name string) error              { return DefaultClient.DeleteConnection(name) }
func GetConnectionPassword(name string) (string, error) { return DefaultClient.GetConnectionPassword(name) }
func ModifyConnection(name, setting, value string) error { return DefaultClient.ModifyConnection(name, setting, value) }
func ConnectionUp(name string) error                     { return DefaultClient.ConnectionUp(name) }
func ConnectionDown(name string) error                   { return DefaultClient.ConnectionDown(name) }
func AddWiFiConnection(ssid, password string) error      { return DefaultClient.AddWiFiConnection(ssid, password) }
func HotspotStart(cfg HotspotConfig) error                  { return DefaultClient.HotspotStart(cfg) }
func HotspotStop() error                                     { return DefaultClient.HotspotStop() }
func HotspotStatus() (*HotspotConfig, error)                 { return DefaultClient.HotspotStatus() }
func GetHotspotIface() string                                { return DefaultClient.GetHotspotIface() }
