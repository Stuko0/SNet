package network

import "strings"

// GetWiFiDevice retorna el nombre del dispositivo Wi-Fi activo, si existe.
func GetWiFiDevice() string {
	state, err := GetActiveConnection()
	if err != nil {
		return ""
	}
	if state.ActiveType == "wifi" {
		return state.ActiveDevice
	}

	out, err := runCmd("-t", "-f", "DEVICE,TYPE", "device", "status")
	if err != nil {
		return ""
	}
	for _, line := range parseLines(out) {
		parts := splitLine(line)
		if len(parts) >= 2 && parts[1] == "wifi" {
			return parts[0]
		}
	}
	return ""
}

// IsWiFiEnabled verifica si el Wi-Fi está habilitado.
func IsWiFiEnabled() bool {
	out, err := runCmd("-t", "-f", "WIFI", "general", "status")
	if err != nil {
		return false
	}
	return out == "enabled"
}

// RadioToggleWiFi activa o desactiva el Wi-Fi.
func RadioToggleWiFi(enable bool) error {
	action := "off"
	if enable {
		action = "on"
	}
	_, err := runCmd("radio", "wifi", action)
	return err
}

// parseLines divide un string en líneas ignorando vacías.
func parseLines(s string) []string {
	return strings.Split(strings.TrimSpace(s), "\n")
}

// splitLine divide una línea de nmcli en campos.
func splitLine(s string) []string {
	return strings.Split(s, ":")
}
