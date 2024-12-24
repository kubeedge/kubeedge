package tclinux

import (
	"fmt"
	"net"
	"syscall"

	"github.com/vishvananda/netlink"
	"k8s.io/klog/v2"
)

const (
	latencyInMillis  = 25
	conversionFactor = 1000.0
	bytesNum         = 8
)

func CreateIfb(ifbDeviceName string, mtu int) error {
	// 判断ifb设备是否已存在
	ifblink, err := netlink.LinkByName(ifbDeviceName)
	if err == nil && ifblink != nil {
		// ifb已存在
		klog.Infof("ifb netlink interface existed,name:%s", ifbDeviceName)
		return nil
	}
	err = netlink.LinkAdd(&netlink.Ifb{
		LinkAttrs: netlink.LinkAttrs{
			Name:  ifbDeviceName,
			Flags: net.FlagUp,
			MTU:   mtu,
		},
	})
	if err != nil {
		return fmt.Errorf("adding link: %s", err)
	}

	return nil
}

func TeardownIfb(deviceName string) error {
	_, err := DelLinkByNameAddr(deviceName)
	if err != nil && err == ErrLinkNotFound {
		return nil
	}
	return err
}

func CreateIngressQdisc(rateInBits, burstInBits uint64, hostDeviceName string) error {
	hostDevice, err := netlink.LinkByName(hostDeviceName)
	if err != nil {
		return fmt.Errorf("get host device: %s", err)
	}
	return createTBF(rateInBits, burstInBits, hostDevice)
}

func CreateEgressQdisc(rateInBits, burstInBits uint64, hostDeviceName, ifbDeviceName string) error {
	ifbDevice, err := netlink.LinkByName(ifbDeviceName)
	if err != nil {
		return fmt.Errorf("get ifb device: %s", err)
	}
	hostDevice, err := netlink.LinkByName(hostDeviceName)
	if err != nil {
		return fmt.Errorf("get host device: %s", err)
	}

	// add qdisc ingress on host device
	ingress := &netlink.Ingress{
		QdiscAttrs: netlink.QdiscAttrs{
			LinkIndex: hostDevice.Attrs().Index,
			Handle:    netlink.MakeHandle(maxHandleValue, 0), // ffff:
			Parent:    netlink.HANDLE_INGRESS,
		},
	}
	ingressFlag := false
	qdiscList, err := netlink.QdiscList(hostDevice)
	if err != nil {
		klog.Warningf("`%s` list qdisc failed,error:%v", hostDeviceName, err)
	}
	for _, qdisc := range qdiscList {
		if qdisc.Attrs().Parent == ingress.Parent && qdisc.Attrs().Handle == ingress.Handle {
			klog.Infof("netlink `%s`  ingress existed ", hostDevice.Attrs().Name)
			ingressFlag = true
		}
	}
	if !ingressFlag {
		err = netlink.QdiscAdd(ingress)
		if err != nil {
			return fmt.Errorf("create ingress qdisc: %s", err)
		}
	}
	// add filter on host device to mirror traffic to ifb device
	filter := &netlink.U32{
		FilterAttrs: netlink.FilterAttrs{
			LinkIndex: hostDevice.Attrs().Index,
			Parent:    ingress.QdiscAttrs.Handle,
			Priority:  1,
			Protocol:  syscall.ETH_P_ALL,
		},
		ClassId:    netlink.MakeHandle(1, 1),
		RedirIndex: ifbDevice.Attrs().Index,
		Actions: []netlink.Action{
			&netlink.MirredAction{
				ActionAttrs:  netlink.ActionAttrs{},
				MirredAction: netlink.TCA_EGRESS_REDIR,
				Ifindex:      ifbDevice.Attrs().Index,
			},
		},
	}
	err = netlink.FilterReplace(filter)
	if err != nil {
		return fmt.Errorf("add filter: %s", err)
	}

	// throttle traffic on ifb device
	err = createTBF(rateInBits, burstInBits, ifbDevice)
	if err != nil {
		return fmt.Errorf("create ifb qdisc: %s", err)
	}
	return nil
}

func createTBF(rateInBits, burstInBits uint64, link netlink.Link) error {
	linkIndex := link.Attrs().Index
	linkName := link.Attrs().Name
	// Equivalent to
	// tc qdisc add dev link root tbf
	//		rate netConf.BandwidthLimits.Rate
	//		burst netConf.BandwidthLimits.Burst
	if rateInBits <= 0 {
		return fmt.Errorf("invalid rate: %d", rateInBits)
	}
	if burstInBits <= 0 {
		return fmt.Errorf("invalid burst: %d", burstInBits)
	}
	rateInBytes := rateInBits / bytesNum
	burstInBytes := burstInBits / bytesNum
	bufferInBytes := buffer(rateInBytes, uint32(burstInBytes))
	latency := latencyInUsec(latencyInMillis)
	limitInBytes := limit(rateInBytes, latency, uint32(burstInBytes))

	qdisc := &netlink.Tbf{
		QdiscAttrs: netlink.QdiscAttrs{
			LinkIndex: linkIndex,
			Handle:    netlink.MakeHandle(1, 0),
			Parent:    netlink.HANDLE_ROOT,
		},
		Limit:  limitInBytes,
		Rate:   rateInBytes,
		Buffer: bufferInBytes,
	}
	//  check network card queue qdisc exists
	list, err := netlink.QdiscList(link)
	if err != nil {
		klog.Warningf("netlink %s list qdisc failed,error:%v", linkName, err)
	}

	if len(list) == 0 {
		klog.Infof("netlink %s add qdisc", linkName)
		err = netlink.QdiscAdd(qdisc)
	} else {
		klog.Infof("netlink %s replace qdisc", linkName)
		err = netlink.QdiscReplace(qdisc)
	}
	if err != nil {
		return fmt.Errorf("create qdisc: %s", err)
	}
	return nil
}

func time2Tick(time uint32) uint32 {
	return uint32(float64(time) * float64(netlink.TickInUsec()))
}

func buffer(rate uint64, burst uint32) uint32 {
	return time2Tick(uint32(float64(burst) * float64(netlink.TIME_UNITS_PER_SEC) / float64(rate)))
}

func limit(rate uint64, latency float64, buffer uint32) uint32 {
	return uint32(float64(rate)*latency/float64(netlink.TIME_UNITS_PER_SEC)) + buffer
}

func latencyInUsec(latencyInMillis float64) float64 {
	return float64(netlink.TIME_UNITS_PER_SEC) * (latencyInMillis / conversionFactor)
}
