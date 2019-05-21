package server

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"unsafe"

	"github.com/golang/glog"
)

var (
	inter = "docker0"
	dnsQr = uint16(0x8000)
	oneByteSize = uint16(1)
	twoByteSize = uint16(2)
	ttl = uint32(64)
	fakeIp = []byte{5,5,5,5}
)

const (
	aRecord = 1
	bufSize = 1024
	notImplem = uint16(0x0004)
)

type dnsHeader struct {
	transactionID uint16
	flags uint16
	queNum uint16
	ansNum uint16
	authNum uint16
	additNum uint16
}

type dnsQuestion struct{
	head *dnsHeader
	name []byte
	queByte []byte
	qType uint16
	qClasss uint16
	queNum uint16
}

type dnsAnswer struct{
	name []byte
	qType uint16
	qClass uint16
	ttl uint32
	dataLen uint16
	addr []byte
}

//define the dns Question list type
type  dnsQs []dnsQuestion

//DnsStart is a External interface
func DnsStart(){
	startDnsServer()
}

// getDnsServer returns the specific interface ip of version 4
func getDnsServer() (net.IP, error){
	ifaces , err := net.InterfaceByName(inter)
	if err != nil {
		return  nil ,err
	}

	addrs, _ := ifaces.Addrs()

	for _, addr := range addrs {
		if ip, inet, _ := net.ParseCIDR(addr.String()); len(inet.Mask) == 4 {
			return ip, nil
		}
	}

	return nil, errors.New("the interface" + inter +  "have not config ip of version 4")
}

// startDnsServer start the DNS Server
func startDnsServer(){
	lip, err := getDnsServer()
	if err != nil {
		glog.Errorf("Dns server Start error : %s",err)
	}

	laddr := &net.UDPAddr{
		IP: lip,
		Port: 53,
	}
	dnsConn ,err := net.ListenUDP("udp",laddr)
	defer dnsConn.Close()
	if err != nil {
		glog.Errorf("Dns server Start error : %s",err)
	}

	for {
		req := make([]byte,bufSize)
		n, from, err := dnsConn.ReadFromUDP(req)
		if err != nil || n <= 0 {
			glog.Infof("Dns Server get an read mistake : %s", err)
			continue
		}

		que,err := parseDnsQuery(req[:n])
		if err != nil {
			continue
		}

		rsp := make([]byte,0)
		rsp , _ = recordHandler(que, req[0:n])

		dnsConn.WriteTo(rsp,from)
	}
}

//recordHandler returns the Answer for the dns question
func recordHandler(que []dnsQuestion, req []byte) (rsp []byte, err error){
	var exist bool
	for _ , q := range que {
		domainName := string(q.name)
		exist, err = lookupFromMetaManager(domainName)
		if err != nil {
			rsp = nil
			return
		}
		if !exist {
			break
		}
	}

	if !exist {
		//if this service don't belongs to this cluster
		rsp = getfromRealDNS(req)
		err = nil
		return
	}
	//gen
	pre,err  := modifyRspPrefix(que)
	rsp = append(rsp, pre...)
	for _ , q := range que{
		//create a deceptive rep
		dnsAns := &dnsAnswer{
			name: q.name,
			qType: q.qType,
			qClass: q.qClasss,
			ttl: ttl,
			dataLen: uint16(len(fakeIp)),
			addr: fakeIp,
		}
		ans := dnsAns.getAnswer()
		rsp = append(rsp,ans...)
	}

	return rsp,nil
}

//parseDnsQuery returns question of the dns request
func parseDnsQuery(req [] byte) ( que []dnsQuestion, err error) {
	head := &dnsHeader{}
	head.getHeader(req)
	if !head.isAQurey() {
		return nil, errors.New("Igenore")
	}

	question := make(dnsQs, head.queNum)

	offset := uint16(unsafe.Sizeof(dnsHeader{}))
	question.getQuestion(req, offset, head)

	que = question
	err = nil
	return
}

//isAQuery judge if the dns pkg is a Qurey process
func (h *dnsHeader)isAQurey()bool {
	if h.flags & dnsQr != dnsQr {
		return true
	}
	return false
}

//getHeader get dns pkg head
func (h *dnsHeader)getHeader(req []byte) {
	h.transactionID = binary.BigEndian.Uint16(req[0:2])
	h.flags = binary.BigEndian.Uint16(req[2:4])
	h.queNum = binary.BigEndian.Uint16(req[4:6])
	h.ansNum = binary.BigEndian.Uint16(req[6:8])
	h.authNum = binary.BigEndian.Uint16(req[8:10])
	h.additNum = binary.BigEndian.Uint16(req[10:12])
}

//getQuestion get dns questions
func (q dnsQs)getQuestion(req []byte, offset uint16, head *dnsHeader)  {
	ost := offset
	qNum := uint16(len(q))

	for i:=uint16(0); i< qNum ; i++ {
		tmp := ost
		ost = q[i].getQName(req,ost)
		q[i].qType = binary.BigEndian.Uint16(req[ost: ost + twoByteSize])
		ost += twoByteSize
		q[i].qClasss = binary.BigEndian.Uint16(req[ost: ost + twoByteSize])
		ost += twoByteSize
		q[i].head = head
		q[i].queByte = req[tmp: ost]
	}
}

