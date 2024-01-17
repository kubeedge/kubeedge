package fsm

import api "github.com/kubeedge/kubeedge/pkg/apis/fsm/v1alpha1"

const (
	RollingBackState  api.State = "RollingBack"
	RollbackInitState api.State = "RollbackInit"
)

var RollbackRule = map[string]api.State{
	"/Init/Init": api.TaskInit,

	"Init/Check/Init":        api.TaskChecking,
	"Checking/Check/Success": RollbackInitState,
	"Checking/Check/Failure": api.TaskFailed,

	"RollbackInit/Rollback/Start":  RollingBackState,
	"RollingBack/Rollback/Failure": api.TaskFailed,
	"RollingBack/Rollback/Success": api.TaskFailed,
}
