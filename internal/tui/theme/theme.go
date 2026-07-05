package theme

import (
	"github.com/charmbracelet/lipgloss"
)

// Colores primarios (Rosé Pine Dawn/Moon)
var (
	ColorPrimary   = lipgloss.AdaptiveColor{Light: "#907aa9", Dark: "#c4a7e7"} // Iris
	ColorSecondary = lipgloss.AdaptiveColor{Light: "#56949f", Dark: "#9ccfd8"} // Foam (Verde/Cyan pastel)
	ColorDanger    = lipgloss.AdaptiveColor{Light: "#b4637a", Dark: "#eb6f92"} // Love (Rojo/Rosa)
	ColorWarning   = lipgloss.AdaptiveColor{Light: "#ea9d34", Dark: "#f6c177"} // Gold (Amarillo)
	ColorMuted     = lipgloss.AdaptiveColor{Light: "#9893a5", Dark: "#6e6a86"} // Muted
	ColorText      = lipgloss.AdaptiveColor{Light: "#575279", Dark: "#e0def4"} // Text
	ColorSubtle    = lipgloss.AdaptiveColor{Light: "#797593", Dark: "#908caa"} // Subtle
	ColorSurface   = lipgloss.AdaptiveColor{Light: "#fffaf3", Dark: "#2a273f"} // Surface
	ColorBorder    = lipgloss.AdaptiveColor{Light: "#f2e9e1", Dark: "#393552"} // Overlay
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
			Foreground(lipgloss.AdaptiveColor{Light: "#faf4ed", Dark: "#232136"}).
			Background(ColorPrimary).
			Bold(true)

	TabSeparator = lipgloss.NewStyle().
			Foreground(ColorBorder).
			SetString("│")

	// Barra de estado (footer)
	FooterStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Padding(0, 1)

	FooterKeyStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	FooterDescStyle = lipgloss.NewStyle().
			Foreground(ColorSubtle)

	// Tarjetas / paneles
	CardStyle = lipgloss.NewStyle().
			Border(lipgloss.ThickBorder()).
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

	TableStyle = lipgloss.NewStyle().
			MarginLeft(2).
			MarginRight(2)
)

// TabTitles son los nombres de las pestañas
var TabTitles = []string{
	" 󰣺 Estado ",
	" 󰤨 Wi-Fi  ",
	" 󰆓 Guardadas ",
	" 󰒄 VPN   ",
	" 󰈀 Hotspot ",
}
