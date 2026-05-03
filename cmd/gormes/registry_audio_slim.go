//go:build !gormes_lite && slim

package main

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func registerAudioTools(_ *tools.Registry, _ config.Config) {}

func audioToolsEnabled() bool { return false }
