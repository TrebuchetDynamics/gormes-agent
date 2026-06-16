package cron

import "github.com/TrebuchetDynamics/gormes-agent/internal/automation/cron/delivery"

// DeliverySink is the abstraction between the cron executor and outbound delivery.
type DeliverySink = delivery.DeliverySink

// FuncSink adapts a plain function to the DeliverySink interface.
type FuncSink = delivery.FuncSink
