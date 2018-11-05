package main

import (
	"kubeedge/beehive/pkg/core"

	"kubeedge/pkg/common/dbm"
	_ "kubeedge/pkg/devicetwin"
	_ "kubeedge/pkg/edged"
	_ "kubeedge/pkg/edgehub"
	_ "kubeedge/pkg/eventbus"
	_ "kubeedge/pkg/metamanager"
	_ "kubeedge/test"
	// _ "kubeedge/pkg/edgefunction"
)

func main() {
	dbm.InitDBManager()
	core.Run()
}
