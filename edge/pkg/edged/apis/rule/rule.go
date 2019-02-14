package rule

//Key string type for rules exporting
type Key string

//constants for different rules key
const (
	EventType     Key = "event_type"
	MessageFilter Key = "message_filter"
	FunctionUrn   Key = "function_urn"
	TargetAddress Key = "target_address"
)

//Rule defines map of rules
type Rule struct {
	Name string         `json:"name,omitempty"`
	Data map[Key]string `json:"data,omitempty"`
}
