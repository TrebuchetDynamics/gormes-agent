package gateway

import (
	"context"
	"time"

	gatewaydelivery "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/delivery"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/store"
)

const (
	DeliveryMirrorSessionMissing   = gatewaydelivery.DeliveryMirrorSessionMissing
	DeliveryMirrorStoreUnavailable = gatewaydelivery.DeliveryMirrorStoreUnavailable
)

type DeliveryMirrorTarget = gatewaydelivery.DeliveryMirrorTarget

type DeliveryMirrorResult = gatewaydelivery.DeliveryMirrorResult

func SelectDeliveryMirrorSession(candidates []session.Metadata, target DeliveryMirrorTarget) (session.Metadata, bool) {
	return gatewaydelivery.SelectDeliveryMirrorSession(candidates, target)
}

func MirrorDeliveryToSession(ctx context.Context, st store.Store, candidates []session.Metadata, target DeliveryMirrorTarget, now time.Time) (DeliveryMirrorResult, error) {
	return gatewaydelivery.MirrorDeliveryToSession(ctx, st, candidates, target, now)
}
