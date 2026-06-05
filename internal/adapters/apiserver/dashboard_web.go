package apiserver

import (
	"net/http"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/apiserver/assets"
)

func staticHandler() http.Handler {
	return assets.StaticHandler()
}
