/*
Copyright 2016 The Kubernetes Authors.

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
	"bufio"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeedge/beehive/pkg/common/log"
)

//AddressFamily is uint type var to describe family ips
type AddressFamily uint

const (
	familyIPv4 AddressFamily = 4
	familyIPv6 AddressFamily = 6
)

const (
	ipv4RouteFile = "/proc/net/route"
	ipv6RouteFile = "/proc/net/ipv6_route"
)

//Route is to define object of connection route
type Route struct {
	Interface   string
	Destination net.IP
	Gateway     net.IP
	Family      AddressFamily
}

//RouteFile is object of file of routs
type RouteFile struct {
	name  string
	parse func(input io.Reader) ([]Route, error)
}

var (
	v4File = RouteFile{name: ipv4RouteFile, parse: GetIPv4DefaultRoutes}
	v6File = RouteFile{name: ipv6RouteFile, parse: GetIPv6DefaultRoutes}
)

func (rf RouteFile) extract() ([]Route, error) {
	file, err := os.Open(rf.name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return rf.parse(file)
}

// GetIPv4DefaultRoutes obtains the IPv4 routes, and filters out non-default routes.
func GetIPv4DefaultRoutes(input io.Reader) ([]Route, error) {
	routes := []Route{}
	scanner := bufio.NewReader(input)
	for {
		line, err := scanner.ReadString('\n')
		if err == io.EOF {
			break
		}
		//ignore the headers in the route info
		if strings.HasPrefix(line, "Iface") {
			continue
		}
		fields := strings.Fields(line)
		// Interested in fields:
		//  0 - interface name
		//  1 - destination address
		//  2 - gateway
		dest, err := ParseIP(fields[1], familyIPv4)
		if err != nil {
			return nil, err
		}
		gw, err := ParseIP(fields[2], familyIPv4)
		if err != nil {
			return nil, err
		}
		if !dest.Equal(net.IPv4zero) {
			continue
		}
		routes = append(routes, Route{
			Interface:   fields[0],
			Destination: dest,
			Gateway:     gw,
			Family:      familyIPv4,
		})
	}
	return routes, nil
}

// GetIPv6DefaultRoutes obtains the IPv6 routes, and filters out non-default routes.
func GetIPv6DefaultRoutes(input io.Reader) ([]Route, error) {
	routes := []Route{}
	scanner := bufio.NewReader(input)
	for {
		line, err := scanner.ReadString('\n')
		if err == io.EOF {
			break
		}
		fields := strings.Fields(line)
		// Interested in fields:
		//  0 - destination address
		//  4 - gateway
		//  9 - interface name
		dest, err := ParseIP(fields[0], familyIPv6)
		if err != nil {
			return nil, err
		}
		gw, err := ParseIP(fields[4], familyIPv6)
		if err != nil {
			return nil, err
		}
		if !dest.Equal(net.IPv6zero) {
			continue
		}
		if gw.Equal(net.IPv6zero) {
			continue // loopback
		}
		routes = append(routes, Route{
			Interface:   fields[9],
			Destination: dest,
			Gateway:     gw,
			Family:      familyIPv6,
		})
	}
	return routes, nil
}

// ParseIP takes the hex IP address string from route file and converts it
// to a net.IP address. For IPv4, the value must be converted to big endian.
func ParseIP(str string, family AddressFamily) (net.IP, error) {
	if str == "" {
		return nil, fmt.Errorf("input is nil")
	}
	bytes, err := hex.DecodeString(str)
	if err != nil {
		return nil, err
	}
	if family == familyIPv4 {
		if len(bytes) != net.IPv4len {
			return nil, fmt.Errorf("invalid IPv4 address in route")
		}
		return net.IP([]byte{bytes[3], bytes[2], bytes[1], bytes[0]}), nil
	}
	// Must be IPv6
	if len(bytes) != net.IPv6len {
		return nil, fmt.Errorf("invalid IPv6 address in route")
	}
	return net.IP(bytes), nil
}

//IsInterfaceUp checks the interface is running or not
func IsInterfaceUp(intf *net.Interface) bool {
	if intf == nil {
		return false
	}
	if intf.Flags&net.FlagUp != 0 {
		log.LOGGER.Infof("Interface %v is up", intf.Name)
		return true
	}
	return false
}

//IsLoopbackOrPointToPoint checks interface is loopback or point to point
func IsLoopbackOrPointToPoint(intf *net.Interface) bool {
	return intf.Flags&(net.FlagLoopback|net.FlagPointToPoint) != 0
}

// GetMatchingGlobalIP returns the first valid global unicast address of the given
// 'family' from the list of 'addrs'.
func GetMatchingGlobalIP(addrs []net.Addr, family AddressFamily) (net.IP, error) {
	if len(addrs) > 0 {
		for i := range addrs {
			log.LOGGER.Infof("Checking addr  %s.", addrs[i].String())
			ip, _, err := net.ParseCIDR(addrs[i].String())
			if err != nil {
				return nil, err
			}
			if MemberOf(ip, family) {
				if ip.IsGlobalUnicast() {
					log.LOGGER.Infof("IP found %v", ip)
					return ip, nil
				}
				log.LOGGER.Infof("Non-global unicast address found %v", ip)
			} else {
				log.LOGGER.Infof("%v is not an IPv%d address", ip, int(family))
			}

		}
	}
	return nil, nil
}

// GetIPFromInterface gets the IPs on an interface and returns a global unicast address, if any. The
// interface must be up, the IP must in the family requested, and the IP must be a global unicast address.
func GetIPFromInterface(intfName string, forFamily AddressFamily, nw networkInterfacer) (net.IP, error) {
	intf, err := nw.InterfaceByName(intfName)
	if err != nil {
		return nil, err
	}
	if IsInterfaceUp(intf) {
		addrs, err := nw.Addrs(intf)
		if err != nil {
			return nil, err
		}
		log.LOGGER.Infof("Interface %q has %d addresses :%v.", intfName, len(addrs), addrs)
		matchingIP, err := GetMatchingGlobalIP(addrs, forFamily)
		if err != nil {
			return nil, err
		}
		if matchingIP != nil {
			log.LOGGER.Infof("Found valid IPv%d address %v for interface %q.", int(forFamily), matchingIP, intfName)
			return matchingIP, nil
		}
	}
	return nil, nil
}

// MemberOf tells if the IP is of the desired family. Used for checking interface addresses.
func MemberOf(ip net.IP, family AddressFamily) bool {
	if ip.To4() != nil {
		return family == familyIPv4
	}
	//else
	return family == familyIPv6
}

// ChooseIPFromHostInterfaces looks at all system interfaces, trying to find one that is up that
// has a global unicast address (non-loopback, non-link local, non-point2point), and returns the IP.
// Searches for IPv4 addresses, and then IPv6 addresses.
func ChooseIPFromHostInterfaces(nw networkInterfacer) (net.IP, error) {
	intfs, err := nw.Interfaces()
	if err != nil {
		return nil, err
	}
	if len(intfs) == 0 {
		return nil, fmt.Errorf("no interfaces found on host")
	}
	for _, family := range []AddressFamily{familyIPv4, familyIPv6} {
		log.LOGGER.Infof("Looking for system interface with a global IPv%d address", uint(family))
		for _, intf := range intfs {
			if !IsInterfaceUp(&intf) {
				log.LOGGER.Infof("Skipping: down interface %q", intf.Name)
				continue
			}
			if IsLoopbackOrPointToPoint(&intf) {
				log.LOGGER.Infof("Skipping: LB or P2P interface %q", intf.Name)
				continue
			}
			addrs, err := nw.Addrs(&intf)
			if err != nil {
				return nil, err
			}
			if len(addrs) == 0 {
				log.LOGGER.Infof("Skipping: no addresses on interface %q", intf.Name)
				continue
			}
			for _, addr := range addrs {
				ip, _, err := net.ParseCIDR(addr.String())
				if err != nil {
					return nil, fmt.Errorf("Unable to parse CIDR for interface %q: %s", intf.Name, err)
				}
				if !MemberOf(ip, family) {
					log.LOGGER.Infof("Skipping: no address family match for %q on interface %q.", ip, intf.Name)
					continue
				}
				// TODO: Decide if should open up to allow IPv6 LLAs in future.
				if !ip.IsGlobalUnicast() {
					log.LOGGER.Infof("Skipping: non-global address %q on interface %q.", ip, intf.Name)
					continue
				}
				log.LOGGER.Infof("Found global unicast address %q on interface %q.", ip, intf.Name)
				return ip, nil
			}
		}
	}
	return nil, fmt.Errorf("no acceptable interface with global unicast address found on host")
}

// ChooseHostInterface is a method used fetch an IP for a daemon.
// If there is no routing info file, it will choose a global IP from the system
// interfaces. Otherwise, it will use IPv4 and IPv6 route information to return the
// IP of the interface with a gateway on it (with priority given to IPv4). For a node
// with no internet connection, it returns error.
func ChooseHostInterface() (net.IP, error) {
	var nw networkInterfacer = networkInterface{}
	if _, err := os.Stat(ipv4RouteFile); os.IsNotExist(err) {
		return ChooseIPFromHostInterfaces(nw)
	}
	routes, err := GetAllDefaultRoutes()
	if err != nil {
		return nil, err
	}
	return ChooseHostInterfaceFromRoute(routes, nw)
}

// networkInterfacer defines an interface for several net library functions. Production
// code will forward to net library functions, and unit tests will override the methods
// for testing purposes.
type networkInterfacer interface {
	InterfaceByName(intfName string) (*net.Interface, error)
	Addrs(intf *net.Interface) ([]net.Addr, error)
	Interfaces() ([]net.Interface, error)
}

// networkInterface implements the networkInterfacer interface for production code, just
// wrapping the underlying net library function calls.
type networkInterface struct{}

func (ni networkInterface) InterfaceByName(intfName string) (*net.Interface, error) {
	return net.InterfaceByName(intfName)
}

func (ni networkInterface) Addrs(intf *net.Interface) ([]net.Addr, error) {
	return intf.Addrs()
}

func (ni networkInterface) Interfaces() ([]net.Interface, error) {
	return net.Interfaces()
}

// GetAllDefaultRoutes obtains IPv4 and IPv6 default routes on the node. If unable
// to read the IPv4 routing info file, we return an error. If unable to read the IPv6
// routing info file (which is optional), we'll just use the IPv4 route information.
// Using all the routing info, if no default routes are found, an error is returned.
func GetAllDefaultRoutes() ([]Route, error) {
	routes, err := v4File.extract()
	if err != nil {
		return nil, err
	}
	v6Routes, _ := v6File.extract()
	routes = append(routes, v6Routes...)
	if len(routes) == 0 {
		return nil, fmt.Errorf("No default routes")
	}
	return routes, nil
}

// ChooseHostInterfaceFromRoute cycles through each default route provided, looking for a
// global IP address from the interface for the route. Will first look all each IPv4 route for
// an IPv4 IP, and then will look at each IPv6 route for an IPv6 IP.
func ChooseHostInterfaceFromRoute(routes []Route, nw networkInterfacer) (net.IP, error) {
	for _, family := range []AddressFamily{familyIPv4, familyIPv6} {
		log.LOGGER.Infof("Looking for default routes with IPv%d addresses", uint(family))
		for _, route := range routes {
			if route.Family != family {
				continue
			}
			log.LOGGER.Infof("Default route transits interface %q", route.Interface)
			finalIP, err := GetIPFromInterface(route.Interface, family, nw)
			if err != nil {
				return nil, err
			}
			if finalIP != nil {
				log.LOGGER.Infof("Found active IP %v ", finalIP)
				return finalIP, nil
			}
		}
	}
	log.LOGGER.Infof("No active IP found by looking at default routes")
	return nil, fmt.Errorf("unable to select an IP from default routes")
}

// ValidateNodeIP validates given node IP belongs to the current host
func ValidateNodeIP(nodeIP net.IP) error {
	// Honor IP limitations set in setNodeStatus()
	if nodeIP.To4() == nil && nodeIP.To16() == nil {
		return fmt.Errorf("nodeIP must be a valid IP address")
	}
	if nodeIP.IsLoopback() {
		return fmt.Errorf("nodeIP can't be loopback address")
	}
	if nodeIP.IsMulticast() {
		return fmt.Errorf("nodeIP can't be a multicast address")
	}
	if nodeIP.IsLinkLocalUnicast() {
		return fmt.Errorf("nodeIP can't be a link-local unicast address")
	}
	if nodeIP.IsUnspecified() {
		return fmt.Errorf("nodeIP can't be an all zeros address")
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return err
	}
	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		if ip != nil && ip.Equal(nodeIP) {
			return nil
		}
	}
	return fmt.Errorf("Node IP: %q not found in the host's network interfaces", nodeIP.String())
}

//Command executes command and returns output
func Command(name string, arg []string) (string, error) {
	cmd := exec.Command(name, arg...)
	ret, err := cmd.Output()
	if err != nil {
		log.LOGGER.Errorf("exec command failed: %v", err)
		return string(ret), err
	}
	return strings.Trim(string(ret), "\n"), nil
}

//GetCurPath returns filepath
func GetCurPath() string {
	file, _ := exec.LookPath(os.Args[0])
	path, _ := filepath.Abs(file)
	rst := filepath.Dir(path)
	return rst
}

//ConvertStrToTime converts time string object to inbuilt time object
func ConvertStrToTime(strTime string) (time.Time, error) {
	loc, _ := time.LoadLocation("Local")
	t, err := time.ParseInLocation("2006-01-02T15:04:05", strTime, loc)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}

//ParseTimeErrorCode is constant set to -1 to show error
const ParseTimeErrorCode = -1

//ParseTimestampStr2Int64 returns given string time into int64 type
func ParseTimestampStr2Int64(s string) (int64, error) {
	timeStamp, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return ParseTimeErrorCode, err
	}
	return timeStamp.Unix(), nil
}

//ParseTimestampInt64 returns given int64 type time into matav1 type time
func ParseTimestampInt64(timestamp int64) metav1.Time {
	if timestamp == ParseTimeErrorCode {
		return metav1.Time{}
	}
	return metav1.NewTime(time.Unix(timestamp, 0))
}

// ReadDirNoStat returns a string of files/directories contained
// in dirname without calling lstat on them.
func ReadDirNoStat(dirname string) ([]string, error) {
	if dirname == "" {
		dirname = "."
	}

	f, err := os.Open(dirname)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return f.Readdirnames(-1)
}
