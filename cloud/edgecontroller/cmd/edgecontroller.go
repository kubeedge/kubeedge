package main

import (
	_ "github.com/kubeedge/kubeedge/cloud/edgecontroller/pkg/cloudhub"
	_ "github.com/kubeedge/kubeedge/cloud/edgecontroller/pkg/controller"

	"github.com/kubeedge/beehive/pkg/core"
)

func main() {
	core.Run()
}
