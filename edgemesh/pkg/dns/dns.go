package dns

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
	"unsafe"

	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/common"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/config"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/listener"
)

type Event int

var (
	// default docker0
	ifi = "docker0"
	// QR: 0 represents query, 1 represents response
	dnsQR       = uint16(0x8000)
	oneByteSize = uint16(1)
	twoByteSize = uint16(2)
	ttl         = uint32(64)
)

const (
	// 1 for ipv4
	aRecord           = 1
	bufSize           = 1024
	errNotImplemented = uint16(0x0004)
	errRefused        = uint16(0x0005)
	eventNothing      = Event(0)
	eventUpstream     = Event(1)
	eventNxDomain     = Event(2)
)

type dnsHeader struct {
	id      uint16
	flags   uint16
	qdCount uint16
	anCount uint16
	nsCount uint16
	arCount uint16
}

type dnsQuestion struct {
	from    *net.UDPAddr
	head    *dnsHeader
	name    []byte
	queByte []byte
	qType   uint16
	qClass  uint16
	queNum  uint16
	event   Event
}

type dnsAnswer struct {
	name    []byte
	qType   uint16
	qClass  uint16
	ttl     uint32
	dataLen uint16
	addr    []byte
}

// metaClient is a query client
var metaClient client.CoreInterface

// dnsConn saves DNS protocol
var dnsConn *net.UDPConn

// Start is for external call
func Start() {
	startDNS()
}

// startDNS starts edgemesh dns server
func startDNS() {
	// init meta client
	metaClient = client.New()
	// get dns listen ip
	lip, err := common.GetInterfaceIP(ifi)
	if err != nil {
		klog.Errorf("[EdgeMesh] get dns listen ip err: %v", err)
		return
	}

	laddr := &net.UDPAddr{
		IP:   lip,
		Port: 53,
	}
	udpConn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		klog.Errorf("[EdgeMesh] dns server listen on %v error: %v", laddr, err)
		return
	}
	defer udpConn.Close()
	dnsConn = udpConn
	for {
		req := make([]byte, bufSize)
		n, from, err := dnsConn.ReadFromUDP(req)
		if err != nil || n <= 0 {
			klog.Errorf("[EdgeMesh] dns server read from udp error: %v", err)
			continue
		}

		que, err := parseDNSQuery(req[:n])
		if err != nil {
			continue
		}

		que.from = from

		rsp := make([]byte, 0)
		rsp, err = recordHandle(que, req[:n])
		if err != nil {
			klog.Warningf("[EdgeMesh] failed to resolve dns: %v", err)
			continue
		}
		if _, err = dnsConn.WriteTo(rsp, from); err != nil {
			klog.Warningf("[EdgeMesh] failed to write: %v", err)
		}
	}
}

// recordHandle returns the answer for the dns question
func recordHandle(que *dnsQuestion, req []byte) (rsp []byte, err error) {
	var exist bool
	var ip string
	// qType should be 1 for ipv4
	if que.name != nil && que.qType == aRecord {
		domainName := string(que.name)
		exist, ip = lookupFromMetaManager(domainName)
	}

	if !exist || que.event == eventUpstream {
		// if this service doesn't belongs to this cluster
		go getFromRealDNS(req, que.from)
		return rsp, fmt.Errorf("get from real dns")
	}

	address := net.ParseIP(ip).To4()
	if address == nil {
		que.event = eventNxDomain
	}
	// gen
	pre := modifyRspPrefix(que)
	rsp = append(rsp, pre...)
	if que.event != eventNothing {
		return rsp, nil
	}
	// create a deceptive resp, if no error
	dnsAns := &dnsAnswer{
		name:    que.name,
		qType:   que.qType,
		qClass:  que.qClass,
		ttl:     ttl,
		dataLen: uint16(len(address)),
		addr:    address,
	}
	ans := dnsAns.getAnswer()
	rsp = append(rsp, ans...)

	return rsp, nil
}

