package util

type MQTTInstTool struct {
	Common
}

func (m *MQTTInstTool) InstallTools() error {
	m.SetOSInterface(GetOSInterface())
	err := m.InstallMQTT()
	if err != nil {
		return err
	}
	return nil
}
