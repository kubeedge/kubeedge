package proxy

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/proxy/poll"
	vdev "github.com/kubeedge/kubeedge/edgemesh/pkg/proxy/virtualdevice"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

type conntrack struct {
	lconn   net.Conn
	rconn   net.Conn
	connNum uint32
}

type listener struct {
	ln          net.Listener
	serviceName string
	fd          int32
}

type serviceTable struct {
	ip         string
	ports      []int32
	targetPort []int32
	lns        []net.Listener
}

type addrTable struct {
	sync.Map
}

const (
	tcpBufSize               = 8192
	defaultNetworkPrefix     = "9.251."
	defaultIpPoolSize        = 20
	defaultTcpClientTimeout  = time.Second * 2
	defaultTcpReconnectTimes = 3
	firstPort                = 0
)

var (
	epoll         *poll.Epoll
	addrByService *addrTable
	unused        []string
	serve         sync.Map
	metaClient    client.CoreInterface
	ipPoolSize    uint16
)

// Init: init the proxy. create virtual device and assign ips, etc..
func Init() {
	go func() {
		unused = make([]string, 0)
		addrByService = &addrTable{}
		c := context.GetContext(context.MsgCtxTypeChannel)
		metaClient = client.New(c)
		//create virtual network device
		for {
			err := vdev.CreateDevice()
			if err == nil {
				break
			}
			klog.Warningf("[L4 Proxy] create Device is failed : %s", err)
			//there may have some exception need to be fixed on OS
			time.Sleep(2 * time.Second)
		}
		//configure vir ip
		ipPoolSize = 0
		expandIpPool()
		//open epoll
		ep, err := poll.CreatePoll(pollCallback)
		if err != nil {
			vdev.DestroyDevice()
			klog.Errorf("[L4 Proxy] epoll is open failed : %s", err)
			return
		}
		epoll = ep
		go epoll.Loop()
		klog.Infof("[L4 Proxy] proxy is running now")
	}()
}

// expandIpPool expand ip pool if virtual ip is not enough
func expandIpPool() {
	// make sure the subnet can't be "255.255"
	if ipPoolSize > 0xfffe {
		return
	}
	for idx := ipPoolSize + 1; idx <= ipPoolSize+defaultIpPoolSize; idx++ {
		ip := defaultNetworkPrefix + getSubNet(idx)
		if err := vdev.AddIP(ip); err != nil {
			vdev.DestroyDevice()
			klog.Errorf("[L4 Proxy] Add ip is failed : %s please checkout the env", err)
			return
		}
		unused = append(unused, ip)
	}
	ipPoolSize = defaultIpPoolSize
}

// pollCallback process the connection from client
func pollCallback(fd int32) {
	value, ok := serve.Load(fd)
	if !ok {
		return
	}
	listen, ok := value.(listener)
	if !ok {
		return
	}
	ln := listen.ln
	serviceName := listen.serviceName

	conn, err := ln.Accept()
	if err != nil {
		return
	}
	getAndSetSocket(conn, false)
	go startTcpServer(conn, serviceName)
}

// startTcpServer implement L4 proxy to the real server
func startTcpServer(conn net.Conn, svcName string) {
	portString := strings.Split(conn.LocalAddr().String(), ":")
	// portString is a standard form, such as "172.17.0.1:8080"
	localPort, _ := strconv.ParseInt(portString[1], 10, 32)
	addr, err := doLoadBalance(svcName, localPort)
	if err != nil {
		klog.Warningf("[L4 Proxy] %s call svc : %s encountered an error: %s", conn.RemoteAddr().String(), svcName, err)
		conn.Close()
		return
	}
	var proxyClient net.Conn
	for retry := 0; retry < defaultTcpReconnectTimes; retry++ {
		proxyClient, err = net.DialTimeout("tcp", addr.String(), defaultTcpClientTimeout)
		if err == nil {
			break
		}
	}
	// Error when connecting to server ,maybe timeout or any other error
	if err != nil {
		klog.Warningf("[L4 Proxy] %s call svc : %s to %s encountered an error: %s", conn.RemoteAddr().String(), svcName, addr.String(), err)
		conn.Close()
		return
	}

	ctk := &conntrack{
		lconn: conn,
		rconn: proxyClient,
	}
	klog.Infof("[L4 Proxy] start a proxy server : %s,%s", svcName, addr.String())
	go func() {
		ctk.processServerProxy()
	}()
	go func() {
		ctk.processClientProxy()
	}()
}

