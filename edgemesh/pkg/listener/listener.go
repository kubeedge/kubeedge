package listener

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"github.com/kubeedge/beehive/pkg/core/model"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/cache"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/config"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/protocol"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/protocol/http"
)

type SvcDescription struct {
	sync.RWMutex
	SvcPortsByIP map[string]string // key: fakeIP, value: SvcPorts
	IPBySvc      map[string]string // key: svcName.svcNamespace, value: fakeIP
}

type sockAddr struct {
	family uint16
	data   [14]byte
}

const (
	defaultNetworkPrefix = "9.251."
	maxPoolSize          = 65534

	SoOriginalDst = 80
)

var (
	svcDesc     *SvcDescription
	unused      []string
	indexOfPool uint16
	metaClient  client.CoreInterface
	once        sync.Once
)

func Init() {
	once.Do(func() {
		unused = make([]string, 0)
		svcDesc = &SvcDescription{
			SvcPortsByIP: make(map[string]string),
			IPBySvc:      make(map[string]string),
		}
		// init meta client
		metaClient = client.New()
		// init fakeIP pool
		initPool()
		// recover listener meta from edge db
		recoverFromDB()
	})
}

// getSubNet converts uint16 to "uint8.uint8"
func getSubNet(subNet uint16) string {
	arg1 := uint64(subNet & 0x00ff)
	arg2 := uint64((subNet & 0xff00) >> 8)
	return strconv.FormatUint(arg2, 10) + "." + strconv.FormatUint(arg1, 10)
}

// initPool initializes fakeIP pool with size of 256
func initPool() {
	// avoid 0.0
	indexOfPool = uint16(1)
	for ; indexOfPool <= uint16(255); indexOfPool++ {
		ip := defaultNetworkPrefix + getSubNet(indexOfPool)
		unused = append(unused, ip)
	}
}

// expandPool expands fakeIP pool, each time with size of 256
func expandPool() {
	end := indexOfPool + uint16(255)
	for ; indexOfPool <= end; indexOfPool++ {
		// avoid 255.255
		if indexOfPool > maxPoolSize {
			return
		}
		ip := defaultNetworkPrefix + getSubNet(indexOfPool)
		// if ip is not used, append it to unused
		if svcDesc.getSvcPorts(ip) == "" {
			unused = append(unused, ip)
		}
	}
}

// reserveIp reserves used fakeIP
func reserveIP(ip string) {
	for i, value := range unused {
		if ip == value {
			unused = append(unused[:i], unused[i+1:]...)
			break
		}
	}
}

// recoverFromDB gets fakeIP from edge db and assigns them to services after EdgeMesh starts
func recoverFromDB() {
	svcs, err := metaClient.Services("all").ListAll()
	if err != nil {
		klog.Errorf("[EdgeMesh] list all services from edge db error: %v", err)
		return
	}
	for _, svc := range svcs {
		svcName := svc.Namespace + "." + svc.Name
		value, err := metaClient.Listener().Get(svcName)
		if err != nil {
			klog.Errorf("[EdgeMesh] get listener of svc %s from edge db error: %v", svcName, err)
			continue
		}
		ip, ok := value.([]string)
		if !ok {
			klog.Errorf("[EdgeMesh] value %+v is not a string", value)
			continue
		}
		if len(ip) == 0 {
			svcPorts := getSvcPorts(svc, svcName)
			addServer(svcName, svcPorts)
			klog.Warningf("[EdgeMesh] listener %s from edge db with no ip", svcName)
			continue
		}
		svcPorts := getSvcPorts(svc, svcName)
		reserveIP(ip[0][1 : len(ip[0])-1])
		svcDesc.set(svcName, ip[0][1:len(ip[0])-1], svcPorts)
		klog.Infof("[EdgeMesh] get listener %s from edge db: %s", svcName, ip[0][1:len(ip[0])-1])
	}
}

