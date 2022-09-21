package messagelayer

import (
	"fmt"
	relayconstants "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/cloudrelay/constants"
	"github.com/kubeedge/kubeedge/common/constants"
	"strings"
)

func BuildResource(nodeID string, oldResource string) (resource string, err error) {
	sli := strings.Split(oldResource, constants.ResourceSep)
	if len(sli) <= relayconstants.ResourceNodeIDIndex {
		return "", fmt.Errorf("node id not found in building fake resource")
	}
	sli[relayconstants.ResourceNodeIDIndex] = nodeID
	res := strings.Join(sli, "/")

	return res, nil
}