// parseDNSQuery converts bytes to *dnsQuestion
func parseDNSQuery(req []byte) (que *dnsQuestion, err error) {
	head := &dnsHeader{}
	head.getHeader(req)
	if !head.isAQuery() {
		return nil, errors.New("not a dns query, ignore")
	}
	que = &dnsQuestion{
		event: eventNothing,
	}
	// Generally, when the recursive DNS server requests upward, it may
	// initiate a resolution request for multiple aliases/domain names
	// at once, Edge DNS does not need to process a message that carries
	// multiple questions at a time.
	if head.qdCount != 1 {
		que.event = eventUpstream
		return
	}

	offset := uint16(unsafe.Sizeof(dnsHeader{}))
	// DNS NS <ROOT> operation
	if req[offset] == 0x0 {
		que.event = eventUpstream
		return
	}
	que.getQuestion(req, offset, head)
	err = nil
	return
}

// isAQuery judges if the dns pkg is a query
func (h *dnsHeader) isAQuery() bool {
	return h.flags&dnsQR != dnsQR
}

// getHeader gets dns pkg head
func (h *dnsHeader) getHeader(req []byte) {
	h.id = binary.BigEndian.Uint16(req[0:2])
	h.flags = binary.BigEndian.Uint16(req[2:4])
	h.qdCount = binary.BigEndian.Uint16(req[4:6])
	h.anCount = binary.BigEndian.Uint16(req[6:8])
	h.nsCount = binary.BigEndian.Uint16(req[8:10])
	h.arCount = binary.BigEndian.Uint16(req[10:12])
}

// getQuestion gets a dns question
func (q *dnsQuestion) getQuestion(req []byte, offset uint16, head *dnsHeader) {
	ost := offset
	tmp := ost
	ost = q.getQName(req, ost)
	q.qType = binary.BigEndian.Uint16(req[ost : ost+twoByteSize])
	ost += twoByteSize
	q.qClass = binary.BigEndian.Uint16(req[ost : ost+twoByteSize])
	ost += twoByteSize
	q.head = head
	q.queByte = req[tmp:ost]
}

// getAnswer generates answer for the dns question
func (da *dnsAnswer) getAnswer() (answer []byte) {
	answer = make([]byte, 0)

	if da.qType == aRecord {
		answer = append(answer, 0xc0)
		answer = append(answer, 0x0c)

		tmp16 := make([]byte, 2)
		tmp32 := make([]byte, 4)

		binary.BigEndian.PutUint16(tmp16, da.qType)
		answer = append(answer, tmp16...)
		binary.BigEndian.PutUint16(tmp16, da.qClass)
		answer = append(answer, tmp16...)
		binary.BigEndian.PutUint32(tmp32, da.ttl)
		answer = append(answer, tmp32...)
		binary.BigEndian.PutUint16(tmp16, da.dataLen)
		answer = append(answer, tmp16...)
		answer = append(answer, da.addr...)
	}

	return answer
}

// getQName gets dns question qName
func (q *dnsQuestion) getQName(req []byte, offset uint16) uint16 {
	ost := offset

	for {
		// one byte to suggest length
		qbyte := uint16(req[ost])

		// qName ends with 0x00, and 0x00 should not be included
		if qbyte == 0x00 {
			q.name = q.name[:uint16(len(q.name))-oneByteSize]
			return ost + oneByteSize
		}
		// step forward one more byte and get the real stuff
		ost += oneByteSize
		q.name = append(q.name, req[ost:ost+qbyte]...)
		// add "." symbol
		q.name = append(q.name, 0x2e)
		ost += qbyte
	}
}

// lookupFromMetaManager confirms if the service exists
func lookupFromMetaManager(serviceURL string) (exist bool, ip string) {
	name, namespace := common.SplitServiceKey(serviceURL)
	s, _ := metaClient.Services(namespace).Get(name)
	if s != nil {
		svcName := namespace + "." + name
		ip := listener.GetServiceServer(svcName)
		klog.Infof("[EdgeMesh] dns server parse %s ip %s", serviceURL, ip)
		return true, ip
	}
	klog.Errorf("[EdgeMesh] service %s is not found in this cluster", serviceURL)
	return false, ""
}

