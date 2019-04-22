package main

import (
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
	_ "github.com/kubeedge/kubeedge/edge/pkg/devicetwin"
	_ "github.com/kubeedge/kubeedge/edgesite/pkg/controller"
	_ "github.com/kubeedge/kubeedge/edge/pkg/edged"
	_ "github.com/kubeedge/kubeedge/edge/pkg/eventbus"
	_ "github.com/kubeedge/kubeedge/edge/pkg/metamanager"
)

func main() {
	dbm.InitDBManager()
	core.Run()
}
