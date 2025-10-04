package main

import (
	"context"
	"image/color"
	"os"
	runtimeDebug "runtime/debug"

	"github.com/charmbracelet/fang"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/nikumar1206/loco/internal/ui"
)

// LocoColorScheme is the Southern Pacific 4449–inspired color scheme generator.
func LocoColorScheme() fang.ColorSchemeFunc {
	return func(ldf lipgloss.LightDarkFunc) fang.ColorScheme {
		return fang.ColorScheme{
			Base:           ldf(ui.LocoLightGray, ui.LocoDarkGray),
			Title:          ldf(ui.LocoRed, ui.LocoOrange),
			Description:    ldf(ui.LocoMuted, ui.LocoSteel),
			Codeblock:      ldf(ui.LocoLightGrey, ui.LocoDeepCoal),
			Program:        ldf(ui.LocoOrange, ui.LocoOrange),
			DimmedArgument: ldf(ui.LocoDimGrey, ui.LocoMidGrey),
			Comment:        ldf(ui.LocoGreyish, ui.LocoDarkGrey),
			Flag:           ldf(ui.LocoOrange, ui.LocoOrange),
			FlagDefault:    ldf(ui.LocoSteel, ui.LocoDimGrey),
			Command:        ldf(ui.LocoRed, ui.LocoOrange),
			QuotedString:   ldf(ui.LocoGreen, ui.LocoGreen),
			Argument:       ldf(ui.LocoCyan, ui.LocoCyan),
			Help:           ldf(ui.LocoDimGrey, ui.LocoMidGrey),
			Dash:           ldf(ui.LocoOrange, ui.LocoOrange),
			ErrorHeader: [2]color.Color{
				ldf(ui.LocoWhite, ui.LocoWhite),
				ldf(ui.LocoRed, ui.LocoRed),
			},
			ErrorDetails: ldf(ui.LocoRed, ui.LocoOrange),
		}
	}
}

func main() {
	i, ok := runtimeDebug.ReadBuildInfo()
	if !ok {
		i = &runtimeDebug.BuildInfo{
			Main: runtimeDebug.Module{
				Path:    "github.com/nikumar1206/loco",
				Version: "v0.0.1",
			},
		}
	}

	if err := fang.Execute(context.Background(),
		RootCmd,
		fang.WithVersion(i.Main.Version),
		fang.WithColorSchemeFunc(LocoColorScheme())); err != nil {
		os.Exit(1)
	}
}
