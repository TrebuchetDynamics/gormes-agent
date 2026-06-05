package auth

import (
	"fmt"
	"strings"
)

const (
	BedrockAuthStatePresent = "present"
	BedrockAuthStateMissing = "missing"

	BedrockAuthStatusCredentialsMissing = "bedrock_credentials_missing"
	BedrockAuthStatusBearerSelected     = "bedrock_bearer_selected"
	BedrockAuthStatusProfileSelected    = "bedrock_profile_selected"
	BedrockAuthStatusStaticKeySelected  = "bedrock_static_key_selected"
	BedrockAuthStatusSigV4Unavailable   = "bedrock_sigv4_unavailable"
)

type BedrockAuthEvidence struct {
	Source string
	State  string
	Status string
}

func (e BedrockAuthEvidence) String() string {
	return fmt.Sprintf("bedrock auth source=%s state=%s status=%s", e.Source, e.State, e.Status)
}

func (e BedrockAuthEvidence) Error() string {
	return e.String()
}

func ResolveBedrockAuth(env map[string]string) BedrockAuthEvidence {
	if strings.TrimSpace(env["AWS_BEARER_TOKEN_BEDROCK"]) != "" {
		return BedrockAuthEvidence{
			Source: "AWS_BEARER_TOKEN_BEDROCK",
			State:  BedrockAuthStatePresent,
			Status: BedrockAuthStatusBearerSelected,
		}
	}
	accessKey := strings.TrimSpace(env["AWS_ACCESS_KEY_ID"])
	secretKey := strings.TrimSpace(env["AWS_SECRET_ACCESS_KEY"])
	if accessKey != "" && secretKey != "" {
		return BedrockAuthEvidence{
			Source: "AWS_ACCESS_KEY_ID",
			State:  BedrockAuthStatePresent,
			Status: BedrockAuthStatusStaticKeySelected,
		}
	}
	if accessKey != "" || secretKey != "" {
		return BedrockAuthEvidence{
			Source: "AWS_ACCESS_KEY_ID",
			State:  BedrockAuthStateMissing,
			Status: BedrockAuthStatusCredentialsMissing,
		}
	}
	if strings.TrimSpace(env["AWS_PROFILE"]) != "" {
		return BedrockAuthEvidence{
			Source: "AWS_PROFILE",
			State:  BedrockAuthStatePresent,
			Status: BedrockAuthStatusProfileSelected,
		}
	}
	if strings.TrimSpace(env["AWS_CONTAINER_CREDENTIALS_RELATIVE_URI"]) != "" {
		return BedrockAuthEvidence{
			Source: "AWS_CONTAINER_CREDENTIALS_RELATIVE_URI",
			State:  BedrockAuthStatePresent,
			Status: BedrockAuthStatusSigV4Unavailable,
		}
	}
	if strings.TrimSpace(env["AWS_WEB_IDENTITY_TOKEN_FILE"]) != "" {
		return BedrockAuthEvidence{
			Source: "AWS_WEB_IDENTITY_TOKEN_FILE",
			State:  BedrockAuthStatePresent,
			Status: BedrockAuthStatusSigV4Unavailable,
		}
	}
	return BedrockAuthEvidence{State: BedrockAuthStateMissing, Status: BedrockAuthStatusCredentialsMissing}
}

func ResolveBedrockRegion(env map[string]string) string {
	if region := strings.TrimSpace(env["AWS_REGION"]); region != "" {
		return region
	}
	if region := strings.TrimSpace(env["AWS_DEFAULT_REGION"]); region != "" {
		return region
	}
	return "us-east-1"
}
