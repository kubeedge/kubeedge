package virtualdevice

import (
	"fmt"
	"github.com/Microsoft/hcsshim/hcn"
	"k8s.io/klog"
	"net"
)

const (
	DeviceNameDefault = "edge0"
)

func HcnGenerateNATNetwork(subnet *hcn.Subnet) *hcn.HostComputeNetwork {
	ipams := []hcn.Ipam{}
	if subnet != nil {
		ipam := hcn.Ipam{
			Type: "Static",
			Subnets: []hcn.Subnet{
				*subnet,
			},
		}
		ipams = append(ipams, ipam)
	}
	network := &hcn.HostComputeNetwork{
		Type: "NAT",
		Name: DeviceNameDefault,
		MacPool: hcn.MacPool{
			Ranges: []hcn.MacRange{
				{
					StartMacAddress: "00-15-5D-52-C0-00",
					EndMacAddress:   "00-15-5D-52-CF-FF",
				},
			},
		},
		Flags: hcn.EnableNonPersistent,
		Ipams: ipams,
		SchemaVersion: hcn.SchemaVersion{
			Major: 2,
			Minor: 0,
		},
	}
	return network
}
func CreateSubnet(AddressPrefix string, NextHop string, DestPrefix string) *hcn.Subnet {
	return &hcn.Subnet{
		IpAddressPrefix: AddressPrefix,
		Routes: []hcn.Route{
			{
				NextHop:           NextHop,
				DestinationPrefix: DestPrefix,
			},
		},
	}
}
func CreateDevice() error {
	//if device is exist,delete and create a new one
	//make sure the latest configuration
	DestroyDevice()
	_, err := HcnGenerateNATNetwork(CreateSubnet("9.251.0.0/24", "9.251.0.1", "0.0.0.0/0")).Create()
	return err
}

// AddIP Bind ip to the default device
func AddIP(address string) error {
	dev, err := hcn.GetNetworkByName(DeviceNameDefault)
	if err != nil {
		return fmt.Errorf("[L4 Proxy] Device %s is not exist!!", DeviceNameDefault)
	}

	addr := net.ParseIP(address)
	if addr == nil {
		return fmt.Errorf("[L4 Proxy] AddIP address is invalid ")
	}
	ic := hcn.IpConfig{IpAddress: address}
	var flags hcn.EndpointFlags

	endPoint := &hcn.HostComputeEndpoint{
		Name:             address,
		Flags:            flags,
		IpConfigurations: []hcn.IpConfig{ic},
		SchemaVersion: hcn.SchemaVersion{
			Major: 2,
			Minor: 0,
		},
	}
	_, err = dev.CreateEndpoint(endPoint)
	if err != nil {
		klog.Warningf("Error Assigning IP %s", address)
		return err
	}
	return nil
}

// DestroyDevice implement release the device
func DestroyDevice() error {
	dev, err := hcn.GetNetworkByName(DeviceNameDefault)
	if err != nil {
		return fmt.Errorf("[L4 Proxy] Device %s is not exist!!", DeviceNameDefault)
	}
	err = dev.Delete()
	return err
}

func init() {

}
