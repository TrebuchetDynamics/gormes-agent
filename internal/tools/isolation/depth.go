package isolation

type IsolationLevel int

const (
	IsolationProcess IsolationLevel = iota
	IsolationContainer
	IsolationVM
)

func (l IsolationLevel) String() string {
	switch l {
	case IsolationProcess:
		return "process"
	case IsolationContainer:
		return "container"
	case IsolationVM:
		return "vm"
	default:
		return "unknown"
	}
}

type IsolationConfig struct {
	Level          IsolationLevel
	ContainerImage string
	VMSocket       string
}

func DefaultIsolationConfig() IsolationConfig {
	return IsolationConfig{Level: IsolationProcess}
}

func (c IsolationConfig) IsAvailable() bool {
	switch c.Level {
	case IsolationProcess:
		return true
	case IsolationContainer:
		return c.ContainerImage != ""
	case IsolationVM:
		return c.VMSocket != ""
	default:
		return false
	}
}

func (c IsolationConfig) RequiresSetup() bool {
	return c.Level != IsolationProcess
}

func ParseIsolationLevel(s string) (IsolationLevel, bool) {
	switch s {
	case "process":
		return IsolationProcess, true
	case "container":
		return IsolationContainer, true
	case "vm":
		return IsolationVM, true
	default:
		return IsolationProcess, false
	}
}

func NewIsolationConfigFromMode(mode string, containerImage string, vmSocket string) (IsolationConfig, error) {
	level, ok := ParseIsolationLevel(mode)
	if !ok {
		return IsolationConfig{Level: IsolationProcess}, &IsolationModeError{Mode: mode}
	}
	return IsolationConfig{
		Level:          level,
		ContainerImage: containerImage,
		VMSocket:       vmSocket,
	}, nil
}

type IsolationModeError struct {
	Mode string
}

func (e *IsolationModeError) Error() string {
	return "unknown isolation mode: " + e.Mode
}
