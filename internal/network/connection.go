package network

import (
	"fmt"
	"strings"
)

// GetConnectionPassword obtiene la contraseña de una conexión WiFi guardada.
func (c *NmcliClient) GetConnectionPassword(name string) (string, error) {
	out, err := runCmd("-s", "-t", "-f", "802-11-wireless-security.psk", "connection", "show", name)
	if err != nil {
		return "", err
	}

	parts := strings.SplitN(out, ":", 2)
	if len(parts) == 2 {
		return parts[1], nil
	}
	return "", fmt.Errorf("no se encontró contraseña para %s", name)
}

// GetConnectionSettings obtiene una o más propiedades de una conexión guardada.
// Usa -s para incluir secretos (contraseñas).
func (c *NmcliClient) GetConnectionSettings(name string, settings ...string) (map[string]string, error) {
	args := []string{"-s", "-t", "-f", strings.Join(settings, ","), "connection", "show", name}
	out, err := runCmd(args...)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string)
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result, nil
}

// ModifyConnection cambia una propiedad de una conexión.
func (c *NmcliClient) ModifyConnection(name, setting, value string) error {
	_, err := runCmd("connection", "modify", name, setting, value)
	return err
}

// ConnectionUp activa una conexión.
func (c *NmcliClient) ConnectionUp(name string) error {
	_, err := runCmd("connection", "up", name)
	return err
}

// ConnectionDown desactiva una conexión.
func (c *NmcliClient) ConnectionDown(name string) error {
	_, err := runCmd("connection", "down", name)
	return err
}

// AddWiFiConnection crea una nueva conexión WiFi guardada.
func (c *NmcliClient) AddWiFiConnection(ssid, password string) error {
	_, err := runCmd("connection", "add",
		"type", "wifi",
		"con-name", ssid,
		"ssid", ssid,
		"802-11-wireless-security.key-mgmt", "wpa-psk",
		"802-11-wireless-security.psk", password,
	)
	return err
}
