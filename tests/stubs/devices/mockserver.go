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
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/tests/stubs/devices/services"
)

const openBeaconUUID = "AA6062F098CA42118EC4193EB73CCEB6"

var timeDuration *int

func createServiceAndAdvertise(d gatt.Device, s gatt.State) {
	// Setup GAP and GATT services for Linux implementation.
	if err := d.AddService(service.NewGapService("SensorTagMock")); err != nil {
		klog.Errorf("failed to add service SensorTagMock, error: %v", err)
	}

	if err := d.AddService(service.NewGattService()); err != nil {
		klog.Errorf("failed to add service, error: %v", err)
	}

	// Creating a temperature reading service
	temperatureSvc := services.NewTemperatureService()
	if err := d.AddService(temperatureSvc); err != nil {
		klog.Errorf("failed to add service, error: %v", err)
	}

	// Advertise device name and service's UUIDs.
	klog.Info("Advertising device name and service UUID")
	if err := d.AdvertiseNameAndServices("mock temp sensor model", []gatt.UUID{temperatureSvc.UUID()}); err != nil {
		klog.Errorf("failed to mock temp sensor model, error: %v", err)
	}

	// Advertise as an OpenBeacon iBeacon
	klog.Info("Advertise as an OpenBeacon iBeacon")
	if err := d.AdvertiseIBeacon(gatt.MustParseUUID(openBeaconUUID), 1, 2, -59); err != nil {
		klog.Errorf("Failed to advertise, error: %v", err)
	}
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
	if err := d.Init(createServiceAndAdvertise); err != nil {
		klog.Errorf("Failed to create service, err: %s", err)
	}

	<-ctx.Done()
	klog.Info("Stopping server and cleaning up")
	if err := d.StopAdvertising(); err != nil {
		klog.Fatalf("failed to stop advertising, err: %v", err)
	}
	if err := d.RemoveAllServices(); err != nil {
		klog.Fatalf("failed to remove all services, err: %v", err)
	}
	klog.Info("Stopped advertising and removed all services!!!!")
}
