package main

import (
	"os"

	"github.com/kubeedge/kubeedge/edge/cmd/edgecore/app"
)

func main() {
	command := app.NewEdgeCoreCommand()

	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
