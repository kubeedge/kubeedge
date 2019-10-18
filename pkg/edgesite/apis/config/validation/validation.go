package validation

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	cloudvalidation "github.com/kubeedge/kubeedge/pkg/cloudcore/apis/config/validation"
	edgevalidation "github.com/kubeedge/kubeedge/pkg/edgecore/apis/config/validation"
	"github.com/kubeedge/kubeedge/pkg/edgesite/apis/config"
)

func ValidateEdgeSideConfiguration(c *config.EdgeSideConfig) field.ErrorList {

	allErrs := field.ErrorList{}
	allErrs = append(allErrs, edgevalidation.ValidateMqttConfiguration(c.Mqtt)...)
	allErrs = append(allErrs, cloudvalidation.ValidateKubeConfiguration(c.Kube)...)
	allErrs = append(allErrs, edgevalidation.ValidateEdgedConfiguration(c.Edged)...)
	return allErrs
}
