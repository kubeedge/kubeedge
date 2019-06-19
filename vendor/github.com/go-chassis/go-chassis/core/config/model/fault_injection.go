package model

import "time"

// FaultProtocolStruct fault protocol struct
type FaultProtocolStruct struct {
	Fault map[string]Fault `yaml:"protocols"`
}

// Fault fault struct
type Fault struct {
	Abort Abort `yaml:"abort"`
	Delay Delay `yaml:"delay"`
}

// Abort abort struct
type Abort struct {
	Percent    int `yaml:"percent"`
	HTTPStatus int `yaml:"httpStatus"`
}

// Delay delay struct
type Delay struct {
	Percent    int           `yaml:"percent"`
	FixedDelay time.Duration `yaml:"fixedDelay"`
}
