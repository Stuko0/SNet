package network

import "strings"

// splitTerse divide una línea del modo terse de nmcli (`-t`) respetando sus
// escapes.
//
// nmcli escapa `:` y `\` con una barra invertida en modo terse, así que un
// nombre como `Casa:5G` llega como `Casa\:5G`. Un `strings.Split(line, ":")`
// ingenuo lo parte en `["Casa\", "5G"]` y desalinea todas las columnas, con lo
// que la conexión se muestra con el nombre equivocado y las acciones sobre ella
// fallan.
//
// Ver `man nmcli`, opción `-e | --escape`.
func splitTerse(line string) []string {
	var (
		parts []string
		cur   strings.Builder
	)
	escaped := false
	for _, r := range line {
		switch {
		case escaped:
			// `\:` → `:`, `\\` → `\`. Cualquier otro `\x` se conserva tal cual
			// (nmcli sólo escapa esos dos, pero no queremos perder datos).
			if r != ':' && r != '\\' {
				cur.WriteRune('\\')
			}
			cur.WriteRune(r)
			escaped = false
		case r == '\\':
			escaped = true
		case r == ':':
			parts = append(parts, cur.String())
			cur.Reset()
		default:
			cur.WriteRune(r)
		}
	}
	// Una barra invertida final suelta se conserva (no hay a quién escapar).
	if escaped {
		cur.WriteRune('\\')
	}
	parts = append(parts, cur.String())
	return parts
}

// splitTerseN divide y devuelve como máximo n campos, dejando el resto (ya
// desescapado) en el último. Se usa para `clave:valor`, donde el valor puede
// contener `:` legítimamente (p. ej. un DNS o una ruta).
func splitTerseN(line string, n int) []string {
	if n <= 1 {
		return []string{line}
	}
	all := splitTerse(line)
	if len(all) <= n {
		return all
	}
	out := make([]string, 0, n)
	out = append(out, all[:n-1]...)
	out = append(out, strings.Join(all[n-1:], ":"))
	return out
}
