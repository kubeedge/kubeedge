package messagelayer

import (
	"fmt"
	relayconstants "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/cloudrelay/constants"
	"github.com/kubeedge/kubeedge/common/constants"
	"strings"
)

func BuildResource(nodeID string, oldResource string) (string, string, error) {
	sli := strings.Split(oldResource, constants.ResourceSep)
	if len(sli) <= relayconstants.ResourceNodeIDIndex {
		return "", "", fmt.Errorf("node id not found in building fake resource")
	}
	oldID := sli[relayconstants.ResourceNodeIDIndex]
	sli[relayconstants.ResourceNodeIDIndex] = nodeID
	res := strings.Join(sli, "/")

	return oldID, res, nil
}
