package validation

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	cloudvalidation "github.com/kubeedge/kubeedge/pkg/cloudcore/apis/config/validation"
	edgevalidation "github.com/kubeedge/kubeedge/pkg/edgecore/apis/config/validation"
	"github.com/kubeedge/kubeedge/pkg/edgesite/apis/config"
)

func ValidateEdgeSideConfiguration(c *config.EdgeSideConfig) field.ErrorList {
	/*
		Mqtt              *edgecoreconfig.MqttConfig         `json:"mqtt,omitempty"`
		Kube              *cloudcoreconfig.KubeConfig        `json:"kube,omitempty"`
		ControllerContext *cloudcoreconfig.ControllerContext `json:"controllerContext"`
		Edged             *edgecoreconfig.EdgedConfig        `json:"edged,omitempty"`
		Modules           *commonconfig.Modules              `json:"modules,omitempty"`
		Metamanager       *Metamanager                       `json:"metamanager,omitempty"`

	*/

	allErrs := field.ErrorList{}
	allErrs = append(allErrs, edgevalidation.ValidateMqttConfiguration(c.Mqtt)...)
	allErrs = append(allErrs, cloudvalidation.ValidateKubeConfiguration(c.Kube)...)
	allErrs = append(allErrs, cloudvalidation.ValidateControllerContextConfiguration(c.ControllerContext)...)
	allErrs = append(allErrs, edgevalidation.ValidateEdgedConfiguration(c.Edged)...)
	return allErrs
}
