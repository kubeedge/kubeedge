package v1alpha1

const (
	UpgradingState State = "Upgrading"
)

// CurrentState/Event/Action: NextState
var UpgradeRule = map[string]State{
	"Init/Init/Success":    TaskChecking,
	"Init/Init/Failure":    TaskFailed,
	"Init/TimeOut/Failure": TaskFailed,
	"Init/Upgrade/Success": TaskSuccessful,

	"Checking/Check/Success":   BackingUpState,
	"Checking/Check/Failure":   TaskFailed,
	"Checking/TimeOut/Failure": TaskFailed,

	"BackingUp/Backup/Success":  UpgradingState,
	"BackingUp/Backup/Failure":  TaskFailed,
	"BackingUp/TimeOut/Failure": TaskFailed,

	"Upgrading/Upgrade/Success": TaskSuccessful,
	"Upgrading/Upgrade/Failure": RollingBackState,
	"Upgrading/TimeOut/Failure": RollingBackState,

	"RollingBack/Rollback/Failure": TaskFailed,
	"RollingBack/TimeOut/Failure":  TaskFailed,
	"RollingBack/Rollback/Success": TaskFailed,

	"Upgrading/Rollback/Failure": TaskFailed,
	"Upgrading/Rollback/Success": TaskFailed,

	//TODO delete in version 1.18
	"Init/Rollback/Failure": TaskFailed,
	"Init/Rollback/Success": TaskFailed,
}

var UpdateStageSequence = map[State]State{
	"":             TaskChecking,
	TaskInit:       TaskChecking,
	TaskChecking:   BackingUpState,
	BackingUpState: UpgradingState,
	UpgradingState: RollingBackState,
}
