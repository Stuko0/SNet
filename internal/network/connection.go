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
	// Formato: 802-11-wireless-security.psk:password
	parts := strings.SplitN(out, ":", 2)
	if len(parts) == 2 {
		return parts[1], nil
	}
	return "", fmt.Errorf("no se encontró contraseña para %s", name)
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
