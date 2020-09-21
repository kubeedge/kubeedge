package devices

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/paypal/gatt"
	"github.com/paypal/gatt/examples/option"
	"github.com/paypal/gatt/examples/service"
	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/tests/stubs/devices/services"
)

const openBeaconUUID = "AA6062F098CA42118EC4193EB73CCEB6"

var timeDuration *int

func createServiceAndAdvertise(d gatt.Device, s gatt.State) {
	// Setup GAP and GATT services for Linux implementation.
	d.AddService(service.NewGapService("SensorTagMock"))
	d.AddService(service.NewGattService())

	// Creating a temperature reading service
	temperatureSvc := services.NewTemperatureService()
	d.AddService(temperatureSvc)

	// Advertise device name and service's UUIDs.
	klog.Info("Advertising device name and service UUID")
	d.AdvertiseNameAndServices("mock temp sensor model", []gatt.UUID{temperatureSvc.UUID()})

	// Advertise as an OpenBeacon iBeacon
	klog.Info("Advertise as an OpenBeacon iBeacon")
	d.AdvertiseIBeacon(gatt.MustParseUUID(openBeaconUUID), 1, 2, -59)
}

//usage is responsible for setting up the default settings of all defined command-line flags for klog.
func usage() {
	flag.PrintDefaults()
	os.Exit(2)
}

//init for getting command line arguments
func init() {
	flag.Usage = usage
	timeDuration = flag.Int("duration", 5, "time duration for which server should be run")
	flag.Parse()
}

func main() {
	d, err := gatt.NewDevice(option.DefaultServerOptions...)
	if err != nil {
		klog.Fatalf("Failed to open device, err: %s", err)
	}

	// Register optional handlers.
	d.Handle(
		gatt.CentralConnected(func(c gatt.Central) { fmt.Println("Connect: ", c.ID()) }),
		gatt.CentralDisconnected(func(c gatt.Central) { fmt.Println("Disconnect: ", c.ID()) }),
	)

	duration := time.Duration(*timeDuration) * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()
	d.Init(createServiceAndAdvertise)

	<-ctx.Done()
	klog.Info("Stopping server and cleaning up")
	d.StopAdvertising()
	d.RemoveAllServices()
	klog.Info("Stopped advertising and removed all services!!!!")
}
