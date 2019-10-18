package config

type Modules struct {
	//default devicecontroller, edgecontroller, cloudhub
	Enabled []string `json:"enabled,omitempty"`
}
