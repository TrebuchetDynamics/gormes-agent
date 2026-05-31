package lifecycle

import "sync"

type Event string

const (
	EventNone      Event = ""
	EventReconnect Event = "reconnect"
	EventShutdown  Event = "shutdown"
)

type Server struct {
	mu        sync.Mutex
	reconnect bool
	shutdown  bool
}

func NewServer() *Server {
	return &Server{}
}

func (l *Server) SignalReconnect() {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.reconnect = true
}

func (l *Server) SignalShutdown() {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.shutdown = true
}

func (l *Server) ReconnectPending() bool {
	if l == nil {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.reconnect
}

func (l *Server) NextEvent() Event {
	if l == nil {
		return EventNone
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.shutdown {
		l.reconnect = false
		l.shutdown = false
		return EventShutdown
	}
	if l.reconnect {
		l.reconnect = false
		return EventReconnect
	}
	return EventNone
}
