package model

import (
	gatewaydelivery "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/delivery"
	modeldelivery "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/model/delivery"
)

// DeliveryTarget converts a directory entry into the gateway delivery target
// contract. Keeping this compatibility wrapper separate from Entry normalization
// lets the core directory value contract stay independent from gateway delivery
// types.
func DeliveryTarget(platform string, entry Entry) gatewaydelivery.Target {
	return modeldelivery.Target(platform, entry)
}
