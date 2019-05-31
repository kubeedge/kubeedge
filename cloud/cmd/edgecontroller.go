package main

import (
	_ "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub"
	_ "github.com/kubeedge/kubeedge/cloud/pkg/controller"
	_ "github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller"
	"github.com/kubeedge/kubeedge/version"

	"github.com/kubeedge/beehive/pkg/core"
)

func main() {
	version.Print()
	core.Run()
}
