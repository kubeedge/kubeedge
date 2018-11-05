package rule

type RuleKey string

const (
	EventType     RuleKey = "event_type"
	MessageFilter RuleKey = "message_filter"
	FunctionUrn   RuleKey = "function_urn"
	TargetAddress RuleKey = "target_address"
)

type Rule struct {
	Name               string             `json:"name,omitempty"`
	Data               map[RuleKey]string `json:"data,omitempty"`
}
