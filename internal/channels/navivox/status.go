package navivox

import "context"

type StatusProvider interface {
	Status(context.Context) (ServerStatus, error)
}

type ServerStatus struct {
	GormesVersion  string   `json:"gormes_version"`
	ConfigVersion  string   `json:"config_version"`
	Protocol       uint32   `json:"protocol"`
	Features       []string `json:"features"`
	ActiveChannels []string `json:"active_channels,omitempty"`
}

type StaticStatusProvider struct {
	StatusValue ServerStatus
	Err         error
}

func (p StaticStatusProvider) Status(context.Context) (ServerStatus, error) {
	if p.Err != nil {
		return ServerStatus{}, p.Err
	}
	status := p.StatusValue
	if status.Protocol == 0 {
		status.Protocol = ProtocolVersion
	}
	if len(status.Features) == 0 {
		status.Features = DefaultFeatures()
	}
	return status, nil
}

func DefaultFeatures() []string {
	return []string{
		"hello",
		"server.status",
		"ping",
		"pong",
	}
}
