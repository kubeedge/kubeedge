package virtualdevice

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

const (
	DeviceNameDefault = "edge0"
)

var (
	nh *netlink.Handle
)

func CreateDevice() error {
	//if device is exist,delete and create a new one
	//make sure the latest configuration
	DestroyDevice()

	edge0 := &netlink.Dummy{
		LinkAttrs: netlink.LinkAttrs{
			Name: DeviceNameDefault,
		},
	}
	return nh.LinkAdd(edge0)
}

// AddIP Bind ip to the default device
func AddIP(address string) error {
	dev, err := nh.LinkByName(DeviceNameDefault)
	if err != nil {
		return fmt.Errorf("[L4 Proxy] Device %s is not exist!!", DeviceNameDefault)
	}

	addr := net.ParseIP(address)
	if addr == nil {
		return fmt.Errorf("[L4 Proxy] AddIP address is invalid ")
	}

	if err := nh.AddrAdd(dev, &netlink.Addr{IPNet: netlink.NewIPNet(addr)}); err != nil {
		if err == unix.EEXIST {
			return nil
		}
		return err
	}

	return nil
}

// DestroyDevice implement release the device
func DestroyDevice() error {
	_, err := net.InterfaceByName(DeviceNameDefault)
	if err != nil {
		return err
	}

	link, err := nh.LinkByName(DeviceNameDefault)
	edge0, ok := link.(*netlink.Dummy)
	if !ok {
		return fmt.Errorf("")
	}

	return nh.LinkDel(edge0)
}

func init() {
	nh = &netlink.Handle{}
}
