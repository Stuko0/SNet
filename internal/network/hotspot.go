package network

import (
	"fmt"
	"os/exec"
	"strings"
)

const hotspotConnPrefix = "SNet-Hotspot"

// HotspotStart inicia un hotspot Wi-Fi con la configuración dada.
// Crea (o reemplaza) una conexión hotspot y la activa.
func (c *NmcliClient) HotspotStart(cfg HotspotConfig) error {
	conName := hotspotConnPrefix

	// Si ya existe una conexión hotspot, eliminarla primero
	existing := c.getHotspotConnection()
	if existing != "" {
		_ = c.ConnectionDown(existing)
		_ = c.DeleteConnection(existing)
	}

	band := cfg.Band
	if band == "" {
		band = "bg" // 2.4GHz por defecto
	}

	// Crear la conexión hotspot
	_, err := runCmd("connection", "add",
		"type", "wifi",
		"con-name", conName,
		"ifname", cfg.Iface,
		"autoconnect", "no",
		"802-11-wireless.mode", "ap",
		"802-11-wireless.ssid", cfg.SSID,
		"802-11-wireless.band", band,
		"802-11-wireless-security.key-mgmt", "wpa-psk",
		"802-11-wireless-security.psk", cfg.Password,
		"ipv4.method", "shared",
	)
	if err != nil {
		return fmt.Errorf("crear hotspot: %w", err)
	}

	// Activar el hotspot
	_, err = runCmd("connection", "up", conName)
	if err != nil {
		return fmt.Errorf("activar hotspot: %w", err)
	}

	return nil
}

// HotspotStop detiene el hotspot si está activo.
func (c *NmcliClient) HotspotStop() error {
	conName := c.getActiveHotspotName()
	if conName == "" {
		return nil // no hay hotspot activo
	}
	err := c.ConnectionDown(conName)
	if err != nil {
		return fmt.Errorf("detener hotspot: %w", err)
	}
	// Limpiar la conexión hotspot
	_ = c.DeleteConnection(hotspotConnPrefix)
	return nil
}

// HotspotStatus retorna el estado actual del hotspot.
func (c *NmcliClient) HotspotStatus() (*HotspotConfig, error) {
	cfg := &HotspotConfig{Active: false}

	// Buscar si hay un hotspot activo por nuestro prefijo
	activeName := c.getActiveHotspotName()
	if activeName == "" {
		return cfg, nil
	}

	cfg.Active = true

	// Obtener detalles de la conexión hotspot activa
	out, err := runCmd("-t", "-f",
		"802-11-wireless.ssid,802-11-wireless.band,802-11-wireless-security.psk,connection.interface-name",
		"connection", "show", activeName)
	if err != nil {
		return cfg, nil
	}

	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		switch parts[0] {
		case "802-11-wireless.ssid":
			cfg.SSID = parts[1]
		case "802-11-wireless.band":
			cfg.Band = parts[1]
		case "802-11-wireless-security.psk":
			cfg.Password = parts[1]
		case "connection.interface-name":
			cfg.Iface = parts[1]
		}
	}

	// Intentar contar clientes conectados (aproximado)
	cfg.Clients = c.countHotspotClients(cfg.Iface)

	return cfg, nil
}

// GetHotspotIface retorna la interfaz Wi-Fi disponible para hotspot.
func (c *NmcliClient) GetHotspotIface() string {
	out, err := runCmd("-t", "-f", "DEVICE,TYPE", "device", "status")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(out, "\n") {
		parts := strings.Split(line, ":")
		if len(parts) >= 2 && parts[1] == "wifi" {
			return parts[0]
		}
	}
	return ""
}

// getHotspotConnection busca si existe una conexión hotspot guardada.
func (c *NmcliClient) getHotspotConnection() string {
	out, err := runCmd("-t", "-f", "NAME", "connection", "show")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(out, "\n") {
		if line == hotspotConnPrefix {
			return line
		}
	}
	return ""
}

// getActiveHotspotName busca una conexión hotspot activa.
func (c *NmcliClient) getActiveHotspotName() string {
	out, err := runCmd("-t", "-f", "NAME,TYPE,MODE", "connection", "show", "--active")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(out, "\n") {
		parts := strings.Split(line, ":")
		if len(parts) >= 3 && parts[2] == "ap" {
			return parts[0]
		}
	}
	// Fallback: buscar por prefijo
	for _, line := range strings.Split(out, "\n") {
		parts := strings.Split(line, ":")
		if len(parts) >= 1 && strings.HasPrefix(parts[0], hotspotConnPrefix) {
			return parts[0]
		}
	}
	return ""
}

// countHotspotClients cuenta clientes conectados al hotspot (vía iw).
func (c *NmcliClient) countHotspotClients(iface string) int {
	if iface == "" {
		return 0
	}
	// Usar iw para contar estaciones conectadas
	cmd := exec.Command("iw", "dev", iface, "station", "dump")
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	count := 0
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "Station ") {
			count++
		}
	}
	return count
}