// doLoadBalance implement the loadbalance function
func doLoadBalance(svcName string, lport int64) (net.Addr, error) {
	svc := strings.Split(svcName, ".")
	namespace, name := svc[0], svc[1]
	pods, err := metaClient.Services(namespace).GetPods(name)
	if err != nil {
		klog.Errorf("[L4 Proxy] get svc error : %s", err)
	}
	// checkout the status of pods
	runPods := make([]v1.Pod, 0, len(pods))
	for i := 0; i < len(pods); i++ {
		if pods[i].Status.Phase == v1.PodRunning {
			runPods = append(runPods, pods[i])
		}
	}
	// support random LB for the early version
	rand.Seed(time.Now().UnixNano())
	idx := rand.Uint32() % uint32(len(runPods))

	hostIP := runPods[idx].Status.HostIP
	st := addrByService.getAddrTable(svcName)

	index := 0
	for i, p := range st.ports {
		if p == int32(lport) {
			index = i
			break
		}
	}
	// kubeedge edgemesh support bridge net ,so, use hostport to access
	tp := st.targetPort[index]
	targetPort := int32(0)
	for _, value := range runPods[idx].Spec.Containers {
		for _, v := range value.Ports {
			if v.ContainerPort == tp {
				targetPort = v.HostPort
				break
			}
		}
	}

	return &net.TCPAddr{
		IP:   net.ParseIP(hostIP),
		Port: int(targetPort),
	}, nil
}

// GetServiceServer returns the proxy IP by given service name
func GetServiceServer(svcName string) string {
	st := addrByService.getAddrTable(svcName)
	if st == nil {
		klog.Warningf("[L4 Proxy] Serivce %s is not ready for Proxy.", svcName)
		return "Proxy-abnormal"
	}
	return st.ip
}

//getSubNet Implement uint16 convert to "uint8.uint8"
func getSubNet(subNet uint16) string {
	arg1 := uint64(subNet & 0x00ff)
	arg2 := uint64((subNet & 0xff00) >> 8)
	return strconv.FormatUint(arg2, 10) + "." + strconv.FormatUint(arg1, 10)
}

//getAndSetSocket get file description and set socket blocking
func getAndSetSocket(ln interface{}, nonblock bool) int {
	fd := int(-1)
	switch network := ln.(type) {
	case *net.TCPListener:
		file, err := network.File()
		if err != nil {
			klog.Infof("[L4 Proxy] get fd %s", err)
		} else {
			fd = int(file.Fd())
		}
	case *net.TCPConn:
		file, err := network.File()
		if err != nil {
			klog.Infof("[L4 Proxy] get fd %s", err)
		} else {
			fd = int(file.Fd())
		}
	default:
		klog.Infof("[L4 Proxy] unknow conn")
	}

	err := syscall.SetNonblock(fd, nonblock)
	if err != nil {
		klog.Errorf("[L4 Proxy] Set Nonblock : %s", err)
	}

	return fd
}

// addAddrTable is a thread-safe operation to add to map
func (at *addrTable) addAddrTable(key string, value *serviceTable) {
	at.Store(key, value)
}

// addAddrTable is a thread-safe operation to del from map
func (at *addrTable) delAddrTable(key string) {
	at.Delete(key)
}

