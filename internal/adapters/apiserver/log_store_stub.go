package apiserver

import "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/apiserver/logs"

type LogStore = logs.Store

func NewLogStore(retentionDays int) *LogStore { return logs.NewStore(retentionDays) }
