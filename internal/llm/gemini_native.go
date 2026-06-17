package llm

import (
	"encoding/json"
	"io"
)

const geminiNativeAPIMode = "gemini_native"

type geminiNativeTransport struct{}

func (geminiNativeTransport) APIMode() string { return geminiNativeAPIMode }

func (geminiNativeTransport) BuildRequest(req ChatRequest) (ProviderRequest, error) {
	descriptors := SanitizeToolDescriptors(req.Tools)
	body, err := buildGeminiCloudCodeRequestBody(req, descriptors)
	if err != nil {
		return ProviderRequest{}, err
	}
	body, err = removeGeminiNativeRequestModel(body)
	if err != nil {
		return ProviderRequest{}, err
	}
	return ProviderRequest{
		APIMode:         geminiNativeAPIMode,
		Path:            "/models/" + req.Model + ":streamGenerateContent?alt=sse",
		Body:            body,
		ToolDescriptors: descriptors,
	}, nil
}

func (geminiNativeTransport) OpenFixtureStream(body io.ReadCloser, req ProviderRequest) (Stream, error) {
	return geminiCloudCodeTransport{}.OpenFixtureStream(body, req)
}

func removeGeminiNativeRequestModel(body []byte) ([]byte, error) {
	var payload geminiRequest
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	payload.Model = ""
	return json.Marshal(payload)
}

func geminiNativeProviderStatus() ProviderStatus {
	status := openAICompatibleProviderStatus("gemini", "https://generativelanguage.googleapis.com/v1beta")
	status.Provider = "gemini"
	status.Runtime = geminiNativeAPIMode
	return status
}
