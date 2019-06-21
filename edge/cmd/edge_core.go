package main

import (
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
	_ "github.com/kubeedge/kubeedge/edge/pkg/devicetwin"
	_ "github.com/kubeedge/kubeedge/edge/pkg/edged"
	_ "github.com/kubeedge/kubeedge/edge/pkg/edgehub"
	_ "github.com/kubeedge/kubeedge/edge/pkg/eventbus"
	_ "github.com/kubeedge/kubeedge/edge/pkg/metamanager"
	_ "github.com/kubeedge/kubeedge/edge/pkg/servicebus"
	_ "github.com/kubeedge/kubeedge/edge/test"
	_ "github.com/kubeedge/kubeedge/edgemesh/pkg"
)

func main() {
	dbm.InitDBManager()
	core.Run()
}
