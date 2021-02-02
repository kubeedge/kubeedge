package deployment

import (
	"bytes"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "github.com/kubeedge/kubeedge/cloud/pkg/apis/rules/v1"
	"github.com/kubeedge/kubeedge/tests/e2e/utils"
)

const (
	RuleEndpointHandler = "/apis/rules.kubeedge.io/v1/namespaces/default/ruleendpoints"
	RuleHandler         = "/apis/rules.kubeedge.io/v1/namespaces/default/rules"
)

var _ = Describe("Rule Management test in E2E scenario", func() {
	var testTimer *utils.TestTimer
	var testDescription GinkgoTestDescription
	Context("Test rule and ruleendpoint Creation and deletion", func() {
		BeforeEach(func() {
			// Delete any pre-existing rules
			var ruleList v1.RuleList
			list, err := utils.GetRuleList(&ruleList, ctx.Cfg.K8SMasterForKubeEdge+RuleHandler, nil)
			Expect(err).To(BeNil())
			for _, rule := range list {
				IsRuleDeleted, statusCode := utils.HandleRule(http.MethodDelete, ctx.Cfg.K8SMasterForKubeEdge+RuleHandler, "/"+rule.Name, "", "")
				Expect(IsRuleDeleted).Should(BeTrue())
				Expect(statusCode).Should(Equal(http.StatusOK))
			}
			// Delete any pre-existing ruleendpoints
			var ruleEndpointList v1.RuleEndpointList
			reList, err := utils.GetRuleEndpointList(&ruleEndpointList, ctx.Cfg.K8SMasterForKubeEdge+RuleEndpointHandler, nil)
			Expect(err).To(BeNil())
			for _, ruleendpoint := range reList {
				IsReDeleted, statusCode := utils.HandleRule(http.MethodDelete, ctx.Cfg.K8SMasterForKubeEdge+RuleEndpointHandler, "/"+ruleendpoint.Name, "", "")
				Expect(IsReDeleted).Should(BeTrue())
				Expect(statusCode).Should(Equal(http.StatusOK))
			}
			// Get current test description
			testDescription = CurrentGinkgoTestDescription()
			// Start test timer
			testTimer = CRDTestTimerGroup.NewTestTimer(testDescription.TestText)
		})
		AfterEach(func() {
			// End test timer
			testTimer.End()
			// Print result
			testTimer.PrintResult()
			// Delete the rules created
			var ruleList v1.RuleList
			list, err := utils.GetRuleList(&ruleList, ctx.Cfg.K8SMasterForKubeEdge+RuleHandler, nil)
			Expect(err).To(BeNil())
			for _, rule := range list {
				IsRuleDeleted, statusCode := utils.HandleRule(http.MethodDelete, ctx.Cfg.K8SMasterForKubeEdge+RuleHandler, "/"+rule.Name, "", "")
				Expect(IsRuleDeleted).Should(BeTrue())
				Expect(statusCode).Should(Equal(http.StatusOK))
			}
			// Delete ruleendpoints
			var ruleEndpointList v1.RuleEndpointList
			reList, err := utils.GetRuleEndpointList(&ruleEndpointList, ctx.Cfg.K8SMasterForKubeEdge+RuleEndpointHandler, nil)
			Expect(err).To(BeNil())
			for _, ruleendpoint := range reList {
				IsReDeleted, statusCode := utils.HandleRule(http.MethodDelete, ctx.Cfg.K8SMasterForKubeEdge+RuleEndpointHandler, "/"+ruleendpoint.Name, "", "")
				Expect(IsReDeleted).Should(BeTrue())
				Expect(statusCode).Should(Equal(http.StatusOK))
			}

			utils.PrintTestcaseNameandStatus()
		})
		It("E2E_CREATE_RULE_1: Create rule: rest to eventbus.", func() {
			var ruleList v1.RuleList
			// create rest ruleendpoint
			IsRestRuleEndpointCreated, status := utils.HandleRuleEndpoint(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+RuleEndpointHandler, "", utils.RestType)
			Expect(IsRestRuleEndpointCreated).Should(BeTrue())
			Expect(status).Should(Equal(http.StatusCreated))
			// create eventbus ruleendpoint
			IsEventbusRuleEndpointCreated, status := utils.HandleRuleEndpoint(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+RuleEndpointHandler, "", utils.EventbusType)
			Expect(IsEventbusRuleEndpointCreated).Should(BeTrue())
			Expect(status).Should(Equal(http.StatusCreated))
			// create rule: rest to eventbus.
			IsRuleCreated, statusCode := utils.HandleRule(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+RuleHandler, "", utils.RestType, utils.EventbusType)
			Expect(IsRuleCreated).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusCreated))
			newRule := utils.NewRule(utils.RestType, utils.EventbusType)
			_, err := utils.GetRuleList(&ruleList, ctx.Cfg.K8SMasterForKubeEdge+RuleHandler, newRule)
			Expect(err).To(BeNil())
			msg := "Hello World!"
			b := new(bytes.Buffer)
			go func() {
				recieveMsg, err := utils.SubscribeMqtt("topic-test")
				if err != nil {
					utils.Fatalf("subscribe topic-test fail. reason: %s. ", err.Error())
				}
				b.WriteString(recieveMsg)
			}()
			time.Sleep(3 * time.Second)
			// call rest api to send message to edge.
			IsSend, statusCode := utils.SendMsg("http://127.0.0.1:9443/edge-node/default/ccc", []byte(msg))
			Expect(IsSend).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusOK))
			Eventually(func() bool {
				utils.Infof("receive: %s, msg: %s ", b.String(), msg)
				return b.String() == msg
			}, "30s", "2s").Should(Equal(true), "eventbus not subscribe anything.")
		})
		It("E2E_CREATE_RULE_2: Create rule: eventbus to rest.", func() {
			IsRestRuleEndpointCreated, status := utils.HandleRuleEndpoint(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+RuleEndpointHandler, "", utils.RestType)
			Expect(IsRestRuleEndpointCreated).Should(BeTrue())
			Expect(status).Should(Equal(http.StatusCreated))
			// create eventbus ruleendpoint
			IsEventbusRuleEndpointCreated, status := utils.HandleRuleEndpoint(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+RuleEndpointHandler, "", utils.EventbusType)
			Expect(IsEventbusRuleEndpointCreated).Should(BeTrue())
			Expect(status).Should(Equal(http.StatusCreated))
			// create rule: eventbus to rest.
			IsRuleCreated, statusCode := utils.HandleRule(http.MethodPost, ctx.Cfg.K8SMasterForKubeEdge+RuleHandler, "", utils.EventbusType, utils.RestType)
			Expect(IsRuleCreated).Should(BeTrue())
			Expect(statusCode).Should(Equal(http.StatusCreated))
			msg := "Hello World!"
			b := new(bytes.Buffer)
			go func() {
				recieveMsg, err := utils.StartEchoServer()
				if err != nil {
					utils.Fatalf("subscribe topic-test fail. reason: %s. ", err.Error())
				}
				b.WriteString(recieveMsg)
			}()
			time.Sleep(3 * time.Second)
			// call rest api to send message to edge.
			err := utils.PublishMqtt("default/test", msg)
			Expect(err).Should(BeNil())
			Eventually(func() bool {
				return b.String() == msg
			}, "30s", "2s").Should(Equal(true), "endpoint not listen any request.")
		})
	})
})
