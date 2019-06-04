package main

import (
	"github.com/kubeedge/beehive/pkg/core"
	_ "github.com/kubeedge/kubeedge/cloud/pkg/controller"
	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
	_ "github.com/kubeedge/kubeedge/edge/pkg/edged"
	_ "github.com/kubeedge/kubeedge/edge/pkg/metamanager"
)

func main() {
	dbm.InitDBManager()
	core.Run()
}
