package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/kubeedge/kubeedge/csidriver/pkg/app"
)

func init() {
	flag.Set("logtostderr", "true")
}

var (
	endpoint   = flag.String("endpoint", "unix://tmp/csi.sock", "CSI endpoint")
	driverName = flag.String("drivername", "csidriver", "name of the driver")
	nodeID     = flag.String("nodeid", "", "node id")
	keEndpoint = flag.String("kubeedgeendpoint", "unix:///kubeedge/kubeedge.sock", "kubeedge endpoint")
	version    = flag.String("version", "", "version")
)

func main() {
	flag.Parse()
	handle()
	os.Exit(0)
}

func handle() {
	driver, err := app.NewCSIDriver(*driverName, *nodeID, *endpoint, *keEndpoint, *version)
	if err != nil {
		fmt.Printf("failed to initialize driver: %s", err.Error())
		os.Exit(1)
	}
	driver.Run()
}
