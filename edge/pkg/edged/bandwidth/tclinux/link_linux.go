package tclinux

import (
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/safchain/ethtool"
	"github.com/vishvananda/netlink"

	"github.com/kubeedge/kubeedge/edge/pkg/edged/bandwidth/tclinux/ns"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/bandwidth/tclinux/sysctl"
)

const (
	entropySize         = 4
	expectedDeviceCount = 2
	maxHandleValue      = 0xffff
)

var ErrLinkNotFound = errors.New("link not found")

// makeVethPair is called from within the container's network namespace
func makeVethPair(name, peer string, mtu int, mac string, hostNS ns.NetNS) (netlink.Link, error) {
	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name: name,
			MTU:  mtu,
		},
		PeerName:      peer,
		PeerNamespace: netlink.NsFd(int(hostNS.Fd())),
	}
	if mac != "" {
		m, err := net.ParseMAC(mac)
		if err != nil {
			return nil, err
		}
		veth.LinkAttrs.HardwareAddr = m
	}
	if err := netlink.LinkAdd(veth); err != nil {
		return nil, err
	}
	// Re-fetch the container link to get its creation-time parameters, e.g. index and mac
	veth2, err := netlink.LinkByName(name)
	if err != nil {
		err = netlink.LinkDel(veth) // try and clean up the link if possible.
		if err != nil {
			return nil, err
		}
		return nil, err
	}

	return veth2, nil
}

func peerExists(name string) bool {
	if _, err := netlink.LinkByName(name); err != nil {
		return false
	}
	return true
}

func makeVeth(name, vethPeerName string, mtu int, mac string, hostNS ns.NetNS) (peerName string,
	veth netlink.Link, err error,
) {
	for i := 0; i < 10; i++ {
		if vethPeerName != "" {
			peerName = vethPeerName
		} else {
			peerName, err = RandomVethName()
			if err != nil {
				return peerName, veth, err
			}
		}

		veth, err = makeVethPair(name, peerName, mtu, mac, hostNS)
		switch {
		case err == nil:
			return peerName, veth, nil

		case os.IsExist(err):
			if peerExists(peerName) && vethPeerName == "" {
				continue
			}
			return peerName, veth, fmt.Errorf("container veth name provided (%v) already exists", name)

		default:
			return peerName, veth, fmt.Errorf("failed to make veth pair: %v", err)
		}
	}

	// should really never be hit
	return "", nil, fmt.Errorf("failed to find a unique veth name")
}

// RandomVethName returns string "veth" with random prefix (hashed from entropy)
func RandomVethName() (string, error) {
	entropy := make([]byte, entropySize)
	_, err := rand.Read(entropy)
	if err != nil {
		return "", fmt.Errorf("failed to generate random veth name: %v", err)
	}

	// NetworkManager (recent versions) will ignore veth devices that start with "veth"
	return fmt.Sprintf("veth%x", entropy), nil
}

func RenameLink(curName, newName string) error {
	link, err := netlink.LinkByName(curName)
	if err == nil {
		err = netlink.LinkSetName(link, newName)
	}
	return err
}

func ifaceFromNetlinkLink(l netlink.Link) net.Interface {
	a := l.Attrs()
	return net.Interface{
		Index:        a.Index,
		MTU:          a.MTU,
		Name:         a.Name,
		HardwareAddr: a.HardwareAddr,
		Flags:        a.Flags,
	}
}

// SetupVethWithName sets up a pair of virtual ethernet devices.
// Call SetupVethWithName from inside the container netns.  It will create both veth
// devices and move the host-side veth into the provided hostNS namespace.
// hostVethName: If hostVethName is not specified, the host-side veth name will use a random string.
// On success, SetupVethWithName returns (hostVeth, containerVeth, nil)
func SetupVethWithName(contVethName, hostVethName string, mtu int, contVethMac string, hostNS ns.NetNS) (contIface,
	hostIface net.Interface, err error,
) {
	hostVethName, contVeth, err := makeVeth(contVethName, hostVethName, mtu, contVethMac, hostNS)
	if err != nil {
		return net.Interface{}, net.Interface{}, err
	}

	var hostVeth netlink.Link
	err = hostNS.Do(func(_ ns.NetNS) error {
		hostVeth, err = netlink.LinkByName(hostVethName)
		if err != nil {
			return fmt.Errorf("failed to lookup %q in %q: %v", hostVethName, hostNS.Path(), err)
		}

		if err = netlink.LinkSetUp(hostVeth); err != nil {
			return fmt.Errorf("failed to set %q up: %v", hostVethName, err)
		}

		// we want to own the routes for this interface
		_, err = sysctl.Sysctl(fmt.Sprintf("net/ipv6/conf/%s/accept_ra", hostVethName), "0")
		if err != nil {
			return fmt.Errorf("sysctl error, hostVethName %s: %v", hostVethName, err)
		}
		return nil
	})
	if err != nil {
		return net.Interface{}, net.Interface{}, err
	}
	return ifaceFromNetlinkLink(hostVeth), ifaceFromNetlinkLink(contVeth), nil
}