// addAddrTable is a thread-safe operation to get from map
func (at *addrTable) getAddrTable(key string) *serviceTable {
	value, ok := at.Load(key)
	if !ok {
		return nil
	}
	st, ok := value.(*serviceTable)
	if !ok {
		return nil
	}
	return st
}

// filterResourceType implement filter. Proxy cares "Service" and "ServiceList" type
func filterResourceType(msg model.Message) []v1.Service {
	svcs := make([]v1.Service, 0)
	content, ok := msg.Content.([]byte)
	if !ok {
		return svcs
	}
	switch getReourceType(msg.GetResource()) {
	case constants.ResourceTypeService:
		s, err := handleServiceMessage(content)
		if err != nil {
			break
		}
		svcs = append(svcs, *s)
	case constants.ResourceTypeServiceList:
		ss, err := handleServiceMessageList(content)
		if err != nil {
			break
		}
		svcs = append(svcs, ss...)
	default:
		klog.Infof("[L4 Proxy] process other resource: %s", msg.Router.Resource)
	}

	return svcs
}

// MsgProcess process from metaManager and start a proxy server
func MsgProcess(msg model.Message) {
	svcs := filterResourceType(msg)
	if len(svcs) == 0 {
		return
	}
	klog.Infof("[L4 Proxy] proxy process svcs : %d resource: %s\n", len(svcs), msg.Router.Resource)
	for _, svc := range svcs {
		svcName := svc.Namespace + "." + svc.Name
		if !IsL4Proxy(&svc) {
			// when server protocol update to http
			delServer(svcName)
			continue
		}

		klog.Infof("[L4 Proxy] proxy process svc : %s,%s", msg.GetOperation(), svcName)
		port := make([]int32, 0)
		targetPort := make([]int32, 0)
		for _, p := range svc.Spec.Ports {
			// this version will support TCP only
			if p.Protocol == "TCP" {
				port = append(port, p.Port)
				// this version will not support string type
				targetPort = append(targetPort, p.TargetPort.IntVal)
			}
		}
		if len(port) == 0 || len(targetPort) == 0 {
			continue
		}
		switch msg.GetOperation() {
		case "insert":
			addServer(svcName, port)
		case "delete":
			delServer(svcName)
		case "update":
			updateServer(svcName, port)
		default:
			klog.Infof("[L4 proxy] Unknown operation")
		}
		st := addrByService.getAddrTable(svcName)
		if st != nil {
			st.targetPort = targetPort
		}
	}
}

// addServer : add the proxy server
func addServer(svcName string, ports []int32) {
	var ip string
	st := addrByService.getAddrTable(svcName)
	if st != nil {
		if len(ports) == 0 && len(st.ports) == 0 {
			unused = append(unused, st.ip)
			addrByService.delAddrTable(svcName)
			return
		}
		ip = st.ip
	} else {
		if len(ports) == 0 {
			return
		}
		if len(unused) == 0 {
			expandIpPool()
		}
		ip = unused[0]
		unused = unused[1:]
	}

	lns := make([]net.Listener, 0)
	for _, port := range ports {
		addr := ip + ":" + strconv.FormatUint(uint64(port), 10)
		klog.Infof("[L4 Proxy] Start listen %s,%d for proxy", addr, port)
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			klog.Errorf("[L4 Proxy] %s", err)
			continue
		}
		lns = append(lns, ln)
		server := listener{
			ln:          ln,
			serviceName: svcName,
			fd:          int32(getAndSetSocket(ln, true)),
		}
		serve.Store(server.fd, server)
		epoll.EpollCtrlAdd(server.fd)
	}
	if st != nil {
		st.lns = append(st.lns, lns...)
		st.ports = append(st.ports, ports...)
	} else {
		st = &serviceTable{
			ip:    ip,
			lns:   lns,
			ports: ports,
		}
		addrByService.addAddrTable(svcName, st)
	}
}

