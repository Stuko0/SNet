package views

import (
	"github.com/Stuko0/SNet/internal/tui/theme"

	tea "github.com/charmbracelet/bubbletea"
)

// VPNListModel placeholder — Fase 5
type VPNListModel struct{}

func NewVPNList() VPNListModel { return VPNListModel{} }
func (m VPNListModel) Init() tea.Cmd { return nil }

func (m VPNListModel) Update(msg tea.Msg) (VPNListModel, tea.Cmd) {
	return m, nil
}

func (m VPNListModel) View() string {
	return theme.CardStyle.Render(
		theme.CardTitleStyle.Render("🔒 VPN") + "\n\n"+
			"    Gestión de VPN disponible en Fase 5.",
	)
}
