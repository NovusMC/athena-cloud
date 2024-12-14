package common

type ServiceState string

const (
	ServiceStatePending   ServiceState = "pending"
	ServiceStateWaiting   ServiceState = "waiting"
	ServiceStateScheduled ServiceState = "scheduled"
	ServiceStateRunning   ServiceState = "running"
)

type ServiceType string

const (
	ServiceTypeProxy  ServiceType = "proxy"
	ServiceTypeServer             = "server"
)

type ServiceInfo struct {
	Name  string       `json:"name"`
	State ServiceState `json:"state"`
	Type  ServiceType  `json:"type"`
	Group string       `json:"group"`
	Slave string       `json:"slave"`
	Port  int          `json:"port"`
}
