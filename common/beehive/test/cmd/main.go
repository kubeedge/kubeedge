package main

import (
	"github.com/kubeedge/kubeedge/common/beehive/pkg/core"
	_ "github.com/kubeedge/kubeedge/common/beehive/test/modules"
)

func main() {
	core.Run()
}
