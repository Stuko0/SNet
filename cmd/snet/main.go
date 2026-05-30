package main

import (
	"fmt"
	"github.com/Stuko0/SNet/internal/tui"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Verificar que nmcli existe
	if !commandExists("nmcli") {
		fmt.Fprintln(os.Stderr, "Error: nmcli no está instalado.")
		fmt.Fprintln(os.Stderr, "Instálalo con: sudo dnf install NetworkManager-cli")
		os.Exit(1)
	}

	p := tea.NewProgram(
		tui.NewModel(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error al ejecutar SNet: %v\n", err)
		os.Exit(1)
	}
}

func commandExists(cmd string) bool {
	_, err := os.Stat("/usr/bin/" + cmd)
	if err == nil {
		return true
	}
	_, err = os.Stat("/usr/local/bin/" + cmd)
	return err == nil
}