// getFromRealDNS returns a dns response from real dns servers
func getFromRealDNS(req []byte, from *net.UDPAddr) {
	rsp := make([]byte, 0)
	ips, err := parseNameServer()
	if err != nil {
		klog.Errorf("[EdgeMesh] parse nameserver err: %v", err)
		return
	}

	laddr := &net.UDPAddr{
		IP:   net.IPv4zero,
		Port: 0,
	}

	// get from real dns servers
	for _, ip := range ips {
		raddr := &net.UDPAddr{
			IP:   ip,
			Port: 53,
		}
		conn, err := net.DialUDP("udp", laddr, raddr)
		if err != nil {
			continue
		}
		defer conn.Close()
		_, err = conn.Write(req)
		if err != nil {
			continue
		}
		if err = conn.SetReadDeadline(time.Now().Add(time.Minute)); err != nil {
			continue
		}
		var n int
		buf := make([]byte, bufSize)
		n, err = conn.Read(buf)
		if err != nil {
			continue
		}

		if n > 0 {
			rsp = append(rsp, buf[:n]...)
			if _, err = dnsConn.WriteToUDP(rsp, from); err != nil {
				klog.Errorf("[EdgeMesh] failed to wirte to udp, err: %v", err)
				continue
			}
			break
		}
	}
}

// parseNameServer gets all real nameservers from the resolv.conf
func parseNameServer() ([]net.IP, error) {
	file, err := os.Open("/etc/resolv.conf")
	if err != nil {
		return nil, fmt.Errorf("error opening /etc/resolv.conf: %v", err)
	}
	defer file.Close()

	scan := bufio.NewScanner(file)
	scan.Split(bufio.ScanLines)

	ip := make([]net.IP, 0)

	for scan.Scan() {
		serverString := scan.Text()
		if strings.Contains(serverString, "nameserver") {
			tmpString := strings.Replace(serverString, "nameserver", "", 1)
			nameserver := strings.TrimSpace(tmpString)
			sip := net.ParseIP(nameserver)
			if sip != nil && !sip.Equal(config.Config.ListenIP) {
				ip = append(ip, sip)
			}
		}
	}
	if len(ip) == 0 {
		return nil, fmt.Errorf("there is no nameserver in /etc/resolv.conf")
	}
	return ip, nil
}

// modifyRspPrefix generates a dns response head
func modifyRspPrefix(que *dnsQuestion) (pre []byte) {
	if que == nil {
		return
	}
	// use head in que
	rspHead := que.head
	rspHead.convertQueryRsp(true)
	if que.qType == aRecord {
		rspHead.setAnswerNum(1)
	} else {
		rspHead.setAnswerNum(0)
	}

	rspHead.setRspRCode(que)
	pre = rspHead.getByteFromDNSHeader()

	pre = append(pre, que.queByte...)
	return
}

// convertQueryRsp converts a dns question head to a response head
func (h *dnsHeader) convertQueryRsp(isRsp bool) {
	if isRsp {
		h.flags |= dnsQR
	} else {
		h.flags |= dnsQR
	}
}

// setAnswerNum sets the answer num for dns head
func (h *dnsHeader) setAnswerNum(num uint16) {
	h.anCount = num
}

// setRspRCode sets dns response return code
func (h *dnsHeader) setRspRCode(que *dnsQuestion) {
	if que.qType != aRecord {
		h.flags &= (^errNotImplemented)
		h.flags |= errNotImplemented
	} else if que.event == eventNxDomain {
		h.flags &= (^errRefused)
		h.flags |= errRefused
	}
}

// getByteFromDNSHeader converts dnsHeader to bytes
func (h *dnsHeader) getByteFromDNSHeader() (rspHead []byte) {
	rspHead = make([]byte, unsafe.Sizeof(*h))

	idxTransactionID := unsafe.Sizeof(h.id)
	idxFlags := unsafe.Sizeof(h.flags) + idxTransactionID
	idxQDCount := unsafe.Sizeof(h.anCount) + idxFlags
	idxANCount := unsafe.Sizeof(h.anCount) + idxQDCount
	idxNSCount := unsafe.Sizeof(h.nsCount) + idxANCount
	idxARCount := unsafe.Sizeof(h.arCount) + idxNSCount

	binary.BigEndian.PutUint16(rspHead[:idxTransactionID], h.id)
	binary.BigEndian.PutUint16(rspHead[idxTransactionID:idxFlags], h.flags)
	binary.BigEndian.PutUint16(rspHead[idxFlags:idxQDCount], h.qdCount)
	binary.BigEndian.PutUint16(rspHead[idxQDCount:idxANCount], h.anCount)
	binary.BigEndian.PutUint16(rspHead[idxANCount:idxNSCount], h.nsCount)
	binary.BigEndian.PutUint16(rspHead[idxNSCount:idxARCount], h.arCount)
	return
}