// Start starts the EdgeMesh listener
func Start() {
	for {
		conn, err := config.Config.Listener.Accept()
		if err != nil {
			klog.Warningf("[EdgeMesh] get tcp conn error: %v", err)
			continue
		}
		ip, port, err := realServerAddress(&conn)
		if err != nil {
			klog.Warningf("[EdgeMesh] get real destination of tcp conn error: %v", err)
			conn.Close()
			continue
		}
		proto, err := newProtocolFromSock(ip, port, conn)
		if err != nil {
			klog.Warningf("[EdgeMesh] get protocol from sock err: %v", err)
			conn.Close()
			continue
		}

		go proto.Process()
	}
}

// newProtocolFromSock returns a protocol.Protocol interface if the ip is in proxy list
func newProtocolFromSock(ip string, port int, conn net.Conn) (proto protocol.Protocol, err error) {
	svcPorts := svcDesc.getSvcPorts(ip)
	protoName, svcName := getProtocol(svcPorts, port)
	if protoName == "" || svcName == "" {
		return nil, fmt.Errorf("protocol name: %s or svcName: %s is invalid", protoName, svcName)
	}

	svcNameSets := strings.Split(svcName, ".")
	if len(svcNameSets) != 2 {
		return nil, fmt.Errorf("invalid length %d after splitting svc name %s", len(svcNameSets), svcName)
	}
	namespace := svcNameSets[0]
	name := svcNameSets[1]

	switch protoName {
	case "http":
		proto = &http.HTTP{
			Conn:         conn,
			SvcName:      name,
			SvcNamespace: namespace,
			Port:         port,
		}
		err = nil
	default:
		proto = nil
		err = fmt.Errorf("protocol: %s is not supported yet", protoName)
	}
	return
}

// getProtocol gets protocol name
func getProtocol(svcPorts string, port int) (string, string) {
	var protoName string
	sub := strings.Split(svcPorts, "|")
	n := len(sub)
	if n < 2 {
		return "", ""
	}
	svcName := sub[n-1]

	pstr := strconv.Itoa(port)
	if pstr == "" {
		return "", ""
	}
	for _, s := range sub {
		if strings.Contains(s, pstr) {
			protoName = strings.Split(s, ",")[0]
			break
		}
	}
	return protoName, svcName
}

// realServerAddress returns an intercepted connection's original destination.
func realServerAddress(conn *net.Conn) (string, int, error) {
	tcpConn, ok := (*conn).(*net.TCPConn)
	if !ok {
		return "", -1, fmt.Errorf("not a TCPConn")
	}

	file, err := tcpConn.File()
	if err != nil {
		return "", -1, err
	}

	// To avoid potential problems from making the socket non-blocking.
	tcpConn.Close()
	*conn, err = net.FileConn(file)
	if err != nil {
		return "", -1, err
	}

	defer file.Close()
	fd := file.Fd()

	var addr sockAddr
	size := uint32(unsafe.Sizeof(addr))
	err = getSockOpt(int(fd), syscall.SOL_IP, SoOriginalDst, uintptr(unsafe.Pointer(&addr)), &size)
	if err != nil {
		return "", -1, err
	}

	var ip net.IP
	switch addr.family {
	case syscall.AF_INET:
		ip = addr.data[2:6]
	default:
		return "", -1, fmt.Errorf("unrecognized address family")
	}

	port := int(addr.data[0])<<8 + int(addr.data[1])
	if err := syscall.SetNonblock(int(fd), true); err != nil {
		return "", -1, nil
	}

	return ip.String(), port, nil
}

func getSockOpt(s int, level int, name int, val uintptr, vallen *uint32) (err error) {
	_, _, e1 := syscall.Syscall6(syscall.SYS_GETSOCKOPT, uintptr(s), uintptr(level), uintptr(name), uintptr(val), uintptr(unsafe.Pointer(vallen)), 0)
	if e1 != 0 {
		err = e1
	}
	return
}

// filterResourceTypeService implements msg filter for "Service" and "ServiceList" resource
func filterResourceTypeService(msg model.Message) []v1.Service {
	svcs := make([]v1.Service, 0)
	content, err := json.Marshal(msg.GetContent())
	if err != nil || len(content) == 0 {
		return svcs
	}
	switch getResourceType(msg.GetResource()) {
	case constants.ResourceTypeService:
		s, err := handleServiceMessage(content)
		if err != nil {
			break
		}
		svcs = append(svcs, *s)
	case constants.ResourceTypeServiceList:
		ss, err := handleServiceListMessage(content)
		if err != nil {
			break
		}
		svcs = append(svcs, ss...)
	default:
		break
	}

	return svcs
}

