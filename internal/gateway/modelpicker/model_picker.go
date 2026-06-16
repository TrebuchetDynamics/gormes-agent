package modelpicker

import (
	"context"
	"strconv"
	"strings"
	"sync"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

type SessionModelOverride struct {
	Model    string
	Provider string
}

func (o SessionModelOverride) IsZero() bool { return o.Model == "" && o.Provider == "" }

type modelPickerState struct {
	stage         string
	messageID     int
	pendingSlug   string
	pendingModels []string
}

type modelPickerManager struct {
	mu     sync.Mutex
	byChat map[string]modelPickerState
}

func (m *modelPickerManager) get(chatID string) modelPickerState {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.byChat[chatID]
}

func (m *modelPickerManager) set(chatID string, s modelPickerState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.byChat == nil {
		m.byChat = map[string]modelPickerState{}
	}
	m.byChat[chatID] = s
}

func (m *modelPickerManager) remove(chatID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.byChat, chatID)
}

type ModelPickerRequest struct {
	ChatID string
}

type ModelPickerCallback struct {
	ChatID    string
	Prefix    string
	Value     string
	MessageID int
}

type ModelPickerResponse struct {
	Text     string
	Provider string
	Model    string
	Finished bool
	Changed  bool
}

type ModelPickerResolver interface {
	OpenModelPicker(ctx context.Context, req ModelPickerRequest) (ModelPickerResponse, error)
	HandleModelPickerCallback(ctx context.Context, cb ModelPickerCallback) (ModelPickerResponse, error)
	PickerProviders() []string
	PickerModels(providerSlug string) []string
}

type ResolverImpl struct {
	pickerState *modelPickerManager
	override    *SessionModelOverride
}

func NewModelPickerResolver(ov *SessionModelOverride) ModelPickerResolver {
	if ov == nil {
		ov = &SessionModelOverride{}
	}
	return &ResolverImpl{
		pickerState: &modelPickerManager{},
		override:    ov,
	}
}

func (r *ResolverImpl) stateManager() *modelPickerManager {
	if r.pickerState == nil {
		r.pickerState = &modelPickerManager{}
	}
	return r.pickerState
}

func (r *ResolverImpl) PickerProviders() []string {
	providers := llm.ListPickerProviders()
	out := make([]string, len(providers))
	for i, p := range providers {
		out[i] = p.Slug
	}
	return out
}

func (r *ResolverImpl) PickerModels(providerSlug string) []string {
	manifest := llm.HermesProviderRegistryManifest()
	for _, entry := range manifest {
		if entry.ID == providerSlug {
			if entry.ModelsDevID != "" {
				return []string{"default"}
			}
			return []string{"default"}
		}
	}
	return []string{"default"}
}

func (r *ResolverImpl) OpenModelPicker(ctx context.Context, req ModelPickerRequest) (ModelPickerResponse, error) {
	providers := llm.ListPickerProviders()
	list := make([]string, len(providers))
	slugs := make([]string, len(providers))
	for i, p := range providers {
		list[i] = p.Label
		slugs[i] = p.Slug
	}
	r.stateManager().set(req.ChatID, modelPickerState{
		stage:         "provider",
		pendingModels: slugs,
	})
	return ModelPickerResponse{
		Text:     "⚙ *Model Configuration*\n\n*Select a provider:*",
		Provider: "",
	}, nil
}

func (r *ResolverImpl) HandleModelPickerCallback(ctx context.Context, cb ModelPickerCallback) (ModelPickerResponse, error) {
	state := r.stateManager().get(cb.ChatID)
	switch cb.Prefix {
	case "mp":
		return r.handleProviderSelection(cb, state)
	case "mm":
		return r.handleModelSelection(cb, state)
	case "mb":
		return r.handleBack(cb, state)
	case "mx":
		r.stateManager().remove(cb.ChatID)
		return ModelPickerResponse{Text: "Selection cancelled.", Finished: true}, nil
	default:
		return ModelPickerResponse{Text: "Unknown picker action.", Finished: true}, nil
	}
}

