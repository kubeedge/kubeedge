package main

import (
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller"
	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
	_ "github.com/kubeedge/kubeedge/edge/pkg/edged"
	_ "github.com/kubeedge/kubeedge/edge/pkg/metamanager"
)

func main() {
	edgecontroller.Register()
	dbm.InitDBManager()
	core.Run()
}