func getSvcPorts(svc v1.Service, svcName string) string {
	svcPorts := ""
	for _, p := range svc.Spec.Ports {
		pro := strings.Split(p.Name, "-")
		sub := fmt.Sprintf("%s,%d,%d|", pro[0], p.Port, p.TargetPort.IntVal)
		svcPorts = svcPorts + sub
	}
	svcPorts += svcName
	return svcPorts
}

// MsgProcess processes messages from metaManager
func MsgProcess(msg model.Message) {
	// process services
	if svcs := filterResourceTypeService(msg); len(svcs) != 0 {
		klog.Infof("[EdgeMesh] %s services: %d resource: %s", msg.GetOperation(), len(svcs), msg.Router.Resource)
		for i := range svcs {
			svcName := svcs[i].Namespace + "." + svcs[i].Name
			svcPorts := getSvcPorts(svcs[i], svcName)
			switch msg.GetOperation() {
			case "insert":
				cache.GetMeshCache().Add("service"+"."+svcName, &svcs[i])
				klog.Infof("[EdgeMesh] insert svc %s.%s into cache", svcs[i].Namespace, svcs[i].Name)
				addServer(svcName, svcPorts)
			case "update":
				cache.GetMeshCache().Add("service"+"."+svcName, &svcs[i])
				klog.Infof("[EdgeMesh] update svc %s.%s in cache", svcs[i].Namespace, svcs[i].Name)
				updateServer(svcName, svcPorts)
			case "delete":
				cache.GetMeshCache().Remove("service" + "." + svcName)
				klog.Infof("[EdgeMesh] delete svc %s.%s from cache", svcs[i].Namespace, svcs[i].Name)
				delServer(svcName)
			default:
				klog.Warningf("[EdgeMesh] invalid %s operation on services", msg.GetOperation())
			}
		}
		return
	}
	// process pods
	if getResourceType(msg.GetResource()) == model.ResourceTypePodlist {
		klog.Infof("[EdgeMesh] %s podlist, resource: %s", msg.GetOperation(), msg.Router.Resource)
		pods := make([]v1.Pod, 0)
		content, err := json.Marshal(msg.GetContent())
		if err != nil {
			klog.Errorf("[EdgeMesh] marshal podlist msg content err: %v", err)
			return
		}
		pods, err = handlePodListMessage(content)
		if err != nil {
			return
		}
		podListName := getResourceName(msg.GetResource())
		podListNamespace := getResourceNamespace(msg.GetResource())
		switch msg.GetOperation() {
		case "insert", "update":
			cache.GetMeshCache().Add("pods"+"."+podListNamespace+"."+podListName, pods)
			klog.Infof("[EdgeMesh] insert/update pods %s.%s into cache", podListNamespace, podListName)
		case "delete":
			cache.GetMeshCache().Remove("pods" + "." + podListNamespace + "." + podListName)
			klog.Infof("[EdgeMesh] delete pods %s.%s from cache", podListNamespace, podListName)
		default:
			klog.Warningf("[EdgeMesh] invalid %s operation on podlist", msg.GetOperation())
		}
	}
}

// addServer adds a server
func addServer(svcName, svcPorts string) {
	ip := svcDesc.getIP(svcName)
	if ip != "" {
		svcDesc.set(svcName, ip, svcPorts)
		return
	}
	if len(unused) == 0 {
		// try to expand
		expandPool()
		if len(unused) == 0 {
			klog.Warningf("[EdgeMesh] insufficient fake IP !!")
			return
		}
	}
	ip = unused[0]
	unused = unused[1:]

	svcDesc.set(svcName, ip, svcPorts)
	err := metaClient.Listener().Add(svcName, ip)
	if err != nil {
		klog.Errorf("[EdgeMesh] add listener %s to edge db error: %v", svcName, err)
		return
	}
}

