package responses

// Object is the OpenAI Responses-compatible response envelope.
type Object struct {
	ID        string       `json:"id"`
	Object    string       `json:"object"`
	Status    string       `json:"status"`
	CreatedAt int64        `json:"created_at"`
	Model     string       `json:"model"`
	Output    []OutputItem `json:"output"`
	Usage     Usage        `json:"usage"`
}

type OutputItem struct {
	Type      string        `json:"type"`
	ID        string        `json:"id,omitempty"`
	Role      string        `json:"role,omitempty"`
	Content   []ContentPart `json:"content,omitempty"`
	CallID    string        `json:"call_id,omitempty"`
	Name      string        `json:"name,omitempty"`
	Arguments string        `json:"arguments,omitempty"`
	Output    string        `json:"output,omitempty"`
}

type ContentPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}
