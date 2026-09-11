package network

import (
	"strings"
	"sync"
)

// Las operaciones de radio Wi-Fi no forman parte de la interfaz Client (no son
// de una conexión concreta, sino del adaptador), así que se exponen vía hooks
// sustituibles para poder testear la UI sin tocar la radio real.
var (
	radioMu       sync.RWMutex
	wifiEnabledFn = func() bool { return isWiFiEnabledReal() }
	radioToggleFn = func(enable bool) error { return radioToggleWiFiReal(enable) }
	wifiDeviceFn  = func() string { return getWiFiDeviceReal() }
)

// SetRadioHooks sustituye las operaciones de radio Wi-Fi. Los argumentos nil se
// dejan sin cambiar. Devuelve una función para restaurar las originales:
//
//	defer network.SetRadioHooks(mockEnabled, mockToggle, nil)()
func SetRadioHooks(
	enabled func() bool,
	toggle func(enable bool) error,
	device func() string,
) func() {
	radioMu.Lock()
	prevEnabled, prevToggle, prevDevice := wifiEnabledFn, radioToggleFn, wifiDeviceFn
	if enabled != nil {
		wifiEnabledFn = enabled
	}
	if toggle != nil {
		radioToggleFn = toggle
	}
	if device != nil {
		wifiDeviceFn = device
	}
	radioMu.Unlock()

	return func() {
		radioMu.Lock()
		wifiEnabledFn, radioToggleFn, wifiDeviceFn = prevEnabled, prevToggle, prevDevice
		radioMu.Unlock()
	}
}

// GetWiFiDevice retorna el nombre del dispositivo Wi-Fi activo, si existe.
func GetWiFiDevice() string {
	radioMu.RLock()
	fn := wifiDeviceFn
	radioMu.RUnlock()
	return fn()
}

// IsWiFiEnabled verifica si el Wi-Fi está habilitado.
func IsWiFiEnabled() bool {
	radioMu.RLock()
	fn := wifiEnabledFn
	radioMu.RUnlock()
	return fn()
}

// RadioToggleWiFi activa o desactiva el Wi-Fi.
func RadioToggleWiFi(enable bool) error {
	radioMu.RLock()
	fn := radioToggleFn
	radioMu.RUnlock()
	return fn(enable)
}

// getWiFiDeviceReal retorna el nombre del dispositivo Wi-Fi activo, si existe.
func getWiFiDeviceReal() string {
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
func isWiFiEnabledReal() bool {
	out, err := runCmd("-t", "-f", "WIFI", "general", "status")
	if err != nil {
		return false
	}
	return out == "enabled"
}

// RadioToggleWiFi activa o desactiva el Wi-Fi.
func radioToggleWiFiReal(enable bool) error {
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
