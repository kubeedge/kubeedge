package fsm

import api "github.com/kubeedge/kubeedge/pkg/apis/fsm/v1alpha1"

const (
	BackingUpState  api.State = "BackingUp"
	BackupInitState api.State = "BackupInit"
)

var BackupRule = map[string]api.State{
	"/Init/Init": api.TaskInit,

	"Init/Check/Init":        api.TaskChecking,
	"Checking/Check/Success": BackupInitState,
	"Checking/Check/Failure": api.TaskFailed,

	"BackupInit/Backup/Start":  BackingUpState,
	"BackingUp/Backup/Success": api.TaskSuccessful,
	"BackingUp/Backup/Failure": api.TaskFailed,
}
