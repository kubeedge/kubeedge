package dev

type DevStatus int

const (
	DEVSTOK        = iota
	DEVSTERR       /* Expected value is not equal as setting */
	DEVSTDISCONN   /* Disconnected */
	DEVSTUNHEALTHY /* Unhealthy status from device */
	DEVSTUNKNOWN
)