// SetupVeth sets up a pair of virtual ethernet devices.
// Call SetupVeth from inside the container netns.  It will create both veth
// devices and move the host-side veth into the provided hostNS namespace.
// On success, SetupVeth returns (hostVeth, containerVeth, nil)
func SetupVeth(contVethName string, mtu int, contVethMac string, hostNS ns.NetNS) (contIface, hostIface net.Interface,
	err error,
) {
	return SetupVethWithName(contVethName, "", mtu, contVethMac, hostNS)
}

// DelLinkByName removes an interface link.
func DelLinkByName(ifName string) error {
	iface, err := netlink.LinkByName(ifName)
	if err != nil {
		if _, ok := err.(netlink.LinkNotFoundError); ok {
			return ErrLinkNotFound
		}
		return fmt.Errorf("failed to lookup %q: %v", ifName, err)
	}

	if err = netlink.LinkDel(iface); err != nil {
		return fmt.Errorf("failed to delete %q: %v", ifName, err)
	}

	return nil
}

// DelLinkByNameAddr remove an interface and returns its addresses
func DelLinkByNameAddr(ifName string) ([]*net.IPNet, error) {
	iface, err := netlink.LinkByName(ifName)
	if err != nil {
		if _, ok := err.(netlink.LinkNotFoundError); ok {
			return nil, ErrLinkNotFound
		}
		return nil, fmt.Errorf("failed to lookup %q: %v", ifName, err)
	}

	addrs, err := netlink.AddrList(iface, netlink.FAMILY_ALL)
	if err != nil {
		return nil, fmt.Errorf("failed to get IP addresses for %q: %v", ifName, err)
	}

	if err = netlink.LinkDel(iface); err != nil {
		return nil, fmt.Errorf("failed to delete %q: %v", ifName, err)
	}

	out := []*net.IPNet{}
	for _, addr := range addrs {
		if addr.IP.IsGlobalUnicast() {
			out = append(out, addr.IPNet)
		}
	}

	return out, nil
}

// GetVethPeerIfindex returns the veth link object, the peer ifindex of the
// veth, or an error. This peer ifindex will only be valid in the peer's
// network namespace.
func GetVethPeerIfindex(ifName string) (netlink.Link, int, error) {
	link, err := netlink.LinkByName(ifName)
	if err != nil {
		return nil, -1, fmt.Errorf("could not look up %q: %v", ifName, err)
	}
	if _, ok := link.(*netlink.Veth); !ok {
		return nil, -1, fmt.Errorf("interface %q was not a veth interface", ifName)
	}

	// veth supports IFLA_LINK (what vishvananda/netlink calls ParentIndex)
	// on 4.1 and higher kernels
	peerIndex := link.Attrs().ParentIndex
	if peerIndex <= 0 {
		// Fall back to ethtool for 4.0 and earlier kernels
		e, err := ethtool.NewEthtool()
		if err != nil {
			return nil, -1, fmt.Errorf("failed to initialize ethtool: %v", err)
		}
		defer e.Close()

		stats, err := e.Stats(link.Attrs().Name)
		if err != nil {
			return nil, -1, fmt.Errorf("failed to request ethtool stats: %v", err)
		}
		n, ok := stats["peer_ifindex"]
		if !ok {
			return nil, -1, fmt.Errorf("failed to find 'peer_ifindex' in ethtool stats")
		}
		if n > 32767 || n == 0 {
			return nil, -1, fmt.Errorf("invalid 'peer_ifindex' %d", n)
		}
		peerIndex = int(n)
	}

	return link, peerIndex, nil
}
