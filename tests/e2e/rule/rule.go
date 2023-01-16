package rule

import (
	"bytes"
	"net/http"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"k8s.io/kubernetes/test/e2e/framework"

	v1 "github.com/kubeedge/kubeedge/pkg/apis/rules/v1"
	edgeclientset "github.com/kubeedge/kubeedge/pkg/client/clientset/versioned"
	"github.com/kubeedge/kubeedge/tests/e2e/utils"
)

var _ = GroupDescribe("Rule Management test in E2E scenario", func() {
	var testTimer *utils.TestTimer
	var testSpecReport ginkgo.GinkgoTestDescription
	var edgeClientSet edgeclientset.Interface

	ginkgo.BeforeEach(func() {
		edgeClientSet = utils.NewKubeEdgeClient(framework.TestContext.KubeConfig)
	})

	msg := "Hello World!"
	ginkgo.Context("Test rule and ruleendpoint Creation and deletion", func() {
		ginkgo.BeforeEach(func() {
			// Delete any pre-existing rules
			list, err := utils.ListRule(edgeClientSet, "default")
			gomega.Expect(err).To(gomega.BeNil())
			for _, rule := range list {
				err := utils.HandleRule(edgeClientSet, http.MethodDelete, rule.Name, "", "")
				gomega.Expect(err).To(gomega.BeNil())
			}
			// Delete any pre-existing ruleendpoints
			reList, err := utils.ListRuleEndpoint(edgeClientSet, "default")
			gomega.Expect(err).To(gomega.BeNil())
			for _, ruleendpoint := range reList {
				err := utils.HandleRule(edgeClientSet, http.MethodDelete, ruleendpoint.Name, "", "")
				gomega.Expect(err).To(gomega.BeNil())
			}
			// Get current test SpecReport
			testSpecReport = ginkgo.CurrentGinkgoTestDescription()
			// Start test timer
			testTimer = utils.CRDTestTimerGroup.NewTestTimer(testSpecReport.TestText)
		})
		ginkgo.AfterEach(func() {
			// End test timer
			testTimer.End()
			// Print result
			testTimer.PrintResult()
			list, err := utils.ListRule(edgeClientSet, "default")
			gomega.Expect(err).To(gomega.BeNil())
			for _, rule := range list {
				err := utils.HandleRule(edgeClientSet, http.MethodDelete, rule.Name, "", "")
				gomega.Expect(err).To(gomega.BeNil())
			}
			// Delete ruleendpoints
			reList, err := utils.ListRuleEndpoint(edgeClientSet, "default")
			gomega.Expect(err).To(gomega.BeNil())
			for _, ruleendpoint := range reList {
				err := utils.HandleRuleEndpoint(edgeClientSet, http.MethodDelete, ruleendpoint.Name, "")
				gomega.Expect(err).To(gomega.BeNil())
			}

			utils.PrintTestcaseNameandStatus()
		})
		ginkgo.It("E2E_CREATE_RULE_1: Create rule: rest to eventbus.", func() {
			// create rest ruleendpoint
			err := utils.HandleRuleEndpoint(edgeClientSet, http.MethodPost, "", v1.RuleEndpointTypeRest)
			gomega.Expect(err).To(gomega.BeNil())
			// create eventbus ruleendpoint
			err = utils.HandleRuleEndpoint(edgeClientSet, http.MethodPost, "", v1.RuleEndpointTypeEventBus)
			gomega.Expect(err).To(gomega.BeNil())
			// create rule: rest to eventbus.
			err = utils.HandleRule(edgeClientSet, http.MethodPost, "", v1.RuleEndpointTypeRest, v1.RuleEndpointTypeEventBus)
			gomega.Expect(err).To(gomega.BeNil())

			newRule := utils.NewRule(v1.RuleEndpointTypeRest, v1.RuleEndpointTypeEventBus)
			ruleList, err := utils.ListRule(edgeClientSet, "default")
			gomega.Expect(err).To(gomega.BeNil())

			err = utils.CheckRuleExists(ruleList, newRule)
			gomega.Expect(err).To(gomega.BeNil())

			b := new(bytes.Buffer)
			go func() {
				receiveMsg, err := utils.SubscribeMqtt("topic-test")
				if err != nil {
					utils.Fatalf("subscribe topic-test fail. reason: %s. ", err.Error())
				}
				b.WriteString(receiveMsg)
			}()
			time.Sleep(3 * time.Second)
			// call rest api to send message to edge.
			IsSend, statusCode := utils.SendMsg("http://127.0.0.1:9443/edge-node/default/ccc", []byte(msg), nil)
			gomega.Expect(IsSend).Should(gomega.BeTrue())
			gomega.Expect(statusCode).Should(gomega.Equal(http.StatusOK))
			gomega.Eventually(func() bool {
				utils.Infof("receive: %s, msg: %s ", b.String(), msg)
				return b.String() == msg
			}, "30s", "2s").Should(gomega.Equal(true), "eventbus not subscribe anything.")
		})
		ginkgo.It("E2E_CREATE_RULE_2: Create rule: eventbus to rest.", func() {
			err := utils.HandleRuleEndpoint(edgeClientSet, http.MethodPost, "", v1.RuleEndpointTypeRest)
			gomega.Expect(err).To(gomega.BeNil())
			// create eventbus ruleendpoint
			err = utils.HandleRuleEndpoint(edgeClientSet, http.MethodPost, "", v1.RuleEndpointTypeEventBus)
			gomega.Expect(err).To(gomega.BeNil())
			// create rule: eventbus to rest.
			err = utils.HandleRule(edgeClientSet, http.MethodPost, "", v1.RuleEndpointTypeEventBus, v1.RuleEndpointTypeRest)
			gomega.Expect(err).To(gomega.BeNil())
			b := new(bytes.Buffer)
			go func() {
				receiveMsg, err := utils.StartEchoServer()
				if err != nil {
					utils.Fatalf("subscribe topic-test fail. reason: %s. ", err.Error())
				}
				b.WriteString(receiveMsg)
			}()
			time.Sleep(3 * time.Second)
			// call rest api to send message to edge.
			err = utils.PublishMqtt("default/test", msg)
			gomega.Expect(err).Should(gomega.BeNil())
			gomega.Eventually(func() bool {
				return b.String() == msg
			}, "30s", "2s").Should(gomega.Equal(true), "endpoint not listen any request.")
		})
		ginkgo.It("E2E_CREATE_RULE_3: Create rule: rest to servicebus.", func() {
			// create rest ruleendpoint
			err := utils.HandleRuleEndpoint(edgeClientSet, http.MethodPost, "", v1.RuleEndpointTypeRest)
			gomega.Expect(err).To(gomega.BeNil())
			// create servicebus ruleendpoint
			err = utils.HandleRuleEndpoint(edgeClientSet, http.MethodPost, "", v1.RuleEndpointTypeServiceBus)
			gomega.Expect(err).To(gomega.BeNil())
			// create rule: rest to servicebus
			err = utils.HandleRule(edgeClientSet, http.MethodPost, "", v1.RuleEndpointTypeRest, v1.RuleEndpointTypeServiceBus)
			gomega.Expect(err).To(gomega.BeNil())
			newRule := utils.NewRule(v1.RuleEndpointTypeRest, v1.RuleEndpointTypeServiceBus)

			ruleList, err := utils.ListRule(edgeClientSet, "default")
			gomega.Expect(err).To(gomega.BeNil())

			err = utils.CheckRuleExists(ruleList, newRule)
			gomega.Expect(err).To(gomega.BeNil())

			gomega.Expect(err).To(gomega.BeNil())
			msgHeader := map[string]string{
				"user":   "I am user",
				"passwd": "I am passwd",
			}
			b := new(bytes.Buffer)
			go func() {
				receiveMsg, err := utils.StartEchoServer()
				if err != nil {
					utils.Fatalf("fail to call edge-app's API. reason: %s. ", err.Error())
				}
				b.WriteString(receiveMsg)
			}()
			time.Sleep(3 * time.Second)
			// call rest api to send message to edge.
			IsSend, statusCode := utils.SendMsg("http://127.0.0.1:9443/edge-node/default/ddd", []byte(msg), msgHeader)
			gomega.Expect(IsSend).Should(gomega.BeTrue())
			gomega.Expect(statusCode).Should(gomega.Equal(http.StatusOK))
			gomega.Eventually(func() bool {
				utils.Infof("receive: %s, sent msg: %s ", b.String(), msg)
				newMsg := "Reply from server: " + msg + " Header of the message: [user]: " + msgHeader["user"] +
					", [passwd]: " + msgHeader["passwd"]
				return b.String() == newMsg
			}, "30s", "2s").Should(gomega.Equal(true), "servicebus did not return any response.")
		})
		ginkgo.It("E2E_CREATE_RULE_4: Create rule: servicebus to rest.", func() {
			r := "Hello World"
			// create rest ruleendpoint
			err := utils.HandleRuleEndpoint(edgeClientSet, http.MethodPost, "", v1.RuleEndpointTypeRest)
			gomega.Expect(err).To(gomega.BeNil())
			// create servicebus ruleendpoint
			err = utils.HandleRuleEndpoint(edgeClientSet, http.MethodPost, "", v1.RuleEndpointTypeServiceBus)
			gomega.Expect(err).To(gomega.BeNil())
			// create rule: servicebus to rest
			err = utils.HandleRule(edgeClientSet, http.MethodPost, "", v1.RuleEndpointTypeServiceBus, v1.RuleEndpointTypeRest)
			gomega.Expect(err).To(gomega.BeNil())
			b := new(bytes.Buffer)
			go func() {
				receiveMsg, err := utils.StartEchoServer()
				if err != nil {
					utils.Fatalf("fail to call cloud-app's API. reason: %s. ", err.Error())
				}
				b.WriteString(receiveMsg)
			}()
			time.Sleep(3 * time.Second)
			resp, err := utils.CallServicebus()
			gomega.Expect(err).Should(gomega.BeNil())
			gomega.Expect(resp).Should(gomega.Equal(r))
		})
	})
})
