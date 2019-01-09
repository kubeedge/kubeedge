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

package e2e

import (
	"crypto/tls"
	"testing"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/kubeedge/kubeedge/test/Edge-SDV/utils/common"
	"github.com/kubeedge/kubeedge/test/Edge-SDV/utils/edge"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	ctx *edge.TestContext
	cfg edge.Config
)

type Token interface {
	Wait() bool
	WaitTimeout(time.Duration) bool
	Error() error
}

const (
	Device_upload            = "$hw/events/upload/#"
	Device_status_update     = "$hw/events/device/+/state/update"
	Device_Twin_update       = "$hw/events/device/+/twin/+"
	Device_Membership_update = "$hw/events/node/+/membership/get"
	Upload_record_to_cloud   = "SYS/dis/upload_records"
	clientID                 = "eventbus"
)

// HubclientInit create mqtt client config
func HubclientInit(server, clientID, username, password string) *MQTT.ClientOptions {
	opts := MQTT.NewClientOptions().AddBroker(server).SetClientID(clientID).SetCleanSession(true)
	if username != "" {
		opts.SetUsername(username)
		if password != "" {
			opts.SetPassword(password)
		}
	}
	tlsConfig := &tls.Config{InsecureSkipVerify: true, ClientAuth: tls.NoClientCert}
	opts.SetTLSConfig(tlsConfig)
	return opts
}

func TestEdgecore_EventBus(t *testing.T) {
	RegisterFailHandler(Fail)
	var _ = BeforeSuite(func() {
		common.InfoV6("Before Suit : Start the Build Script")

		cfg = edge.LoadConfig()
		ctx = edge.NewTestContext(cfg)

	})
	AfterSuite(func() {
		By("Executing the AfterSuit....!")
	})

	RunSpecs(t, "edgecore Suite")
}
