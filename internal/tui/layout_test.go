package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TestLayoutResponsive verifica que el layout principal no se rompa ni quede
// pegado a la izquierda en terminales anchas (la ventana del usuario es
// kitty 1000px @ font 12 ≈ 150 columnas).
func TestLayoutResponsive(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {100, 40}, {150, 40}, {220, 50}} {
		w, h := size[0], size[1]
		m := NewModel()
		var tm tea.Model = m
		tm, _ = tm.Update(tea.WindowSizeMsg{Width: w, Height: h})

		// La tab 2 es Guardadas; el contenido debe venir ya poblado en SizeMsg.
		for tab := 0; tab < len(tabTitlesForTest()); tab++ {
			mm := tm.(Model)
			mm.activeTab = tab
			v := mm.View()

			maxW, minW := 0, 1<<30
			for _, l := range strings.Split(v, "\n") {
				lw := lipgloss.Width(l)
				if lw > maxW {
					maxW = lw
				}
				if lw < minW {
					minW = lw
				}
			}
			fmt.Printf("size=%dx%d tab=%d maxLine=%d\n", w, h, tab, maxW)

			// El layout no debe exceder el ancho de la terminal (con 1 de
			// holgura para el redondeo del borde).
			if maxW > w {
				t.Errorf("size %dx%d tab %d: línea de %d columnas desborda la terminal", w, h, tab, maxW)
			}
			// Y la tarjeta no debe quedar diminuta en una terminal ancha.
			if w >= 100 && maxW < 60 {
				t.Errorf("size %dx%d tab %d: contenido de solo %d columnas (layout colapsado)", w, h, tab, maxW)
			}
		}
	}
}

func tabTitlesForTest() []string { return []string{"a", "b", "c", "d", "e"} }