func (r *ResolverImpl) handleProviderSelection(cb ModelPickerCallback, state modelPickerState) (ModelPickerResponse, error) {
	slug := cb.Value
	if state.stage != "provider" || !modelPickerValueAllowed(slug, state.pendingModels) {
		r.stateManager().remove(cb.ChatID)
		return ModelPickerResponse{Text: "Invalid provider selection.", Finished: true}, nil
	}
	providers := llm.ListPickerProviders()
	var label string
	for _, p := range providers {
		if p.Slug == slug {
			label = p.Label
			break
		}
	}
	if label == "" {
		r.stateManager().remove(cb.ChatID)
		return ModelPickerResponse{Text: "Invalid provider selection.", Finished: true}, nil
	}
	models := r.PickerModels(slug)
	r.stateManager().set(cb.ChatID, modelPickerState{
		stage:         "model",
		messageID:     cb.MessageID,
		pendingSlug:   slug,
		pendingModels: models,
	})
	text := "⚙ *Model Configuration*\n\nProvider: *" + label + "*\n\n*Select a model:*"
	return ModelPickerResponse{Text: text, Provider: slug}, nil
}

func (r *ResolverImpl) handleModelSelection(cb ModelPickerCallback, state modelPickerState) (ModelPickerResponse, error) {
	if state.stage != "model" || strings.TrimSpace(state.pendingSlug) == "" {
		r.stateManager().remove(cb.ChatID)
		return ModelPickerResponse{Text: "Invalid model selection.", Finished: true}, nil
	}
	if state.messageID != 0 && cb.MessageID != 0 && cb.MessageID != state.messageID {
		r.stateManager().remove(cb.ChatID)
		return ModelPickerResponse{Text: "Invalid model selection.", Finished: true}, nil
	}
	modelIdx := 0
	if cb.Value != "" {
		n, err := parseInt(cb.Value)
		if err != nil {
			r.stateManager().remove(cb.ChatID)
			return ModelPickerResponse{Text: "Invalid model selection.", Finished: true}, nil
		}
		modelIdx = n
	}
	models := state.pendingModels
	if modelIdx < 0 || modelIdx >= len(models) {
		r.stateManager().remove(cb.ChatID)
		return ModelPickerResponse{Text: "Invalid model selection.", Finished: true}, nil
	}
	model := models[modelIdx]
	if r.override != nil {
		r.override.Model = model
		r.override.Provider = state.pendingSlug
	}
	r.stateManager().remove(cb.ChatID)
	text := "⚙ *Model Configuration*\n\nModel set to `" + modelPickerCodeValue(model) + "`\nProvider: *" + modelPickerEmphasisValue(titleCase(state.pendingSlug)) + "*"
	return ModelPickerResponse{Text: text, Model: model, Provider: state.pendingSlug, Finished: true, Changed: true}, nil
}

func (r *ResolverImpl) handleBack(cb ModelPickerCallback, state modelPickerState) (ModelPickerResponse, error) {
	return r.OpenModelPicker(context.Background(), ModelPickerRequest{ChatID: cb.ChatID})
}

func modelPickerValueAllowed(value string, allowed []string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func modelPickerCodeValue(value string) string {
	replacer := strings.NewReplacer("`", "'", "\\", "/")
	return strings.Join(strings.Fields(replacer.Replace(strings.TrimSpace(value))), " ")
}

func modelPickerEmphasisValue(value string) string {
	replacer := strings.NewReplacer("*", "'", "_", " ", "`", "'", "\\", "/")
	return strings.Join(strings.Fields(replacer.Replace(strings.TrimSpace(value))), " ")
}

func parseInt(s string) (int, error) {
	if s == "" {
		return 0, errInvalidInt
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, errInvalidInt
		}
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, errInvalidInt
	}
	return n, nil
}

var errInvalidInt = &intError{"invalid integer"}

type intError struct{ msg string }

func (e *intError) Error() string { return e.msg }

func titleCase(s string) string {
	if s == "" {
		return s
	}
	out := make([]byte, len(s))
	upper := true
	for i, b := range []byte(s) {
		if b == '-' || b == '_' || b == ' ' {
			out[i] = ' '
			upper = true
		} else if upper && b >= 'a' && b <= 'z' {
			out[i] = b - 32
			upper = false
		} else {
			out[i] = b
			upper = false
		}
	}
	return string(out)
}
