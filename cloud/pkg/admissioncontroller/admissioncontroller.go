package admissioncontroller

import (
	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/cloud/pkg/admissioncontroller/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/admissioncontroller/controller"
	"github.com/kubeedge/kubeedge/cloud/pkg/admissioncontroller/utils"
)

func init() {
	Run()
}

// Run starts admissioncontroller
func Run() {
	cli, err := utils.KubeClient()
	if err != nil {
		klog.Fatalf("Create kube client failed with error: %v", err)
	}
	admcontroller := &controller.AdmissionController{Client: cli}
	context := utils.SetupServerCert(constants.NamespaceName, constants.ServiceName)
	admcontroller.Start(context)
}
