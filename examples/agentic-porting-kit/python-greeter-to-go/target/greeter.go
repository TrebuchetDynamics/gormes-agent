package greeter

import "strings"

func Hello(name string) string {
	cleaned := strings.TrimSpace(name)
	if cleaned == "" {
		cleaned = "world"
	}
	return "Hello, " + cleaned + "!"
}
