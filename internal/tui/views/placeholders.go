package views

import (
	"github.com/Stuko0/SNet/internal/tui/theme"

	tea "github.com/charmbracelet/bubbletea"
)

// WifiListModel placeholder — se implementa en Fase 2
type WifiListModel struct {
}

func NewWifiList() WifiListModel {
	return WifiListModel{}
}

func (m WifiListModel) Init() tea.Cmd { return nil }

func (m WifiListModel) Update(msg tea.Msg) (WifiListModel, tea.Cmd) {
	return m, nil
}

func (m WifiListModel) View() string {
	return theme.CardStyle.Render(
		theme.CardTitleStyle.Render("📶 Redes Wi-Fi") + "\n\n"+
			"    Escaneo de redes disponible en Fase 2.\n\n"+
			"    Presiona Tab para cambiar de vista.",
	)
}

// SavedModel placeholder — Fase 3
type SavedModel struct{}

func NewSaved() SavedModel { return SavedModel{} }
func (m SavedModel) Init() tea.Cmd { return nil }

func (m SavedModel) Update(msg tea.Msg) (SavedModel, tea.Cmd) {
	return m, nil
}

func (m SavedModel) View() string {
	return theme.CardStyle.Render(
		theme.CardTitleStyle.Render("💾 Conexiones Guardadas") + "\n\n"+
			"    Gestión de conexiones disponible en Fase 3.",
	)
}

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

// HotspotModel placeholder — Fase 4
type HotspotModel struct{}

func NewHotspot() HotspotModel { return HotspotModel{} }
func (m HotspotModel) Init() tea.Cmd { return nil }

func (m HotspotModel) Update(msg tea.Msg) (HotspotModel, tea.Cmd) {
	return m, nil
}

func (m HotspotModel) View() string {
	return theme.CardStyle.Render(
		theme.CardTitleStyle.Render("🔥 Hotspot") + "\n\n"+
			"    Creación y control de hotspot disponible en Fase 4.",
	)
}
