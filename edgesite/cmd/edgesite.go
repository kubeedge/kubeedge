package main

import (
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
	_ "github.com/kubeedge/kubeedge/edge/pkg/edged"
	_ "github.com/kubeedge/kubeedge/edge/pkg/metamanager"
	_ "github.com/kubeedge/kubeedge/edgesite/pkg/controller"
)

func main() {
	dbm.InitDBManager()
	core.Run()
}
