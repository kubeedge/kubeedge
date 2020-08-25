package model

//ControlPanel define control panel config
type ControlPanel struct {
	Infra    string            `yaml:"infra"`
	Settings map[string]string `yaml:"settings"`
}
