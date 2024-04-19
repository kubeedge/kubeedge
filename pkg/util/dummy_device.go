/*
Copyright 2024 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
	"k8s.io/klog/v2"
)

type DummyDeviceManager struct {
	netlink.Handle
}

func NewDummyDeviceManager() *DummyDeviceManager {
	return &DummyDeviceManager{netlink.Handle{}}
}

// EnsureDummyDevice ensure dummy device exist
func (d *DummyDeviceManager) EnsureDummyDevice(devName string) (bool, error) {
	_, err := d.LinkByName(devName)
	if err == nil {
		// found dummy device
		return true, nil
	}
	klog.Warningf("No dummy device %s, link it", devName)
	dummy := &netlink.Dummy{
		LinkAttrs: netlink.LinkAttrs{Name: devName},
	}
	return false, d.LinkAdd(dummy)
}

// DeleteDummyDevice delete dummy device.
func (d *DummyDeviceManager) DeleteDummyDevice(devName string) error {
	link, err := d.LinkByName(devName)
	if err != nil {
		_, ok := err.(netlink.LinkNotFoundError)
		if ok {
			return nil
		}
		return fmt.Errorf("failed to delete a non-exist dummy device %s: %v", devName, err)
	}
	dummy, ok := link.(*netlink.Dummy)
	if !ok {
		return fmt.Errorf("expect dummy device, got device type: %s", link.Type())
	}
	return d.LinkDel(dummy)
}

// ListBindAddress list all IP addresses which are bound in a given interface
func (d *DummyDeviceManager) ListBindAddress(devName string) ([]string, error) {
	dev, err := d.LinkByName(devName)
	if err != nil {
		return nil, fmt.Errorf("failed to get interface: %s, err: %v", devName, err)
	}
	addrs, err := d.AddrList(dev, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to list bound address of interface %s, err: %v", devName, err)
	}
	var ips []string
	for _, addr := range addrs {
		ips = append(ips, addr.IP.String())
	}
	return ips, nil
}

// EnsureAddressBind checks if address is bound to the interface, if not, binds it. If the address is already bound, return true.
func (d *DummyDeviceManager) EnsureAddressBind(address, devName string) (bool, error) {
	dev, err := d.LinkByName(devName)
	if err != nil {
		return false, fmt.Errorf("failed to get interface: %s, err: %v", devName, err)
	}
	addr := net.ParseIP(address)
	if addr == nil {
		return false, fmt.Errorf("failed to parse ip address: %s", address)
	}
	if err := d.AddrAdd(dev, &netlink.Addr{IPNet: netlink.NewIPNet(addr)}); err != nil {
		// "EEXIST" will be returned if the address is already bound to device
		if err == unix.EEXIST {
			return true, nil
		}
		return false, fmt.Errorf("failed to bind address %s to interface %s, err: %v", address, devName, err)
	}
	return false, nil
}
