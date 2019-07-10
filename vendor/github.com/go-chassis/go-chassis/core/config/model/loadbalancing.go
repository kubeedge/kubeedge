package model

// LBWrapper loadbalancing structure
type LBWrapper struct {
	Prefix *LoadBalancingConfig `yaml:"cse"`
}

// LoadBalancingConfig loadbalancing structure
type LoadBalancingConfig struct {
	LBConfig *LoadBalancing `yaml:"loadbalance"`
}

// LoadBalancing loadbalancing structure
type LoadBalancing struct {
	Strategy              map[string]string            `yaml:"strategy"`
	RetryEnabled          bool                         `yaml:"retryEnabled"`
	RetryOnNext           int                          `yaml:"retryOnNext"`
	RetryOnSame           int                          `yaml:"retryOnSame"`
	Filters               string                       `yaml:"serverListFilters"`
	Backoff               BackoffStrategy              `yaml:"backoff"`
	SessionStickinessRule SessionStickinessRule        `yaml:"SessionStickinessRule"`
	AnyService            map[string]LoadBalancingSpec `yaml:",inline"`
}

// LoadBalancingSpec loadbalancing structure
type LoadBalancingSpec struct {
	Strategy              map[string]string     `yaml:"strategy"`
	SessionStickinessRule SessionStickinessRule `yaml:"SessionStickinessRule"`
	RetryEnabled          bool                  `yaml:"retryEnabled"`
	RetryOnNext           int                   `yaml:"retryOnNext"`
	RetryOnSame           int                   `yaml:"retryOnSame"`
	Backoff               BackoffStrategy       `yaml:"backoff"`
}

// SessionStickinessRule loadbalancing structure
type SessionStickinessRule struct {
	SessionTimeoutInSeconds int `yaml:"sessionTimeoutInSeconds"`
	SuccessiveFailedTimes   int `yaml:"successiveFailedTimes"`
}

// BackoffStrategy back off strategy
type BackoffStrategy struct {
	Kind  string `yaml:"kind"`
	MinMs int    `yaml:"minMs"`
	MaxMs int    `yaml:"maxMs"`
}
