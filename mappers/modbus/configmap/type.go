package configmap

type ModbusVisitorConfig struct {
	Register       string `json:"register"`
	Offset         uint16 `json:"offset"`
	Limit          int    `json:"limit"`
	Scale          int    `json:"scale,omitempty"`
	IsSwap         bool   `json:"isSwap,omitempty"`
	IsRegisterSwap bool   `json:"isRegisterSwap,omitempty"`
}

type ModbusProtocolCommonConfig struct {
	Com COM `json:"com,omitempty"`
	Tcp TCP `json:"tcp,omitempty"`
}

type COM struct {
	SerialPort string `json:"serialPort"`
	BaudRate   int64  `json:"baudRate"`
	DataBits   int64  `json:"dataBits"`
	Parity     string `json:"parity"`
	StopBits   int64  `json:"parity"`
}

type TCP struct {
	Ip   string `json:"ip"`
	Port int64  `json:"port"`
}
