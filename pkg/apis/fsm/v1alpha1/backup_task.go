package v1alpha1

const (
	BackingUpState State = "BackingUp"
)

var BackupRule = map[string]State{
	"/Init/Success": TaskChecking,

	"Checking/Check/Success": BackingUpState,
	"Checking/Check/Failure": TaskFailed,

	"BackingUp/Backup/Success": TaskSuccessful,
	"BackingUp/Backup/Failure": TaskFailed,
}
