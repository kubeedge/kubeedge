package main

import (
	"os"

	"github.com/kubeedge/kubeedge/edgesite/cmd/app"
)

func main() {
	command := app.NewEdgeSiteCommand()

	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
