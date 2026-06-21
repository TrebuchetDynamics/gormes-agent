package deepseekv31

import "fmt"

type degradedError string

func (e degradedError) Error() string { return string(e) }
func (e degradedError) Degraded() bool { return true }

func degradedf(format string, args ...any) degradedError {
	return degradedError(fmt.Sprintf(format, args...))
}
