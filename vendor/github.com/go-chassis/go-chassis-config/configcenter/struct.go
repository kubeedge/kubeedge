package configcenter

//ConfigCenterEvent stores info about an config center event
type Event struct {
	Action string `json:"action"`
	Value  string `json:"value"`
}

//Instance is a struct
type Instance struct {
	Status      string   `json:"status"`
	ServiceName string   `json:"serviceName"`
	IsHTTPS     bool     `json:"isHttps"`
	EntryPoints []string `json:"endpoints"`
}

//Members is a struct
type Members struct {
	Instances []Instance `json:"instances"`
}
type DeleteConfigApi struct {
	DimensionInfo string   `json:"dimensionsInfo"`
	Keys          []string `json:"keys"`
}

type CreateConfigApi struct {
	DimensionInfo string                 `json:"dimensionsInfo"`
	Items         map[string]interface{} `json:"items"`
}
