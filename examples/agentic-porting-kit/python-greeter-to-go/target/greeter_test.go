package greeter

import "testing"

func TestHelloMatchesPythonGreeting(t *testing.T) {
	tests := map[string]string{
		"Ada": "Hello, Ada!",
		"":    "Hello, world!",
		"  ":  "Hello, world!",
	}
	for input, want := range tests {
		if got := Hello(input); got != want {
			t.Fatalf("Hello(%q) = %q, want %q", input, got, want)
		}
	}
}
