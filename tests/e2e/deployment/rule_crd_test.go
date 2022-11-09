package deployment

import (
	"bytes"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "github.com/kubeedge/kubeedge/pkg/apis/rules/v1"
	edgeclientset "github.com/kubeedge/kubeedge/pkg/client/clientset/versioned"
	"github.com/kubeedge/kubeedge/tests/e2e/utils"
)

var _ = Describe("Rule Management test in E2E scenario", func() {
	var testTimer *utils.TestTimer
	var testSpecReport SpecReport
	var edgeClientSet edgeclientset.Interface

	BeforeEach(func() {
		edgeClientSet = utils.NewKubeEdgeClient(ctx.Cfg.KubeConfigPath)
	})

	msg := "Hello World!"
	Context("Test rule and ruleendpoint Creation and deletion", func() {
		BeforeEach(func() {
			// Delete any pre-existing rules
			list, err := utils.ListRule(edgeClientSet, "default")
			Expect(err).To(BeNil())
			for _, rule := range list {
				err := utils.HandleRule(edgeClientSet, http.MethodDelete, rule.Name, "", "")
				Expect(err).To(BeNil())
			}
			// Delete any pre-existing ruleendpoints
			reList, err := utils.ListRuleEndpoint(edgeClientSet, "default")
			Expect(err).To(BeNil())
			for _, ruleendpoint := range reList {
				err := utils.HandleRule(edgeClientSet, http.MethodDelete, ruleendpoint.Name, "", "")
				Expect(err).To(BeNil())
			}
			// Get current test SpecReport
			testSpecReport = CurrentSpecReport()
			// Start test timer
			testTimer = CRDTestTimerGroup.NewTestTimer(testSpecReport.LeafNodeText)
		})
		AfterEach(func() {
			// End test timer
			testTimer.End()
			// Print result
			testTimer.PrintResult()
			list, err := utils.ListRule(edgeClientSet, "default")
			Expect(err).To(BeNil())
			for _, rule := range list {
				err := utils.HandleRule(edgeClientSet, http.MethodDelete, rule.Name, "", "")
				Expect(err).To(BeNil())
			}
			// Delete ruleendpoints
			reList, err := utils.ListRuleEndpoint(edgeClientSet, "default")
			Expect(err).To(BeNil())
			for _, ruleendpoint := range reList {
				err := utils.HandleRuleEndpoint(edgeClientSet, http.MethodDelete, ruleendpoint.Name, "")
				Expect(err).To(BeNil())
			}

			utils.PrintTestcaseNameandStatus()
		})
		It("E2E_CREATE_RULE_1: Create rule: rest to eventbus.", func() {
			// create rest ruleendpoint
			err := utils.HandleRuleEndpoint(edgeClientSet, http.MethodPost, "", v1.RuleEndpointTypeRest)
			Expect(err).To(BeNil())
			// create eventbus ruleendpoint
			err = utils.HandleRuleEndpoint(edgeClientSet, http.MethodPost, "", v1.RuleEndpointTypeEventBus)
			Expect(err).To(BeNil())
			// create rule: rest to eventbus.
			err = utils.HandleRule(edgeClientSet, http.MethodPost, "", v1.RuleEndpointTypeRest, v1.RuleEndpointTypeEventBus)
			Expect(err).To(BeNil())

			newRule := utils.NewRule(v1.RuleEndpointTypeRest, v1.RuleEndpointTypeEventBus)
			ruleList, err := utils.ListRule(edgeClientSet, "default")
			Expect(err).To(BeNil())

			err = utils.CheckRuleExists(ruleList, newRule)
			Expect(err).To(BeNil())

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
			Expect(IsSend).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusOK))
			Eventually(func() bool {
				utils.Infof("receive: %s, msg: %s ", b.String(), msg)
				return b.String() == msg
			}, "30s", "2s").Should(Equal(true), "eventbus not subscribe anything.")
		})
		It("E2E_CREATE_RULE_2: Create rule: eventbus to rest.", func() {
			err := utils.HandleRuleEndpoint(edgeClientSet, http.MethodPost, "", v1.RuleEndpointTypeRest)
			Expect(err).To(BeNil())
			// create eventbus ruleendpoint
			err = utils.HandleRuleEndpoint(edgeClientSet, http.MethodPost, "", v1.RuleEndpointTypeEventBus)
			Expect(err).To(BeNil())
			// create rule: eventbus to rest.
			err = utils.HandleRule(edgeClientSet, http.MethodPost, "", v1.RuleEndpointTypeEventBus, v1.RuleEndpointTypeRest)
			Expect(err).To(BeNil())
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
			Expect(err).Should(BeNil())
			Eventually(func() bool {
				return b.String() == msg
			}, "30s", "2s").Should(Equal(true), "endpoint not listen any request.")
		})
		It("E2E_CREATE_RULE_3: Create rule: rest to servicebus.", func() {
			// create rest ruleendpoint
			err := utils.HandleRuleEndpoint(edgeClientSet, http.MethodPost, "", v1.RuleEndpointTypeRest)
			Expect(err).To(BeNil())
			// create servicebus ruleendpoint
			err = utils.HandleRuleEndpoint(edgeClientSet, http.MethodPost, "", v1.RuleEndpointTypeServiceBus)
			Expect(err).To(BeNil())
			// create rule: rest to servicebus
			err = utils.HandleRule(edgeClientSet, http.MethodPost, "", v1.RuleEndpointTypeRest, v1.RuleEndpointTypeServiceBus)
			Expect(err).To(BeNil())
			newRule := utils.NewRule(v1.RuleEndpointTypeRest, v1.RuleEndpointTypeServiceBus)

			ruleList, err := utils.ListRule(edgeClientSet, "default")
			Expect(err).To(BeNil())

			err = utils.CheckRuleExists(ruleList, newRule)
			Expect(err).To(BeNil())

			Expect(err).To(BeNil())
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
			Expect(IsSend).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusOK))
			Eventually(func() bool {
				utils.Infof("receive: %s, sent msg: %s ", b.String(), msg)
				newMsg := "Reply from server: " + msg + " Header of the message: [user]: " + msgHeader["user"] +
					", [passwd]: " + msgHeader["passwd"]
				return b.String() == newMsg
			}, "30s", "2s").Should(Equal(true), "servicebus did not return any response.")
		})
		It("E2E_CREATE_RULE_4: Create rule: servicebus to rest.", func() {
			r := "Hello World"
			// create rest ruleendpoint
			err := utils.HandleRuleEndpoint(edgeClientSet, http.MethodPost, "", v1.RuleEndpointTypeRest)
			Expect(err).To(BeNil())
			// create servicebus ruleendpoint
			err = utils.HandleRuleEndpoint(edgeClientSet, http.MethodPost, "", v1.RuleEndpointTypeServiceBus)
			Expect(err).To(BeNil())
			// create rule: servicebus to rest
			err = utils.HandleRule(edgeClientSet, http.MethodPost, "", v1.RuleEndpointTypeServiceBus, v1.RuleEndpointTypeRest)
			Expect(err).To(BeNil())
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
			Expect(err).Should(BeNil())
			Expect(resp).Should(Equal(r))
		})
	})
})
