package sessiondb

import (
	"errors"
	"testing"
)

func TestIsMemoryDatabaseNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "matching directory error", err: errors.New("memory database not found at /tmp/memory.db"), want: true},
		{name: "other error", err: errors.New("permission denied"), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsMemoryDatabaseNotFound(tt.err); got != tt.want {
				t.Fatalf("IsMemoryDatabaseNotFound(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
