package main

import (
	"edge-core/beehive/pkg/core"

	"edge-core/pkg/common/dbm"
	_ "edge-core/pkg/devicetwin"
	_ "edge-core/pkg/edged"
	_ "edge-core/pkg/edgehub"
	_ "edge-core/pkg/eventbus"
	_ "edge-core/pkg/metamanager"
	_ "edge-core/test"
	// _ "edge-core/pkg/edgefunction"
)

func main() {
	dbm.InitDBManager()
	core.Run()
}
