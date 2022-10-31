package config

import (
	"testing"

	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha2"
)

func TestInitConfigure(t *testing.T) {
	t.Run("configure only once", func(t *testing.T) {
		eh := &v1alpha2.EdgeHub{
			WebSocket: &v1alpha2.EdgeHubWebSocket{
				Server: "test_server",
			},
			ProjectID: "e632aba927ea4ac2b575ec1603d56f10",
		}
		InitConfigure(eh, "test 1")
		InitConfigure(eh, "test 2")
		InitConfigure(eh, "test 3")

		if Config.NodeName != "test 1" {
			t.Fatalf("InitConfigure() changes the config for each function calls")
		}
	})
}
