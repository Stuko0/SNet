package views

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// nmcliFalso instala un nmcli de mentira en un PATH temporal que registra las
// llamadas, para poder verificar el ORDEN y el contenido de los `connection
// modify` sin tocar la red real.
//
// Devuelve la ruta del log de llamadas.
func nmcliFalso(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	log := filepath.Join(dir, "calls.log")

	script := `#!/bin/bash
echo "$*" >> ` + log + `
if [ "$1" = "--version" ]; then echo "nmcli fake"; exit 0; fi
case "$*" in
  *"connection show"*)
    # Valores precargados de prueba, incluido IPv6.
    printf '802-11-wireless-security.psk:secreto\n'
    printf 'ipv4.method:auto\n'
    printf 'ipv4.addresses:192.168.1.50/24\n'
    printf 'ipv6.method:auto\n'
    printf 'connection.autoconnect:yes\n'
    ;;
  *) : ;;
esac
`
	path := filepath.Join(dir, "nmcli")
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return log
}

func leerLog(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var out []string
	for _, l := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if l != "" {
			out = append(out, l)
		}
	}
	return out
}

// TestBuildFieldsIncluyeIPv6: el editor debe poder configurar IPv6, que era la
// brecha principal para considerarlo completo.
func TestBuildFieldsIncluyeIPv6(t *testing.T) {
	fields := buildFields("ethernet")

	porSetting := map[string]Field{}
	for _, f := range fields {
		porSetting[f.Setting] = f
	}

	for _, want := range []string{"ipv6.addresses", "ipv6.gateway", "ipv6.dns", "ipv4.routes"} {
		if _, ok := porSetting[want]; !ok {
			t.Errorf("falta el campo %q en el editor", want)
		}
	}

	// Los placeholders deben mostrar el formato esperado.
	if ph := porSetting["ipv6.addresses"].Input.Placeholder; !strings.Contains(ph, "2001:db8") {
		t.Errorf("placeholder de IPv6 poco claro: %q", ph)
	}
	if ph := porSetting["ipv4.routes"].Input.Placeholder; !strings.Contains(ph, "10.0.0.0/8") {
		t.Errorf("placeholder de rutas poco claro: %q", ph)
	}
}

// TestSettingsToLoadPideMetodos: sin pedir ipv4.method/ipv6.method no se puede
// saber si la conexión usa DHCP o IP fija.
func TestSettingsToLoadPideMetodos(t *testing.T) {
	got := settingsToLoad("wifi")
	joined := strings.Join(got, ",")
	for _, want := range []string{"ipv4.method", "ipv6.method", "ipv6.addresses", "ipv4.routes", "connection.id"} {
		if !strings.Contains(joined, want) {
			t.Errorf("settingsToLoad no pide %q: %v", want, got)
		}
	}
}

// TestSaveConnectionPoneMetodoManualAntesDeLosAddresses es el test clave: nmcli
// rechaza `ipv6.addresses` si el método sigue en auto, así que el método debe
// aplicarse PRIMERO.
func TestSaveConnectionPoneMetodoManualAntesDeLosAddresses(t *testing.T) {
	log := nmcliFalso(t)

	fields := buildFields("ethernet")
	for i := range fields {
		switch fields[i].Setting {
		case "connection.id":
			fields[i].Input.SetValue("mi-red")
		case "ipv4.addresses":
			fields[i].Input.SetValue("192.168.1.100/24")
		case "ipv6.addresses":
			fields[i].Input.SetValue("2001:db8::5/64")
		}
	}

	msg := saveConnection("mi-red", fields)
	if sm, ok := msg.(editorSaveMsg); ok && sm.err != nil {
		t.Fatalf("saveConnection devolvió error: %v", sm.err)
	}

	llamadas := leerLog(t, log)

	// Debe haberse puesto cada método ANTES de sus direcciones.
	idxIPv4Method := indexOfContains(llamadas, "ipv4.method manual")
	idxIPv4Addr := indexOfContains(llamadas, "ipv4.addresses 192.168.1.100/24")
	idxIPv6Method := indexOfContains(llamadas, "ipv6.method manual")
	idxIPv6Addr := indexOfContains(llamadas, "ipv6.addresses 2001:db8::5/64")

	if idxIPv4Method < 0 {
		t.Fatalf("no se puso ipv4.method manual. llamadas: %v", llamadas)
	}
	if idxIPv6Method < 0 {
		t.Fatalf("no se puso ipv6.method manual. llamadas: %v", llamadas)
	}
	if idxIPv4Addr < 0 || idxIPv6Addr < 0 {
		t.Fatalf("no se guardaron las direcciones. llamadas: %v", llamadas)
	}
	if idxIPv4Method > idxIPv4Addr {
		t.Errorf("ipv4.method se aplicó DESPUÉS de ipv4.addresses (nmcli lo rechaza)")
	}
	if idxIPv6Method > idxIPv6Addr {
		t.Errorf("ipv6.method se aplicó DESPUÉS de ipv6.addresses (nmcli lo rechaza)")
	}
}

// TestSaveConnectionNoTocaMetodoSinDirecciones: si no se configura IP fija, no
// hay que forzar `manual` (rompería el DHCP).
func TestSaveConnectionNoTocaMetodoSinDirecciones(t *testing.T) {
	log := nmcliFalso(t)

	fields := buildFields("ethernet")
	for i := range fields {
		// Sólo cambiar el nombre: nada de IPs.
		if fields[i].Setting == "connection.id" {
			fields[i].Input.SetValue("solo-nombre")
		}
	}

	_ = saveConnection("solo-nombre", fields)

	for _, l := range leerLog(t, log) {
		if strings.Contains(l, "method manual") {
			t.Errorf("no debería forzar manual sin direcciones: %q", l)
		}
	}
}

// TestSaveConnectionNoGuardaCamposVacios: dejar un campo en blanco no debe
// mandar un modify vacío (borraría la config existente).
func TestSaveConnectionNoGuardaCamposVacios(t *testing.T) {
	log := nmcliFalso(t)

	fields := buildFields("ethernet")
	for i := range fields {
		if fields[i].Setting == "connection.id" {
			fields[i].Input.SetValue("x")
		}
		// El resto queda vacío.
	}

	_ = saveConnection("x", fields)

	for _, l := range leerLog(t, log) {
		if strings.Contains(l, "connection modify x ipv6") ||
			strings.Contains(l, "connection modify x ipv4") {
			t.Errorf("se envió un modify para un campo vacío: %q", l)
		}
	}
}

func indexOfContains(items []string, sub string) int {
	for i, s := range items {
		if strings.Contains(s, sub) {
			return i
		}
	}
	return -1
}
