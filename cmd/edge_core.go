package main

import (
	"github.com/kubeedge/kubeedge/beehive/pkg/core"

	"github.com/kubeedge/kubeedge/pkg/common/dbm"
	_ "github.com/kubeedge/kubeedge/pkg/devicetwin"
	_ "github.com/kubeedge/kubeedge/pkg/edged"
	_ "github.com/kubeedge/kubeedge/pkg/edgehub"
	_ "github.com/kubeedge/kubeedge/pkg/eventbus"
	_ "github.com/kubeedge/kubeedge/pkg/metamanager"
	_ "github.com/kubeedge/kubeedge/test"
	// _ "github.com/kubeedge/kubeedge/pkg/edgefunction"
)

func main() {
	dbm.InitDBManager()
	core.Run()
}
