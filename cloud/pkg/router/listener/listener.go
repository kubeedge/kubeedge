package listener

type Handle func(interface{}) (interface{},error)

type Listener interface {
	AddListener(interface{}, Handle)
	RemoveListener(interface{})
}
