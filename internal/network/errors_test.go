package network

import (
	"strings"
	"testing"
	"time"
)

// TestNmcliErrorNoFiltraSecretos es el test central de la fase: el mensaje de
// error nunca debe contener la contraseña ni el comando completo.
func TestNmcliErrorNoFiltraSecretos(t *testing.T) {
	casos := []struct {
		desc string
		args []string
	}{
		{"password de wifi", []string{"device", "wifi", "connect", "Casa", "password", "hunter2"}},
		{"psk de conexión", []string{"connection", "add", "type", "wifi", "802-11-wireless-security.psk", "hunter2"}},
		{"secretos vpn", []string{"connection", "modify", "vpn1", "vpn.secrets", "password=hunter2"}},
	}
	for _, c := range casos {
		err := &NmcliError{
			Args:     c.args,
			ExitCode: 4,
			Stderr:   "Error: Connection activation failed: (7) Secrets were required, but not provided.",
		}
		msg := err.Error()
		if strings.Contains(msg, "hunter2") {
			t.Errorf("%s: FUGA de la contraseña en Error(): %q", c.desc, msg)
		}
		if strings.Contains(msg, "device wifi connect") || strings.Contains(msg, "vpn.secrets") {
			t.Errorf("%s: FUGA del comando completo en Error(): %q", c.desc, msg)
		}
	}
}

// TestRedactArgsOcultaValoresSensibles: el log de depuración conserva el
// contexto pero tapa el secreto.
func TestRedactArgsOcultaValoresSensibles(t *testing.T) {
	casos := []struct {
		args []string
		want string
	}{
		{
			[]string{"device", "wifi", "connect", "Casa", "password", "hunter2"},
			"device wifi connect Casa password ****",
		},
		{
			[]string{"connection", "modify", "x", "802-11-wireless-security.psk", "secreto"},
			"connection modify x 802-11-wireless-security.psk ****",
		},
		{
			// vpn.secrets es la CLAVE; el valor es el secreto completo.
			[]string{"connection", "modify", "x", "vpn.secrets", "password=hunter2"},
			"connection modify x vpn.secrets ****",
		},
		{
			// Un valor embebido como argumento único sí conserva la etiqueta.
			[]string{"connection", "modify", "x", "password=hunter2"},
			"connection modify x password=****",
		},
		{
			[]string{"connection", "show", "Casa"},
			"connection show Casa",
		},
	}
	for _, c := range casos {
		got := redactArgs(c.args)
		if got != c.want {
			t.Errorf("redactArgs(%v) = %q, se esperaba %q", c.args, got, c.want)
		}
		if strings.Contains(got, "hunter2") || strings.Contains(got, "secreto") {
			t.Errorf("redactArgs filtró el secreto: %q", got)
		}
	}
}

// TestUserMessageTraduceLosErroresConocidos: cada clase de fallo de nmcli debe
// dar un mensaje en español y accionable.
func TestUserMessageTraduceLosErroresConocidos(t *testing.T) {
	casos := []struct {
		stderr string
		want   string
	}{
		{"Error: Connection activation failed: Not authorized to control networking.", msgSinPermisos},
		{"Error: Could not create polkit authentication agent.", msgSinPermisos},
		{"Error: Secrets were required, but not provided.", msgPwdIncorrecta},
		{"Error: 802-1x authentication failed.", msgPwdIncorrecta},
		{"Error: No network with SSID 'X' found.", msgNoEncontrada},
		{"Error: unknown connection 'foo'.", msgNoEncontrada},
		{"Error: device is busy.", msgHardwareOcup},
		{"Error: Connection 'x' is already active on device 'wlan0'.", msgYaActiva},
		{"Error: Could not connect to NetworkManager.", msgSinNM},
		{"Error: timed out waiting for activation.", msgTimeout},
	}
	for _, c := range casos {
		err := &NmcliError{Args: []string{"x"}, Stderr: c.stderr}
		got := userMessage(err)
		if got != c.want {
			t.Errorf("stderr %q\n  got  %q\n  want %q", c.stderr, got, c.want)
		}
		// Todos los mensajes conocidos deben estar en español.
		if strings.Contains(got, "Error:") {
			t.Errorf("stderr %q: el mensaje conserva el prefijo en inglés: %q", c.stderr, got)
		}
	}
}

// TestUserMessageTimeoutTienePrioridad: un error con deadline agotado se
// reporta como timeout aunque el stderr diga otra cosa.
func TestUserMessageTimeoutTienePrioridad(t *testing.T) {
	err := &NmcliError{
		Args:    []string{"device", "wifi", "connect", "Casa", "password", "hunter2"},
		Timeout: 5 * time.Second,
		Stderr:  "algo irrelevante",
	}
	if got := userMessage(err); got != msgTimeout {
		t.Errorf("userMessage() = %q, se esperaba el mensaje de timeout", got)
	}
	if msg := err.Error(); strings.Contains(msg, "hunter2") {
		t.Errorf("FUGA con timeout: %q", msg)
	}
}

// TestUserMessageDesconocidoNoExponeArgumentos: un stderr no reconocido se
// muestra recortado pero sin argumentos.
func TestUserMessageDesconocidoNoExponeArgumentos(t *testing.T) {
	err := &NmcliError{
		Args:   []string{"device", "wifi", "connect", "Casa", "password", "hunter2"},
		Stderr: "Error: something entirely unexpected happened on device wlan0",
	}
	msg := userMessage(err)
	if strings.Contains(msg, "hunter2") {
		t.Errorf("FUGA en fallback: %q", msg)
	}
	if !strings.Contains(msg, "unexpected") {
		t.Errorf("se perdió el detalle útil: %q", msg)
	}
	if strings.HasPrefix(msg, "Error: ") {
		t.Errorf("no se limpió el prefijo inglés: %q", msg)
	}
}

// TestSanitizeStderrRecorta: no dejar que un stderr gigante rompa la UI.
func TestSanitizeStderrRecorta(t *testing.T) {
	largo := strings.Repeat("x", 500)
	got := sanitizeStderr(largo)
	// Se mide en runas (no bytes): la elipsis ocupa 3 bytes en UTF-8.
	if n := len([]rune(got)); n != 160 {
		t.Errorf("sanitizeStderr devolvió %d runas, se esperaban 160", n)
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("se esperaba elipsis, got %q", got)
	}
	// Los espacios múltiples se colapsan.
	if got := sanitizeStderr("a  \n b\t\tc"); got != "a b c" {
		t.Errorf("sanitizeStderr no colapsó espacios: %q", got)
	}
}

// TestDebugConservaContexto: para depurar sí queremos el comando, redactado.
func TestDebugConservaContexto(t *testing.T) {
	err := &NmcliError{
		Args:     []string{"device", "wifi", "connect", "Casa", "password", "hunter2"},
		ExitCode: 4,
		Stderr:   "Error: Connection activation failed",
	}
	d := err.Debug()
	if strings.Contains(d, "hunter2") {
		t.Errorf("Debug() filtró el secreto: %q", d)
	}
	for _, want := range []string{"nmcli", "connect", "Casa", "exit 4", "activation failed"} {
		if !strings.Contains(d, want) {
			t.Errorf("Debug() perdió %q: %q", want, d)
		}
	}
}
