package network

import (
	"fmt"
	"strings"
)

// Mensajes de error orientados al usuario.
//
// Se centralizan acá por dos razones:
//
//  1. nmcli devuelve texto en inglés y con jerga ("Secrets were required, but
//     not provided", "Connection activation failed"). Mostrarlo crudo deja al
//     usuario sin saber qué hacer.
//  2. Los argumentos pueden contener una contraseña (`device wifi connect X
//     password Y`). El error viejo concatenaba TODOS los argumentos, así que el
//     secreto terminaba impreso en pantalla. Acá nunca se exponen.
const (
	msgSinPermisos   = "Sin permisos para modificar la red. Agregá tu usuario al grupo 'network' o autenticate con polkit."
	msgPwdIncorrecta = "Contraseña incorrecta o el método de seguridad de la red no coincide."
	msgNoEncontrada  = "No se encontró la red o la conexión. Puede haberse alejado del alcance o haber sido eliminada."
	msgTimeout       = "NetworkManager no respondió a tiempo. Reintentá o revisá el estado con 'nmcli general status'."
	msgHardwareOcup  = "El adaptador Wi-Fi no puede hacer hotspot mientras está conectado a una red."
	msgYaActiva      = "La conexión ya estaba activa."
	msgSinNM         = "No se puede contactar a NetworkManager. Verificá que el servicio esté corriendo."
	msgGenerico      = "NetworkManager rechazó la operación"
)

// Mensajes exportados para que la capa de UI pueda comparar (p. ej. detectar un
// timeout y ofrecer reintentar) sin duplicar el texto.
const (
	MsgTimeout = msgTimeout
	// MsgSinPermisos permite mostrar una ayuda específica de permisos.
	MsgSinPermisos = msgSinPermisos
)

// userMessage traduce un error de nmcli a un mensaje en español, sin secretos.
//
// Si no reconoce el patrón devuelve el stderr limpio (recortado) en vez del
// comando completo, para no filtrar argumentos sensibles.
func userMessage(err error) string {
	if err == nil {
		return ""
	}

	nerr, ok := err.(*NmcliError)
	if !ok {
		return err.Error()
	}

	if nerr.IsTimeout() {
		return msgTimeout
	}

	// El texto de nmcli llega en inglés, con LC_ALL=C forzado.
	e := strings.ToLower(nerr.Stderr)

	switch {
	// Permisos / polkit. nmcli no dice "permission" siempre: usa varias formas.
	case strings.Contains(e, "not authorized"),
		strings.Contains(e, "insufficient privileges"),
		strings.Contains(e, "permission denied"),
		strings.Contains(e, "could not create polkit"),
		strings.Contains(e, "requires authentication"):
		return msgSinPermisos

	// Contraseña / secretos / credenciales 802.1X.
	case strings.Contains(e, "secrets were required"),
		strings.Contains(e, "no secrets"),
		strings.Contains(e, "password"),
		strings.Contains(e, "invalid password"),
		strings.Contains(e, "802-1x"), // nmcli escribe "802-1x" en los mensajes
		strings.Contains(e, "802.1x"), // y "802.1x" en las propiedades
		strings.Contains(e, "psk"):
		return msgPwdIncorrecta

	// Timeout / no responde.
	case strings.Contains(e, "timeout"),
		strings.Contains(e, "timed out"):
		return msgTimeout

	// No encontrada / fuera de alcance / no existe.
	case strings.Contains(e, "not found"),
		strings.Contains(e, "no network with ssid"),
		strings.Contains(e, "not in range"),
		strings.Contains(e, "unknown connection"),
		strings.Contains(e, "does not exist"):
		return msgNoEncontrada

	// Hotspot sobre adaptador ocupado.
	case strings.Contains(e, "not compatible with"),
		strings.Contains(e, "device is busy"),
		strings.Contains(e, "busy"),
		strings.Contains(e, "ap mode"),
		strings.Contains(e, "not supported"):
		return msgHardwareOcup

	// Ya estaba activa.
	case strings.Contains(e, "already active"),
		strings.Contains(e, "already connected"):
		return msgYaActiva

	// NetworkManager ausente o inaccesible.
	case strings.Contains(e, "could not connect to networkmanager"),
		strings.Contains(e, "networkmanager is not running"),
		strings.Contains(e, "is not available"):
		return msgSinNM
	}

	// Sin match: devolver el stderr recortado, nunca los argumentos (pueden
	// contener la contraseña).
	detalle := sanitizeStderr(nerr.Stderr)
	if detalle == "" {
		return msgGenerico
	}
	return msgGenerico + ": " + detalle
}

// sanitizeStderr limpia el stderr de nmcli para mostrarlo: quita el prefijo
// "Error: ", colapsa espacios y recorta si es muy largo. El recorte es por
// runas (la elipsis `…` ocupa 3 bytes en UTF-8, así que medir en bytes daba un
// resultado más largo de lo pedido).
func sanitizeStderr(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "Error: ")
	s = strings.Join(strings.Fields(s), " ")
	const max = 160
	runas := []rune(s)
	if len(runas) > max {
		return string(runas[:max-1]) + "…"
	}
	return s
}

// UserMessage expone la traducción para la capa de UI.
func UserMessage(err error) string { return userMessage(err) }

// redactArgs devuelve los argumentos con los valores sensibles ocultos. Se usa
// en logs de depuración y en el mensaje de error como último recurso.
func redactArgs(args []string) string {
	sensibles := map[string]bool{
		"password": true, "psk": true, "802-11-wireless-security.psk": true,
		"vpn.secrets": true, "key": true, "secret": true,
	}
	out := make([]string, 0, len(args))
	ocultarSiguiente := false
	for _, a := range args {
		switch {
		case ocultarSiguiente:
			out = append(out, "****")
			ocultarSiguiente = false
		// El valor embebido (`key=secret`) se revisa ANTES de la lista de
		// claves: si no, `vpn.secrets` matchea primero y el `****` tapa la
		// clave entera, perdiendo el contexto de qué se estaba configurando.
		case strings.Contains(strings.ToLower(a), "password="):
			i := strings.Index(strings.ToLower(a), "password=")
			out = append(out, a[:i+len("password=")]+"****")
		case strings.Contains(strings.ToLower(a), "psk="):
			i := strings.Index(strings.ToLower(a), "psk=")
			out = append(out, a[:i+len("psk=")]+"****")
		case sensibles[strings.ToLower(a)]:
			out = append(out, a)
			ocultarSiguiente = true
		default:
			out = append(out, a)
		}
	}
	return strings.Join(out, " ")
}

// ErrorConDetalle arma un error con mensaje de usuario. Lo usan las capas que
// necesitan envolver el fallo con contexto (p. ej. "crear conexión OpenVPN").
func ErrorConDetalle(contexto string, err error) error {
	if err == nil {
		return nil
	}
	if msg := userMessage(err); msg != "" {
		return fmt.Errorf("%s: %s", contexto, msg)
	}
	return fmt.Errorf("%s: %w", contexto, err)
}
