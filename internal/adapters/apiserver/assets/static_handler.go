package assets

import (
	"net/http"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/apiserver/assets/staticfiles"
)

// StaticHandler serves embedded dashboard static assets under /static/.
func StaticHandler() http.Handler {
	return staticfiles.Handler()
}
