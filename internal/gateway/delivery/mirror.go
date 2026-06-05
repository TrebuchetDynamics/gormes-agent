package delivery

import (
	"context"
	"time"

	deliverymirror "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/delivery/mirror"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/store"
)

const (
	DeliveryMirrorSessionMissing   = deliverymirror.DeliveryMirrorSessionMissing
	DeliveryMirrorStoreUnavailable = deliverymirror.DeliveryMirrorStoreUnavailable
)

type DeliveryMirrorTarget = deliverymirror.DeliveryMirrorTarget
type DeliveryMirrorResult = deliverymirror.DeliveryMirrorResult

func SelectDeliveryMirrorSession(candidates []session.Metadata, target DeliveryMirrorTarget) (session.Metadata, bool) {
	return deliverymirror.SelectDeliveryMirrorSession(candidates, target)
}

func MirrorDeliveryToSession(ctx context.Context, st store.Store, candidates []session.Metadata, target DeliveryMirrorTarget, now time.Time) (DeliveryMirrorResult, error) {
	return deliverymirror.MirrorDeliveryToSession(ctx, st, candidates, target, now)
}
