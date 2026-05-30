package theme

import (
	"github.com/charmbracelet/lipgloss"
)

// Colores primarios
var (
	ColorPrimary   = lipgloss.AdaptiveColor{Light: "#2563EB", Dark: "#60A5FA"}  // Azul
	ColorSecondary = lipgloss.AdaptiveColor{Light: "#059669", Dark: "#34D399"}  // Verde
	ColorDanger    = lipgloss.AdaptiveColor{Light: "#DC2626", Dark: "#F87171"}  // Rojo
	ColorWarning   = lipgloss.AdaptiveColor{Light: "#D97706", Dark: "#FBBF24"}  // Amarillo
	ColorMuted     = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}  // Gris
	ColorText      = lipgloss.AdaptiveColor{Light: "#1F2937", Dark: "#F3F4F6"}  // Texto
	ColorSubtle    = lipgloss.AdaptiveColor{Light: "#9CA3AF", Dark: "#4B5563"}  // Texto sutil
	ColorSurface   = lipgloss.AdaptiveColor{Light: "#F3F4F6", Dark: "#1F2937"}  // Fondo tarjetas
	ColorBorder    = lipgloss.AdaptiveColor{Light: "#D1D5DB", Dark: "#374151"}  // Bordes
)

// Estilos globales
var (
	AppStyle = lipgloss.NewStyle().
		Padding(0, 1)

	TitleStyle = lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true).
		MarginLeft(1)

	LogoStyle = lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true)

	// Tabs
	TabStyle = lipgloss.NewStyle().
		Padding(0, 2).
		Foreground(ColorSubtle)

	ActiveTabStyle = lipgloss.NewStyle().
    	Padding(0, 2).
		Foreground(ColorPrimary).
		Border(lipgloss.Border{
			Top:         "─",
			Bottom:      " ",
			Left:        "│",
			Right:       "│",
			TopLeft:     "╭",
			TopRight:    "╮",
			BottomLeft:  "│",
			BottomRight: "│",
		}, false).
		BorderForeground(ColorPrimary).
		Bold(true)

	TabSeparator = lipgloss.NewStyle().
			Foreground(ColorBorder).
			SetString("│")

	// Barra de estado (footer)
	FooterStyle = lipgloss.NewStyle().
			Background(ColorPrimary).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1)

	FooterKeyStyle = lipgloss.NewStyle().
			Background(ColorPrimary).
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true)

	FooterDescStyle = lipgloss.NewStyle().
			Background(ColorPrimary).
			Foreground(lipgloss.AdaptiveColor{Light: "#BFDBFE", Dark: "#1E3A5F"})

	// Tarjetas / paneles
	CardStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Padding(1, 2).
		MarginBottom(1)

	CardTitleStyle = lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true).
		MarginBottom(1)

	// Labels y valores
	LabelStyle = lipgloss.NewStyle().
		Foreground(ColorSubtle).
		Width(14).
		Align(lipgloss.Right)

	ValueStyle = lipgloss.NewStyle().
		Foreground(ColorText)

	// Estados
	StatusOnline  = lipgloss.NewStyle().Foreground(ColorSecondary).SetString("●")
	StatusOffline = lipgloss.NewStyle().Foreground(ColorDanger).SetString("●")
	StatusLimited = lipgloss.NewStyle().Foreground(ColorWarning).SetString("●")

	// Error / Success banners
	ErrorStyle = lipgloss.NewStyle().
		Foreground(ColorDanger).
		Bold(true)

	SuccessStyle = lipgloss.NewStyle().
		Foreground(ColorSecondary).
		Bold(true)

	WarningStyle = lipgloss.NewStyle().
		Foreground(ColorWarning).
		Bold(true)

	// Help overlay
	HelpStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Padding(1, 2)

	HelpKeyStyle = lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true)

	HelpDescStyle = lipgloss.NewStyle().
		Foreground(ColorText)

	ToastStyle = lipgloss.NewStyle().
		Padding(0, 2).
		MarginTop(1)
)

// TabTitles son los nombres de las pestañas
var TabTitles = []string{
	" 📡 Estado ",
	" 📶 Wi-Fi  ",
	" 💾 Guardadas ",
	" 🔒 VPN   ",
	" 🔥 Hotspot ",
}
