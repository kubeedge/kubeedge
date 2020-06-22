---
title: EdgeMesh Optimization
authors:
    - "@daixiang0"
approvers:
creation-date: 2020-06-22
last-updated: 2020-08-18
status:
---

# EdgeMesh Optimization

## Background

Now EdgeMesh works as a DNS and proxy server with some limitations:

- Depend on docker0 network interface, cannot work for other CRIs
- Only support IPV4 DNS resolver
- Use fake ip replaced with cluster ip
- Only support HTTP protocol proxy when apps use hostPort
- Hack go-chassis which is a microservice framework indeed

## Motivation

To support more use case, EdgeMesh need to optimize.

### Goals

- Alpha(Release 1.4)
  * Support multiple CRIs like docker, containerd, cri-o, etc.
  * Support IPV4 and IPV6 DNS resolver
  * Support L4 and L7 proxy
  * Support multiple types like containerPort, hostPort, etc.

- Beta(Release 1.5)
  * Abort go-chassis
  * Support cluster IP proxy
  * Support multiple protocols like HTTPS, gRPC, etc.

### Non-goals

* Cross pubilc network subnet commuinication

## Proposal

Refactor EdgeMesh to achieve goals.

### Advantage

For DNS server, now we code by ourselves, need to spend a lot of time coding and doing performance bottleneck and robustness test.
By swiching to use open source DNS library, we do not need to worry about those parts any more and save lots of time.

For proxy server, now we hack go-chassis framework, still need to test stability. Meanwhile, it is hard to upgrade go-chassis version
since we hack it. We would spend much time learning how we hack it with old version and trying how to hack it with new version,
go-chassis community may provide little help about hack.

On the other hand, use iptables to proxy is a mature and stable solution, no need to consider upgradation. Also we can learn from
kube-proxy.

### Risks and Mitigations

For DNS server, We introduce new dependence: "miekg/dns", need to consider upgrading if any high risk bug is fixed.

For proxy server, we almost reuse kube-poxy with some changes. When upgrade kubernetes version, kube-proxy related logic may be changed
since it is not independent.

## High Level Design

### DNS server

* Alpha(Release 1.5)

    To remove docker dependence, create a virtual network interface and bind it as a DNS server.

    It provides Edge Level DNS resolver support.

    When a request comes:
    - if not in the cluster: reply nothing
    - if in the cluster: reply a fake IP

* Beta(Release 1.6)

    When a request comes:
    - if not in the cluster: reply nothing
    - if in the cluster:
      - for headless services:
       - with selectors, reply endpint IP
       - without selectors, reply endpoint IP list
      - for other services, reply cluster IP

### proxy server

Here we do not consider how to commuinicate between node and container or between container and container, CNIs would implement them.

* Alpha(Release 1.5)

    Proxy server store fake IP and kubernetes service name using KV map, then use go-chassis(set load balance) to request all endpoint IP list and return the result.

* Beta(Release 1.6)

    Based on cluster IP, proxy server does load balancing proxy to a running endpoint.

    We use IPVS as default due to iptables known issue when the number of services is too many. Almost edge device use Ubuntu/Raspberrypi, whose kernel version is high enough to support IPVS.
    For others which use low kernel version, we use iptables.

## Low Level Design

### DNS server

* Alpha(Release 1.5)

    Call "miekg/dns" library and implement ServeDNS interface:

    ```go
    func (h *handler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
      msg := dns.Msg{}
      msg.SetReply(r)
      switch r.Question[0].Qtype {
        // IPV4
        case dns.TypeA:
          domain := msg.Question[0].Name
          fakeIP, exist := lookupFromMetaManager(domain)
          if exist {
            // write response with fakeIP
          } else {
            return
          }
        // IPV6
        case dns.TypeAAAA:
          // same logic as IPV4
        }
      ...
    }

    func startDNS(){
      // lip is the ip which EdgeMesh listens on
      addr := fmt.Sprintf("%v:53", lip)
      srv := &dns.Server{Addr: addr, Net: "udp"}
      srv.Handler = &handler{}
      metaClient = client.New()
      if err := srv.ListenAndServe(); err != nil {
        ...
      }
    }

    func lookupFromMetaManager(domain string) (string, bool) {
      // search KV map by service and return fakeIP if exist
    }
    ```