// updateServer updates a server
func updateServer(svcName, svcPorts string) {
	ip := svcDesc.getIP(svcName)
	if ip == "" {
		if len(unused) == 0 {
			// try to expand
			expandPool()
			if len(unused) == 0 {
				klog.Warningf("[EdgeMesh] insufficient fake IP !!")
				return
			}
		}
		ip = unused[0]
		unused = unused[1:]
		err := metaClient.Listener().Add(svcName, ip)
		if err != nil {
			klog.Errorf("[EdgeMesh] add listener %s to edge db error: %v", svcName, err)
		}
	}
	svcDesc.set(svcName, ip, svcPorts)
}

// delServer deletes a server
func delServer(svcName string) {
	ip := svcDesc.getIP(svcName)
	if ip == "" {
		return
	}
	svcDesc.del(svcName, ip)
	err := metaClient.Listener().Del(svcName)
	if err != nil {
		klog.Errorf("[EdgeMesh] delete listener from edge db error: %v", err)
	}
	// recycling fakeIP
	unused = append(unused, ip)
}

// handleServiceMessage converts bytes to k8s service meta
func handleServiceMessage(content []byte) (*v1.Service, error) {
	var s v1.Service
	err := json.Unmarshal(content, &s)
	if err != nil {
		klog.Errorf("[EdgeMesh] unmarshal message to k8s service failed, err: %v", err)
		return nil, err
	}
	return &s, nil
}

// handleServiceListMessage converts bytes to k8s service list meta
func handleServiceListMessage(content []byte) ([]v1.Service, error) {
	var ss []v1.Service
	err := json.Unmarshal(content, &ss)
	if err != nil {
		klog.Errorf("[EdgeMesh] unmarshal message to k8s service list failed, err: %v", err)
		return nil, err
	}
	return ss, nil
}

// handlePodListMessage converts bytes to k8s pod list meta
func handlePodListMessage(content []byte) ([]v1.Pod, error) {
	var pp []v1.Pod
	err := json.Unmarshal(content, &pp)
	if err != nil {
		klog.Errorf("[EdgeMesh] unmarshal message to k8s pod list failed, err: %v", err)
		return nil, err
	}
	return pp, nil
}

// getResourceType returns the resource type as a string
func getResourceType(resource string) string {
	str := strings.Split(resource, "/")
	if len(str) == 3 {
		return str[1]
	} else if len(str) == 5 {
		return str[3]
	} else {
		return resource
	}
}

// getResourceName returns the resource name as a string
func getResourceName(resource string) string {
	str := strings.Split(resource, "/")
	if len(str) == 3 {
		return str[2]
	} else if len(str) == 5 {
		return str[4]
	} else {
		return resource
	}
}

// getResourceNamespace returns the resource namespace as a string
func getResourceNamespace(resource string) string {
	str := strings.Split(resource, "/")
	if len(str) == 3 {
		return str[0]
	} else if len(str) == 5 {
		return str[2]
	} else {
		return resource
	}
}

// set is a thread-safe operation to add to map
func (sd *SvcDescription) set(svcName, ip, svcPorts string) {
	sd.Lock()
	defer sd.Unlock()
	sd.IPBySvc[svcName] = ip
	sd.SvcPortsByIP[ip] = svcPorts
}

// del is a thread-safe operation to del from map
func (sd *SvcDescription) del(svcName, ip string) {
	sd.Lock()
	defer sd.Unlock()
	delete(sd.IPBySvc, svcName)
	delete(sd.SvcPortsByIP, ip)
}

// getIP is a thread-safe operation to get from map
func (sd *SvcDescription) getIP(svcName string) string {
	sd.RLock()
	defer sd.RUnlock()
	ip := sd.IPBySvc[svcName]
	return ip
}

// getSvcPorts is a thread-safe operation to get from map
func (sd *SvcDescription) getSvcPorts(ip string) string {
	sd.RLock()
	defer sd.RUnlock()
	svcPorts := sd.SvcPortsByIP[ip]
	return svcPorts
}

// GetServiceServer returns the proxy IP by given service name
func GetServiceServer(svcName string) string {
	ip := svcDesc.getIP(svcName)
	return ip
}
