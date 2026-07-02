package network

import (
	"fmt"
)

// VPNAuthType representa el tipo de autenticación VPN
type VPNAuthType string

const (
	VPNAuthPassword  VPNAuthType = "password"
	VPNAuthCert      VPNAuthType = "cert"
	VPNAuthStaticKey VPNAuthType = "static-key"
)

// AddOpenVPNConnection crea una nueva conexión OpenVPN.
func (c *NmcliClient) AddOpenVPNConnection(name, remote, port, username, password string) error {
	if remote == "" {
		return fmt.Errorf("el servidor remoto es obligatorio")
	}
	if port == "" {
		port = "1194"
	}

	args := []string{
		"connection", "add",
		"type", "vpn",
		"vpn-type", "openvpn",
		"con-name", name,
		"ifname", "--",
		"vpn.data", fmt.Sprintf("remote=%s", remote),
		"vpn.data", fmt.Sprintf("port=%s", port),
		"vpn.data", "connection-type=password",
	}

	_, err := runCmd(args...)
	if err != nil {
		return fmt.Errorf("crear conexión OpenVPN: %w", err)
	}

	if username != "" && password != "" {
		_, err = runCmd("connection", "modify", name,
			"vpn.user-name", username,
			"vpn.secrets", fmt.Sprintf("password=%s", password),
		)
		if err != nil {
			return fmt.Errorf("configurar credenciales OpenVPN: %w", err)
		}
	}

	return nil
}

// AddWireGuardConnection crea una nueva conexión WireGuard.
func (c *NmcliClient) AddWireGuardConnection(name, iface, configFile string) error {
	args := []string{
		"connection", "add",
		"type", "wireguard",
		"con-name", name,
		"ifname", iface,
		"autoconnect", "no",
	}

	if configFile != "" {
		args = append(args, "wgcfg.filename", configFile)
	}

	_, err := runCmd(args...)
	if err != nil {
		return fmt.Errorf("crear conexión WireGuard: %w", err)
	}
	return nil
}

// AddSSTPConnection crea una nueva conexión SSTP (usada comúnmente en Fedora).
func (c *NmcliClient) AddSSTPConnection(name, server, username, password string) error {
	args := []string{
		"connection", "add",
		"type", "vpn",
		"vpn-type", "sstp",
		"con-name", name,
		"ifname", "--",
		"vpn.data", fmt.Sprintf("gateway=%s", server),
	}

	_, err := runCmd(args...)
	if err != nil {
		return fmt.Errorf("crear conexión SSTP: %w", err)
	}

	if username != "" && password != "" {
		_, err = runCmd("connection", "modify", name,
			"vpn.user-name", username,
			"vpn.secrets", fmt.Sprintf("password=%s", password),
		)
		if err != nil {
			return fmt.Errorf("configurar credenciales SSTP: %w", err)
		}
	}

	return nil
}

// GetVPNStatus retorna información detallada de una VPN activa.
func (c *NmcliClient) GetVPNStatus(name string) (string, error) {
	out, err := runCmd("-t", "-f", "GENERAL.STATE,IP4.ADDRESS", "connection", "show", name)
	if err != nil {
		return "", err
	}
	return out, nil
}

// ListVPNTypes retorna los tipos de VPN soportados.
func ListVPNTypes() []string {
	types := []string{"openvpn", "wireguard", "sstp", "l2tp", "pptp"}
	return types
}