* Beta(Release 1.6)

    Based on implement in Alpha version, update lookupFromMetaManager function:

    ```go
    func lookupFromMetaManager(requestService string) ([]string, bool) {
      name, namespace := common.SplitServiceKey(domain)
      // search meta data database by service name
      service, err := metaClient.Services(namespace).Get(name)
      ...
      // check whether service is headless or not
      clusterIP, ok := getClusterIP(service)
      if ok {
        return []string{clusterIP}, true
      } else {
        endpoints, err := metaClient.Endpoints(namespace).Get(name)
        var ipList []string
        for _, e := range endpoint.Subsets {
          for _, addr := range e.Addresses {
            ipList = apeend(ipList, addr.IP)
          }
        }
        return ipList, true
      }
      ...
    }

    ```

### proxy server

* Alpha(Release 1.5)

    In Alpha version, proxy server should support more types rather than only containerPort:

    ```go
    func (esd *EdgeServiceDiscovery) FindMicroServiceInstances(consumerID, microServiceName string, tags utiltags.Tags) ([]*registry.MicroServiceInstance, error) {
    ...
    -	for _, p := range pods {
    -		if p.Status.Phase == v1.PodRunning {
    +	for _, e := range endpoint.Subsets {
    +		for _, addr := range e.Addresses {
                microServiceInstances = append(microServiceInstances, &registry.MicroServiceInstance{
                    InstanceID:   "",
                    ServiceID:    name + "." + namespace,
                    HostName:     "",
    -				EndpointsMap: map[string]string{"rest": fmt.Sprintf("%s:%d", p.Status.HostIP, hostPort)},
    +				EndpointsMap: map[string]string{"rest": fmt.Sprintf("%s:%d", addr.IP, targetPort)},
                })
            }
        }

    ```

    Old version search running pods then get `hostIP` and `hostPort` to request, which mean it only support `hostPort` type.
    In Alpha version, change the logic to search all endpoints then get endpoint ip and port to request.

* Beta(Release 1.6)

    In Beta version, proxy server aborts go-chassis and switch to iptables.

    Kube-proxy use informer to trigger corresponding event,
    take "add service" as an example(all codes under [kubernetes 1.17.1](https://github.com/kubernetes/kubernetes/tree/v1.17.1) which we depend on):
    ```go
    func (s *ProxyServer) Run() error {
        ...
        // Create configs (i.e. Watches for Services and Endpoints or EndpointSlices)
        // Note: RegisterHandler() calls need to happen before creation of Sources because sources
        // only notify on changes, and the initial update (on process start) may be lost if no handlers
        // are registered yet.
        serviceConfig := config.NewServiceConfig(informerFactory.Core().V1().Services(), s.ConfigSyncPeriod)
        serviceConfig.RegisterEventHandler(s.Proxier)
        go serviceConfig.Run(wait.NeverStop)
        ...
    }

    func NewServiceConfig(serviceInformer coreinformers.ServiceInformer, resyncPeriod time.Duration) *ServiceConfig {
    	result := &ServiceConfig{
    		listerSynced: serviceInformer.Informer().HasSynced,
    	}

    	serviceInformer.Informer().AddEventHandlerWithResyncPeriod(
    		cache.ResourceEventHandlerFuncs{
    			AddFunc:    result.handleAddService,
    			UpdateFunc: result.handleUpdateService,
    			DeleteFunc: result.handleDeleteService,
    		},
    		resyncPeriod,
    	)

    	return result
    }

    func (c *ServiceConfig) handleAddService(obj interface{}) {
    	service, ok := obj.(*v1.Service)
    	if !ok {
    		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
    		return
    	}
    	for i := range c.eventHandlers {
    		klog.V(4).Info("Calling handler.OnServiceAdd")
    		c.eventHandlers[i].OnServiceAdd(service)
    	}
    }
    ```
    if reuse it as much as possible, when EdgeMesh start, listen all messages from metaManager:
    ```go
    func (proxier *Proxier) MsgProcess(msg model.Message) {
      switch getResourceaTpye(msg) {
      case "service":
          proxier.handServices(msg)
      case "endpoint":
          proxier.handEndpoints(msg)
      }
      ...
    }
    ```

    Then `handServices` and `handEndpoints` handle message and trigger event:
    ```go
    func (proxier *Proxier) handServices(msg model.Message) {
        service := getServices(msg)
        switch msg.GetOperation() {
        case "insert":
            proxier.OnServiceAdd(service)
        case "update":
            proxier.OnServiceUpdate(service)
        case "delete":
            proxier.OnServiceDelete(service)
        default:
            return
        }
    }

    func handEndpoints(msg model.Message) {
        service := getServices(msg)
        switch msg.GetOperation() {
        case "insert":
            proxier.OnEndpointsAdd(service)
        case "update":
            proxier.OnEndpointsUpdate(service)
        case "delete":
            proxier.OnEndpointsDelete(service)
        default:
            return
        }
    }
    ```
