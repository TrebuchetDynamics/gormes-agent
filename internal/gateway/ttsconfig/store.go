package ttsconfig

import "sync"

// Store owns per-session TTS settings.
type Store struct {
	mu      sync.Mutex
	configs map[string]Config
}

// NewStore returns an empty per-session TTS config store.
func NewStore() *Store {
	return &Store{configs: map[string]Config{}}
}

// Get returns the stored config for sessionKey, creating the default config on
// first use to preserve the gateway's historical per-session behavior.
func (s *Store) Get(sessionKey string) Config {
	if s == nil {
		return DefaultConfig
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.configs == nil {
		s.configs = map[string]Config{}
	}
	cfg, ok := s.configs[sessionKey]
	if !ok {
		cfg = DefaultConfig
		s.configs[sessionKey] = cfg
	}
	return cfg
}

// Set stores cfg for sessionKey.
func (s *Store) Set(sessionKey string, cfg Config) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.configs == nil {
		s.configs = map[string]Config{}
	}
	s.configs[sessionKey] = cfg
}
