//go:build gormes_lite

package main

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools"

func registerAudioTools(_ *tools.Registry) {}

func audioToolsEnabled() bool { return false }
