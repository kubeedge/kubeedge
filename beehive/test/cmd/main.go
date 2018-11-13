package main

import (
	"github.com/kubeedge/kubeedge/beehive/pkg/core"
	_ "github.com/kubeedge/kubeedge/beehive/test/modules"
)

func main() {
	core.Run()
}
