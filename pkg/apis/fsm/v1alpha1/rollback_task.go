package v1alpha1

const (
	RollingBackState State = "RollingBack"
)

var RollbackRule = map[string]State{
	"/Init/Success": TaskChecking,

	"Checking/Check/Success": RollingBackState,
	"Checking/Check/Failure": TaskFailed,

	"RollingBack/Rollback/Failure": TaskFailed,
	"RollingBack/Rollback/Success": TaskFailed,
}
