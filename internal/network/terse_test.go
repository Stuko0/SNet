package network

import (
	"reflect"
	"testing"
)

func TestSplitTerseDesescapa(t *testing.T) {
	casos := []struct {
		in   string
		want []string
		desc string
	}{
		{"a:b:c", []string{"a", "b", "c"}, "sin escapes"},
		{`Casa\:5G:802-11-wireless`, []string{"Casa:5G", "802-11-wireless"}, "dos puntos en el nombre"},
		{`WiFi\: invitados:wifi:conectado`, []string{"WiFi: invitados", "wifi", "conectado"}, "dos puntos con espacio"},
		{`back\\slash:wifi`, []string{`back\slash`, "wifi"}, "barra invertida escapada"},
		{`a\\\:b:c`, []string{`a\:b`, "c"}, "barra escapada seguida de dos puntos escapado"},
		{``, []string{""}, "vacío"},
		{`:`, []string{"", ""}, "sólo separador"},
		{`foo\:bar`, []string{"foo:bar"}, "sin campo extra"},
		{`trailing\`, []string{`trailing\`}, "barra final suelta"},
		{`Casa:AA\:BB\:CC:WPA2:90`, []string{"Casa", "AA:BB:CC", "WPA2", "90"}, "BSSID con dos puntos"},
	}
	for _, c := range casos {
		got := splitTerse(c.in)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: splitTerse(%q) = %#v, se esperaba %#v", c.desc, c.in, got, c.want)
		}
	}
}

// TestSplitTerseNoRompeElCasoComun: la mayoría de las líneas no tienen escapes
// y deben comportarse exactamente como strings.Split.
func TestSplitTerseNoRompeElCasoComun(t *testing.T) {
	lineas := []string{
		"Blacksoul cafe:802-11-wireless:wlan0:yes",
		"tailscale0:tun:tailscale0:yes",
		"lo:loopback:lo:yes",
		"wlan0:wifi:connected:Casa",
	}
	for _, l := range lineas {
		got := splitTerse(l)
		want := splitSimple(l)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("splitTerse(%q) = %#v, se esperaba %#v (igual que Split)", l, got, want)
		}
	}
}

// TestSplitTerseNConservaValoresConDosPuntos: para `clave:valor` el valor puede
// contener `:` legítimamente (DNS, rutas, endpoints).
func TestSplitTerseNConservaValoresConDosPuntos(t *testing.T) {
	casos := []struct {
		in   string
		n    int
		want []string
	}{
		{"802-11-wireless.ssid:Casa", 2, []string{"802-11-wireless.ssid", "Casa"}},
		{"ipv4.dns:1.1.1.1,8.8.8.8", 2, []string{"ipv4.dns", "1.1.1.1,8.8.8.8"}},
		{"vpn.data:remote=vpn.example.com,port=1194", 2, []string{"vpn.data", "remote=vpn.example.com,port=1194"}},
		{`a:b\:c`, 2, []string{"a", "b:c"}},
		{"solo-un-campo", 2, []string{"solo-un-campo"}},
	}
	for _, c := range casos {
		got := splitTerseN(c.in, c.n)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("splitTerseN(%q, %d) = %#v, se esperaba %#v", c.in, c.n, got, c.want)
		}
	}
}

// splitSimple replica el viejo comportamiento sólo para el test comparativo.
func splitSimple(s string) []string {
	var parts []string
	cur := ""
	for _, r := range s {
		if r == ':' {
			parts = append(parts, cur)
			cur = ""
			continue
		}
		cur += string(r)
	}
	return append(parts, cur)
}
