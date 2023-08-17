package grpcserver

import (
	"fmt"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"k8s.io/klog/v2"

	"github.com/kubeedge/mapper-generator/pkg/global"
	dmiapi "github.com/kubeedge/mapper-generator/pkg/temp"
)

type Config struct {
	SockPath string `json:"sock_path"`
	Protocol string `json:"protocol"`
}

type Server struct {
	dmiapi.UnimplementedDeviceMapperServiceServer
	lis      net.Listener
	cfg      Config
	devPanel global.DevPanel
}

func NewServer(cfg Config, devPanel global.DevPanel) Server {
	s := Server{cfg: cfg}
	s.devPanel = devPanel
	return s
}

func (s *Server) Start() error {
	klog.Infof("uds socket path: %s", s.cfg.SockPath)
	err := initSock(s.cfg.SockPath)
	if err != nil {
		klog.Fatalf("failed to remove uds socket with err: %v", err)
		return err
	}

	s.lis, err = net.Listen("unix", s.cfg.SockPath)
	if err != nil {
		klog.Fatalf("failed to remove uds socket with err: %v", err)
		return err
	}
	grpcServer := grpc.NewServer()
	dmiapi.RegisterDeviceMapperServiceServer(grpcServer, s)
	reflection.Register(grpcServer)
	klog.Info("start grpc server")
	return grpcServer.Serve(s.lis)
}

func (s *Server) Stop() {
	err := s.lis.Close()
	if err != nil {
		return
	}
	err = os.Remove(s.cfg.SockPath)
	if err != nil {
		return
	}
}

func initSock(sockPath string) error {
	klog.Infof("init uds socket: %s", sockPath)
	_, err := os.Stat(sockPath)
	if err == nil {
		err = os.Remove(sockPath)
		if err != nil {
			return err
		}
		return nil
	} else if os.IsNotExist(err) {
		return nil
	} else {
		return fmt.Errorf("fail to stat uds socket path")
	}
}
