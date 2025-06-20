package main

import (
	"context"
	"os"
	"runtime/debug"

	"github.com/charmbracelet/fang"
	"github.com/nikumar1206/loco/cmd"
)

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

	if err := fang.Execute(context.Background(), cmd.RootCmd, fang.WithVersion(i.Main.Version)); err != nil {
		os.Exit(1)
	}
}
