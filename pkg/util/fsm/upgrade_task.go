package fsm

import api "github.com/kubeedge/kubeedge/pkg/apis/fsm/v1alpha1"

const (
	UpgradingState   api.State = "Upgrading"
	UpgradeInitState api.State = "UpgradeInit"
)

// CurrentState/Event/Action: NextState
var UpgradeRule = map[string]api.State{
	"/Init/Init": api.TaskInit,

	"Init/Check/Init":        api.TaskChecking,
	"Checking/Check/Success": BackupInitState,
	"Checking/Check/Failure": api.TaskFailed,

	"BackupInit/Backup/Start":  BackingUpState,
	"BackingUp/Backup/Success": UpgradeInitState,
	"BackingUp/Backup/Failure": api.TaskFailed,

	"UpgradeInit/Upgrade/Start": UpgradingState,
	"Upgrading/Upgrade/Success": api.TaskSuccessful,
	"Upgrading/Upgrade/Failure": RollbackInitState,

	"RollbackInit/Rollback/Start":  RollingBackState,
	"RollingBack/Rollback/Failure": api.TaskFailed,
	"RollingBack/Rollback/Success": api.TaskFailed,
}