// delServer implement delete the proxy server
func delServer(svcName string) {
	st := addrByService.getAddrTable(svcName)

	if st == nil {
		return
	}
	unused = append(unused, st.ip)
	for _, ln := range st.lns {
		fd := getAndSetSocket(ln, true)
		epoll.EpollCtrlDel(fd)
		ln.Close()
	}
	addrByService.delAddrTable(svcName)
}

// updateServer implement update the proxy server
func updateServer(svcName string, ports []int32) error {
	st := addrByService.getAddrTable(svcName)
	if st == nil { // if not exist
		addServer(svcName, ports)
	} else {
		oldports := make([]int32, len(st.ports))
		copy(oldports, st.ports)
		for idx, oldport := range oldports {
			update := true
			for k, newport := range ports {
				if oldport == newport {
					ports = append(ports[:k], ports[k+1:]...)
					update = false
					break
				}
			}
			if update {
				fd := getAndSetSocket(st.lns[idx], true)
				epoll.EpollCtrlDel(fd)
				st.lns[idx].Close()
				st.lns = append(st.lns[:idx], st.lns[idx+1:]...)
				st.ports = append(st.ports[:idx], st.ports[idx+1:]...)
			}
		}
		addServer(svcName, ports)
	}
	return nil
}

//handleMessageFromMetaManager convert []byte to k8s Service struct
func handleServiceMessage(content []byte) (*v1.Service, error) {
	var s v1.Service
	err := json.Unmarshal(content, &s)
	if err != nil {
		return nil, fmt.Errorf("[L4 Proxy] unmarshal message to Service failed, err: %v", err)
	}
	return &s, nil
}

//handleMessageFromMetaManager convert []byte to k8s Service struct
func handleServiceMessageList(content []byte) ([]v1.Service, error) {
	var s []v1.Service
	err := json.Unmarshal(content, &s)
	if err != nil {
		return nil, fmt.Errorf("[L4 Proxy] unmarshal message to Service failed, err: %v", err)
	}
	return s, nil
}

// getReourceType returns the reourceType as a string
func getReourceType(reource string) string {
	str := strings.Split(reource, "/")
	if len(str) == 3 {
		return str[1]
	} else if len(str) == 5 {
		return str[3]
	} else {
		return reource
	}
}

// processServerProxy process up link traffic
func (c *conntrack) processClientProxy() {
	buf := make([]byte, tcpBufSize)
	for {
		n, err := c.lconn.Read(buf)
		//service caller closess the connection
		if n == 0 {
			c.lconn.Close()
			c.rconn.Close()
			break
		}
		if err != nil {
			c.lconn.Close()
			c.rconn.Close()
			break
		}
		_, rerr := c.rconn.Write(buf[:n])
		if rerr != nil {
			c.lconn.Close()
			c.rconn.Close()
			break
		}
	}
}

// processServerProxy process down link traffic
func (c *conntrack) processServerProxy() {
	buf := make([]byte, tcpBufSize)
	for {
		n, err := c.rconn.Read(buf)
		if n == 0 {
			c.rconn.Close()
			c.lconn.Close()
			break
		}
		if err != nil {
			c.rconn.Close()
			c.lconn.Close()
			break
		}
		_, rerr := c.lconn.Write(buf[:n])
		if rerr != nil {
			c.rconn.Close()
			c.lconn.Close()
			break
		}
	}
}

//isL4Proxy Determine whether to use L4 proxy
func IsL4Proxy(svc *v1.Service) bool {
	if len(svc.Spec.Ports) == 0 {
		return false
	}
	// In the defination of k8s-Service, we can use Service.Spec.Ports.Name to
	// indicate whether the Service enables L4 proxy mode. According to our
	// current L7 mode only support the http protocol. Other 7-layer protocols
	// are automatically degraded to tcp until supported
	port := svc.Spec.Ports[firstPort]
	switch port.Name {
	case "websocket", "grpc", "https", "tcp":
		return true
	case "http", "udp":
		return false
	default:
		return true
	}
}
