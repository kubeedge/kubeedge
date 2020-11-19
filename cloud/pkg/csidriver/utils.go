/*
Copyright 2019 The KubeEdge Authors.

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

package csidriver

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/kubernetes-csi/csi-lib-utils/protosanitizer"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/common/constants"
)

// Constant defines csi related parameters
const (
	GroupResource            = "resource"
	DefaultNamespace         = "default"
	DefaultReceiveModuleName = "cloudhub"
)

// newNonBlockingGRPCServer creates a new nonblocking server
func newNonBlockingGRPCServer() *nonBlockingGRPCServer {
	return &nonBlockingGRPCServer{}
}

// NonBlocking server
type nonBlockingGRPCServer struct {
	wg     sync.WaitGroup
	server *grpc.Server
}

func (s *nonBlockingGRPCServer) Start(endpoint string, ids csi.IdentityServer, cs csi.ControllerServer, ns csi.NodeServer) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		err := s.serve(endpoint, ids, cs, ns)
		if err != nil {
			panic(err.Error())
		}
	}()
}

func (s *nonBlockingGRPCServer) Wait() {
	s.wg.Wait()
}

func (s *nonBlockingGRPCServer) Stop() {
	s.server.GracefulStop()
}

func (s *nonBlockingGRPCServer) ForceStop() {
	s.server.Stop()
}

func (s *nonBlockingGRPCServer) serve(endpoint string, ids csi.IdentityServer, cs csi.ControllerServer, ns csi.NodeServer) error {
	proto, addr, err := parseEndpoint(endpoint)
	if err != nil {
		klog.Errorf(err.Error())
		return err
	}

	if proto == "unix" {
		addr = "/" + addr
		if err := os.Remove(addr); err != nil && !os.IsNotExist(err) {
			klog.Warningf("failed to remove %s, error: %s", addr, err.Error())
		}
	}

	listener, err := net.Listen(proto, addr)
	if err != nil {
		klog.Errorf("failed to listen: %v", err)
		return err
	}

	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(logGRPC),
	}
	server := grpc.NewServer(opts...)
	s.server = server

	if ids != nil {
		csi.RegisterIdentityServer(server, ids)
	}
	if cs != nil {
		csi.RegisterControllerServer(server, cs)
	}
	if ns != nil {
		csi.RegisterNodeServer(server, ns)
	}
	klog.Infof("listening for connections on address: %#v", listener.Addr())
	return server.Serve(listener)
}

func parseEndpoint(ep string) (string, string, error) {
	if strings.HasPrefix(strings.ToLower(ep), "unix://") || strings.HasPrefix(strings.ToLower(ep), "tcp://") {
		s := strings.SplitN(ep, "://", 2)
		if s[1] != "" {
			return s[0], s[1], nil
		}
	}
	return "", "", fmt.Errorf("invalid endpoint: %v", ep)
}

func logGRPC(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	klog.Infof("grpc call: %s", info.FullMethod)
	klog.Infof("grpc request: %+v", protosanitizer.StripSecrets(req))
	resp, err := handler(ctx, req)
	if err != nil {
		klog.Errorf("grpc error: %v", err)
	} else {
		klog.Infof("grpc response: %+v", protosanitizer.StripSecrets(resp))
	}
	return resp, err
}

// buildResource return a string as "beehive/pkg/core/model".Message.Router.Resource
func buildResource(nodeID, namespace, resourceType, resourceID string) (string, error) {
	if nodeID == "" || namespace == "" || resourceType == "" {
		return "", fmt.Errorf("required parameter are not set (node id, namespace or resource type)")
	}
	resource := fmt.Sprintf("%s%s%s%s%s%s%s", "node", constants.ResourceSep, nodeID, constants.ResourceSep, namespace, constants.ResourceSep, resourceType)
	if resourceID != "" {
		resource += fmt.Sprintf("%s%s", constants.ResourceSep, resourceID)
	}
	return resource, nil
}

// sendToKubeEdge sends messages to KubeEdge
func sendToKubeEdge(context, kubeEdgeEndpoint string) (string, error) {
	us := NewUnixDomainSocket(kubeEdgeEndpoint)
	// connect
	r, err := us.Connect()
	if err != nil {
		return "", err
	}
	// send
	res, err := us.Send(r, context)
	if err != nil {
		return "", err
	}
	return res, nil
}

// extractMessage extracts message
func extractMessage(context string) (*model.Message, error) {
	var msg *model.Message
	if context != "" {
		err := json.Unmarshal([]byte(context), &msg)
		if err != nil {
			return nil, err
		}
	} else {
		err := errors.New("failed to extract message with empty context")
		klog.Errorf("%v", err)
		return nil, err
	}
	return msg, nil
}
