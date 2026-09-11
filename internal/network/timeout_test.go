package network

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestTimeoutForClasificaComandos(t *testing.T) {
	casos := []struct {
		args []string
		want time.Duration
		desc string
	}{
		{[]string{"general", "status"}, timeoutQuery, "lectura simple"},
		{[]string{"networking", "connectivity"}, timeoutQuery, "conectividad"},
		{[]string{"-t", "-f", "NAME,TYPE", "connection", "show"}, timeoutQuery, "listado"},
		{[]string{"device", "wifi", "list", "--rescan", "yes"}, timeoutConnect, "escaneo"},
		{[]string{"device", "wifi", "connect", "Casa", "password", "x"}, timeoutConnect, "conexión wifi"},
		{[]string{"connection", "modify", "Casa", "ipv4.dns", "1.1.1.1"}, timeoutModify, "modificar"},
		{[]string{"connection", "delete", "Casa"}, timeoutModify, "borrar"},
		{[]string{"connection", "up", "Casa"}, timeoutModify, "activar"},
		{[]string{"connection", "down", "Casa"}, timeoutModify, "desactivar"},
		{[]string{"device", "disconnect", "wlan0"}, timeoutModify, "desconectar"},
		{[]string{"radio", "wifi", "off"}, timeoutModify, "radio"},
		{[]string{"connection", "add", "type", "vpn", "vpn-type", "openvpn"}, timeoutVPNActivate, "crear VPN"},
	}
	for _, c := range casos {
		if got := timeoutFor(c.args); got != c.want {
			t.Errorf("%s: timeoutFor(%v) = %v, se esperaba %v", c.desc, c.args, got, c.want)
		}
	}
}

// TestRunCmdTimeoutMataProcesoColgado es el test central de la fase: un comando
// que nunca responde debe cortarse por deadline en vez de bloquear para siempre.
func TestRunCmdTimeoutMataProcesoColgado(t *testing.T) {
	// `sleep` no existe como subcomando de nmcli, pero runCmdTimeout ejecuta el
	// binario "nmcli" hardcodeado. Para probar el corte usamos un binario real
	// que sí cuelga: verificamos que el deadline se respeta sobre un comando
	// de nmcli que fuerza espera.
	//
	// `nmcli` con un timeout de 1ns debe agotarse inmediatamente.
	inicio := time.Now()
	_, err := runCmdTimeout(time.Nanosecond, "general", "status")
	transcurrido := time.Since(inicio)

	if err == nil {
		t.Fatal("se esperaba error por timeout")
	}

	var nerr *NmcliError
	if !errors.As(err, &nerr) {
		t.Fatalf("se esperaba *NmcliError, se obtuvo %T", err)
	}
	if !nerr.IsTimeout() {
		t.Errorf("IsTimeout() = false; el error debería marcar timeout (err=%v)", err)
	}
	if nerr.Timeout != time.Nanosecond {
		t.Errorf("Timeout = %v, se esperaba 1ns", nerr.Timeout)
	}
	// El corte debe ser inmediato, no esperar los 10s del timeout por defecto.
	if transcurrido > 3*time.Second {
		t.Errorf("tardó %v en cortar; debería ser casi inmediato", transcurrido)
	}
}

// TestRunCmdTimeoutRespetaComandoRapido verifica que no rompimos el caso normal.
func TestRunCmdTimeoutRespetaComandoRapido(t *testing.T) {
	out, err := runCmdTimeout(10*time.Second, "general", "status")
	if err != nil {
		t.Fatalf("nmcli general status falló: %v", err)
	}
	if !strings.Contains(out, "STATE") {
		t.Errorf("salida inesperada: %q", out)
	}
}

// TestIsTimeoutFalsoEnErrorNormal: un error de nmcli que no es timeout no debe
// marcarse como tal.
func TestIsTimeoutFalsoEnErrorNormal(t *testing.T) {
	// Un subcomando inexistente hace que nmcli salga con error rápido.
	_, err := runCmdTimeout(5*time.Second, "subcomando-que-no-existe")
	if err == nil {
		t.Fatal("se esperaba error")
	}
	var nerr *NmcliError
	if !errors.As(err, &nerr) {
		t.Fatalf("se esperaba *NmcliError, se obtuvo %T", err)
	}
	if nerr.IsTimeout() {
		t.Errorf("IsTimeout() = true, pero fue un error normal: %v", err)
	}
}
