package main

import "github.com/TrebuchetDynamics/gormes-agent/gormes/internal/tools"

// buildDefaultRegistry returns a Registry populated with Gormes's built-in
// Go-native tools (echo, now, rand_int). Consumer forks that want to add
// domain-specific tools (scientific simulators, business wrappers, etc.)
// call reg.Register on the returned *Registry before passing it into the
// kernel Config. Gormes itself ships no domain-specific tools.
func buildDefaultRegistry() *tools.Registry {
	reg := tools.NewRegistry()
	reg.MustRegister(&tools.EchoTool{})
	reg.MustRegister(&tools.NowTool{})
	reg.MustRegister(&tools.RandIntTool{})
	return reg
}
