package bedrock

import (
	"net/http"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/bedrock/sigv4"
)

type StaticAWSCredentials = sigv4.StaticAWSCredentials

func SignBedrockRequest(req *http.Request, creds StaticAWSCredentials, now time.Time) error {
	return sigv4.SignBedrockRequest(req, creds, now)
}
