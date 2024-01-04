package v1alpha1

type State string

type Action string

const (
	NodeAvailable      State = "Available"
	NodeUpgrading      State = "Upgrading"
	NodeRollingBack    State = "RollingBack"
	NodeConfigUpdating State = "ConfigUpdating"
)

const (
	TaskInit       State = "Init"
	TaskChecking   State = "Checking"
	TaskSuccessful State = "Successful"
	TaskFailed     State = "Failed"
	TaskPause      State = "Pause"
)

const (
	ActionSuccess Action = "Success"
	ActionFailure Action = "Failure"
)
