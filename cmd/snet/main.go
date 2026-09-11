package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/Stuko0/SNet/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {

	if !commandExists("nmcli") {
		fmt.Fprintln(os.Stderr, "Error: nmcli no está instalado.")
		fmt.Fprintln(os.Stderr, "Instálalo con: sudo dnf install NetworkManager-cli")
		os.Exit(1)
	}

	// Al recibir SIGTERM/SIGINT (kill, cierre de la ventana del terminal) hay
	// que guardar la pestaña activa antes de morir; si no, se pierde.
	// Bubble Tea no ejecuta su Update en ese caso.
	escribirConfig := func() {}

	m := tui.NewModel()
	escribirConfig = m.SaveConfigHook()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sig
		escribirConfig()
		os.Exit(0)
	}()

	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		// Guardar también si el programa termina por error.
		escribirConfig()
		fmt.Fprintf(os.Stderr, "Error al ejecutar SNet: %v\n", err)
		os.Exit(1)
	}
	escribirConfig()
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
