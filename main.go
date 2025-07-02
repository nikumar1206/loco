package main

import (
	"context"
	"image/color"
	"os"
	"runtime/debug"

	"github.com/charmbracelet/fang"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/nikumar1206/loco/cmd"
)

// LocoColorScheme is the Southern Pacific 4449–inspired color scheme generator.
func LocoColorScheme() fang.ColorSchemeFunc {
	return func(ldf lipgloss.LightDarkFunc) fang.ColorScheme {
		return fang.ColorScheme{
			Base:           ldf(lipgloss.Color("#FAFAFA"), lipgloss.Color("#1B1B1B")), // Smoke / Coal
			Title:          ldf(lipgloss.Color("#D23A2E"), lipgloss.Color("#F57900")), // Red (light) / Orange (dark)
			Description:    ldf(lipgloss.Color("#4B4B4B"), lipgloss.Color("#D8D8D8")), // Muted / Steel
			Codeblock:      ldf(lipgloss.Color("#E5E5E5"), lipgloss.Color("#1C2C3C")), // Light Grey / Deep Coal Blue
			Program:        ldf(lipgloss.Color("#F57900"), lipgloss.Color("#F57900")), // Orange constant
			DimmedArgument: ldf(lipgloss.Color("#AAAAAA"), lipgloss.Color("#888888")), // Grey / Dim Grey
			Comment:        ldf(lipgloss.Color("#999999"), lipgloss.Color("#666666")), // Greyish
			Flag:           ldf(lipgloss.Color("#F57900"), lipgloss.Color("#F57900")), // Orange
			FlagDefault:    ldf(lipgloss.Color("#D8D8D8"), lipgloss.Color("#AAAAAA")), // Steel / Grey
			Command:        ldf(lipgloss.Color("#D23A2E"), lipgloss.Color("#F57900")), // Red / Orange
			QuotedString:   ldf(lipgloss.Color("#04B575"), lipgloss.Color("#04B575")), // Green (highlighted string)
			Argument:       ldf(lipgloss.Color("#5DD6FF"), lipgloss.Color("#5DD6FF")), // Cyan
			Help:           ldf(lipgloss.Color("#AAAAAA"), lipgloss.Color("#888888")), // Dim
			Dash:           ldf(lipgloss.Color("#F57900"), lipgloss.Color("#F57900")), // Orange
			ErrorHeader: [2]color.Color{
				ldf(lipgloss.Color("#FFFFFF"), lipgloss.Color("#FFFFFF")), // White foreground
				ldf(lipgloss.Color("#D23A2E"), lipgloss.Color("#D23A2E")), // Red background
			},
			ErrorDetails: ldf(lipgloss.Color("#D23A2E"), lipgloss.Color("#F57900")), // Red / Orange
		}
	}
}

func main() {
	i, ok := debug.ReadBuildInfo()
	if !ok {
		i = &debug.BuildInfo{
			Main: debug.Module{
				Path:    "github.com/nikumar1206/loco",
				Version: "v0.0.1",
			},
		}
	}

	if err := fang.Execute(context.Background(),
		cmd.RootCmd,
		fang.WithVersion(i.Main.Version),
		fang.WithColorSchemeFunc(LocoColorScheme())); err != nil {
		os.Exit(1)
	}
}
