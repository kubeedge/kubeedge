package cmd

import (
	"github.com/kubeedge/kubeedge/edgemesh/pkg/server"
)

func main() {
	//Initialize the resolvers

	//Initialize the handlers
	go server.DnsStart()
	//Start server
	server.StartTCP()
}
