package dns

import (
	"fmt"
	"net"

	"k8s.io/klog/v2"
	"github.com/miekg/dns"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/common"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/listener"
)

var (
	// default interface
	// TODO: use custom or unfixed interface
	ifi            = "docker0"
	metaClient     client.CoreInterface
	resolvFile     = "/etc/resolv.conf"
	dnsDefaultPort = "53"
)

type handler struct{}

func (h *handler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	msg := dns.Msg{}
	msg.SetReply(r)
	switch r.Question[0].Qtype {
	case dns.TypeA:
		msg.Authoritative = true
		domain := msg.Question[0].Name
		address, ok := lookupFromMetaManager(domain)
		if ok {
			msg.Answer = append(msg.Answer, &dns.A{
				Hdr: dns.RR_Header{Name: domain, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60},
				A:   net.ParseIP(address),
			})
		} else {
			return
		}
	}
	if err := w.WriteMsg(&msg); err != nil {
		klog.Errorf("[EdgeMesh] failed to send dns response, err: %v", err)
	}
}

func lookupFromMetaManager(domain string) (string, bool) {
	name, namespace := common.SplitServiceKey(domain)
	svc := namespace + "." + name
	if _, err := metaClient.Listener().Get(svc); err != nil {
		klog.V(2).Infof("[EdgeMesh] request %s is not found in this cluster", svc)
		return "", false
	}

	s, _ := metaClient.Services(namespace).Get(name)
	if s != nil {
		svcName := namespace + "." + name
		ip := listener.GetServiceServer(svcName)
		klog.V(2).Infof("[EdgeMesh] dns server parse %s ip %s", domain, ip)
		return ip, true
	}
	klog.V(2).Infof("[EdgeMesh] service %s is not found in this cluster", domain)

	return "", false
}

func Start() {
	startDNS()
}

func startDNS() {
	lip, err := common.GetInterfaceIP(ifi)
	if err != nil {
		klog.Errorf("[EdgeMesh] get dns listen ip err: %v", err)
		return
	}
	addr := fmt.Sprintf("%v:53", lip)
	srv := &dns.Server{Addr: addr, Net: "udp"}
	srv.Handler = &handler{}
	metaClient = client.New()
	if err := srv.ListenAndServe(); err != nil {
		klog.Fatalf("Failed to set udp listener, err: %v", err)
		return
	}
}
