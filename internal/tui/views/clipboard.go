package views

import (
	"fmt"
	"os/exec"
	"strings"
)

// clipboardCopy deja `texto` en el portapapeles del sistema. Prueba, en orden,
// las herramientas habituales en Linux (Wayland primero, después X11/macOS).
// El texto se pasa por stdin, nunca como argumento, para que no quede en la
// lista de procesos ni en el historial del shell.
func clipboardCopy(texto string) error {
	candidatos := [][]string{
		{"wl-copy"},
		{"xclip", "-selection", "clipboard"},
		{"xsel", "--clipboard", "--input"},
		{"pbcopy"},
	}

	var intentos []string
	for _, args := range candidatos {
		if _, err := exec.LookPath(args[0]); err != nil {
			continue
		}
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdin = strings.NewReader(texto)
		if err := cmd.Run(); err != nil {
			intentos = append(intentos, fmt.Sprintf("%s: %v", args[0], err))
			continue
		}
		return nil
	}

	if len(intentos) == 0 {
		return fmt.Errorf("instala wl-clipboard, xclip o xsel")
	}
	return fmt.Errorf("%s", strings.Join(intentos, "; "))
}
