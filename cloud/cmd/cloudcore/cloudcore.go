package main

import (
	"os"

	"github.com/kubeedge/kubeedge/cloud/cmd/cloudcore/app"
)

func main() {
	command := app.NewCloudCoreCommand()

	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