//getAnswer Generate Answer for the dns question
func (d *dnsAnswer)getAnswer() (ans []byte) {
	ans = make([]byte,0)

	if d.qType == aRecord {
		ans = append(ans,0xc0)
		ans = append(ans,0x0c)

		tmp16 := make([]byte,2)
		tmp32 := make([]byte,4)

		binary.BigEndian.PutUint16(tmp16,d.qType)
		ans = append(ans,tmp16...)
		binary.BigEndian.PutUint16(tmp16, d.qClass)
		ans = append(ans,tmp16...)
		binary.BigEndian.PutUint32(tmp32, d.ttl)
		ans = append(ans,tmp32...)
		binary.BigEndian.PutUint16(tmp16, d.dataLen)
		ans = append(ans,tmp16...)
		ans = append(ans,d.addr...)
	}

	return ans
}

// getQName get dns question domain name
func (q *dnsQuestion)getQName(req []byte, offset uint16) (uint16) {
	ost := offset

	for {
		qbyte := uint16(req[ost])

		if qbyte == 0x00 {
			q.name = q.name[: uint16(len(q.name))- oneByteSize]
			return ost + oneByteSize
		}
		ost += oneByteSize
		q.name = append(q.name, req[ost: ost+qbyte]...)
		q.name = append(q.name,0x2e)
		ost += qbyte
	}
}

// lookupFromMetaManager implement confirm the service exists
// for earlier version returns true by default
func lookupFromMetaManager(name string) (exist bool ,err error){
	return true, nil
}

// getfromRealDNS returns the dns response from the real DNS server
func getfromRealDNS(req []byte) []byte {
	ips := parseNameServer()

	rsp := make([]byte,0)
	laddr := &net.UDPAddr{
		IP: net.IPv4zero,
		Port: 0,
	}
	for _, ip := range ips  { // get from real
		raddr := &net.UDPAddr{
			IP: ip,
			Port: 53,
		}
		conn, err := net.DialUDP("udp",laddr,raddr)
		defer conn.Close()
		if err != nil {
			continue
		}

		_, err = conn.Write(req)
		if err != nil {
			continue
		}

		var n int
		buf := make([]byte,bufSize)
		n, err = conn.Read(buf)
		if err != nil {
			continue
		}

		if n >0 {
			rsp = append(rsp,buf[: n]...)
		}
	}
	return rsp
}

// parseNameServer gets the nameserver from the resolv.conf
func parseNameServer()([]net.IP){
	file, err := os.Open("/etc/resolv.conf")
	if err != nil {
		glog.Errorf("error opening /etc/resolv.conf : %s", err)
	}
	defer file.Close()

	scan := bufio.NewScanner(file)
	scan.Split(bufio.ScanLines)

	ip := make([]net.IP,0)

	for scan.Scan() { //get name server
		serverString := scan.Text()
		fmt.Println(serverString)
		if strings.Contains(serverString,"nameserver") {
			tmpString := strings.Replace(serverString, "nameserver", "",1)
			nameserver := strings.TrimSpace(tmpString)
			sip := net.ParseIP(nameserver)
			if sip != nil {
				ip = append(ip, sip)
			}
		}
	}
	return ip
}

// modifyRspPrefix use req' head generate a rsp head
func modifyRspPrefix(que []dnsQuestion) (pre []byte, err error) {
	ansNum := len(que)
	if ansNum == 0{
		return
	}
	//use head in que. All the same
	rspHead := que[0].head
	rspHead.converQueryRsp(true)
	if que[0].qType == aRecord {
		rspHead.setAnswerNum(uint16(ansNum))
	}else{
		rspHead.setAnswerNum(0)
	}

	rspHead.setRspRcode(que)
	pre = rspHead.getByteFromDnsHeader()

	for _ , q := range que {
		pre = append(pre, q.queByte...)
	}

	err = nil
	return
}

// converQueryRsp conversion the dns head to a response for one query
func (h *dnsHeader)converQueryRsp(isRsp bool){
	if isRsp {
		h.flags |= dnsQr
	}else {
		h.flags |= dnsQr
	}
}

// set the Answer num for dns head
func (h *dnsHeader)setAnswerNum(num uint16){
	h.ansNum = num
}

// set the dns response return code
func (h *dnsHeader)setRspRcode(que dnsQs){
	for _ , q := range que {
		if q.qType != aRecord {
			h.flags &= (^notImplem)
			h.flags |= notImplem
		}
	}
}

//getByteFromDnsHeader implement from dnsHeader struct to []byte
func (h *dnsHeader)getByteFromDnsHeader() (rspHead []byte){
	rspHead = make([]byte, unsafe.Sizeof(*h))

	idxTran := unsafe.Sizeof(h.transactionID)
	idxflag := unsafe.Sizeof(h.flags) + idxTran
	idxque := unsafe.Sizeof(h.ansNum) + idxflag
	idxans := unsafe.Sizeof(h.ansNum) + idxque
	idxaut := unsafe.Sizeof(h.authNum) + idxans
	idxadd := unsafe.Sizeof(h.additNum) + idxaut

	binary.BigEndian.PutUint16(rspHead[: idxTran], h.transactionID)
	binary.BigEndian.PutUint16(rspHead[idxTran: idxflag], h.flags)
	binary.BigEndian.PutUint16(rspHead[idxflag: idxque], h.queNum)
	binary.BigEndian.PutUint16(rspHead[idxque: idxans], h.ansNum)
	binary.BigEndian.PutUint16(rspHead[idxans: idxaut], h.authNum)
	binary.BigEndian.PutUint16(rspHead[idxaut: idxadd], h.additNum)
	return
}
