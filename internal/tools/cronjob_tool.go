package tools

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	rc "github.com/robfig/cron/v3"
)

const CronjobToolName = "cronjob"

// CronjobToolConfig wires the cronjob public tool to the native cron Store.
// Store is intentionally accepted as any to avoid an import cycle: internal/cron
// imports internal/kernel, and internal/kernel imports internal/tools.
type CronjobToolConfig struct {
	Store             any
	ScriptsRoot       string
	Now               func() time.Time
	RunNowUnsupported bool
}

// CronjobTool is a store-only action adapter. It never starts the scheduler,
// submits to the kernel, calls providers, or executes scripts.
type CronjobTool struct {
	cfg CronjobToolConfig
}

// CronjobRunNowRequest is returned by action=run so the caller can decide
// whether and how to execute the job.
type CronjobRunNowRequest struct {
	Action     string `json:"action"`
	JobID      string `json:"job_id"`
	PromptHash string `json:"prompt_hash"`
}

func NewCronjobTool(cfg CronjobToolConfig) *CronjobTool {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &CronjobTool{cfg: cfg}
}

func (*CronjobTool) Name() string { return CronjobToolName }

func (*CronjobTool) Description() string {
	return "Manage scheduled cron jobs over the native cron store without starting the scheduler."
}

func (*CronjobTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"action":{"type":"string","description":"One of: create, list, update, pause, resume, remove, run"},"job_id":{"type":"string","description":"Required for update/pause/resume/remove/run"},"prompt":{"type":"string"},"schedule":{"type":"string","description":"For create/update: '30m', 'every 2h', '0 9 * * *', or ISO timestamp"},"name":{"type":"string"},"repeat":{"type":"integer"},"skills":{"type":"array","items":{"type":"string"}},"model":{"type":"object","properties":{"provider":{"type":"string"},"model":{"type":"string"}}},"script":{"type":"string"},"context_from":{"oneOf":[{"type":"array","items":{"type":"string"}},{"type":"string"}]},"enabled_toolsets":{"type":"array","items":{"type":"string"}},"workdir":{"type":"string"}},"required":["action"]}`)
}

func (*CronjobTool) Timeout() time.Duration { return 0 }

func (t *CronjobTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	_ = ctx
	var in cronjobArgs
	if err := json.Unmarshal(args, &in); err != nil {
		return cronjobError("invalid cronjob args: " + err.Error()), nil
	}

	action := strings.ToLower(strings.TrimSpace(in.Action))
	if action == "" {
		return cronjobError("action is required"), nil
	}
	if action == "run_now" || action == "trigger" {
		action = "run"
	}
	if !validCronjobAction(action) {
		return cronjobError(fmt.Sprintf("unknown cron action %q", in.Action)), nil
	}
	if action == "run" && t.cfg.RunNowUnsupported {
		return cronjobError("run-now unsupported by this cronjob tool adapter"), nil
	}

	store, err := newReflectedCronStore(t.cfg.Store)
	if err != nil {
		return cronjobError(err.Error()), nil
	}

	switch action {
	case "create":
		return t.create(store, in), nil
	case "list":
		return t.list(store), nil
	case "update":
		return t.update(store, in), nil
	case "pause":
		return t.pauseResume(store, in, true), nil
	case "resume":
		return t.pauseResume(store, in, false), nil
	case "remove":
		return t.remove(store, in), nil
	case "run":
		return t.runNow(store, in), nil
	default:
		return cronjobError(fmt.Sprintf("unknown cron action %q", in.Action)), nil
	}
}

type cronjobArgs struct {
	Action          string                `json:"action"`
	JobID           string                `json:"job_id"`
	Name            *string               `json:"name"`
	Prompt          *string               `json:"prompt"`
	Schedule        *string               `json:"schedule"`
	Repeat          *int                  `json:"repeat"`
	Model           *cronjobModelArg      `json:"model"`
	Skills          *[]string             `json:"skills"`
	EnabledToolsets *[]string             `json:"enabled_toolsets"`
	Workdir         *string               `json:"workdir"`
	Script          *string               `json:"script"`
	ContextFrom     *cronjobStringListArg `json:"context_from"`
}

type cronjobModelArg struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

type cronjobStringListArg struct {
	Values []string
}

func (a *cronjobStringListArg) UnmarshalJSON(raw []byte) error {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		if strings.TrimSpace(text) == "" {
			a.Values = nil
			return nil
		}
		a.Values = []string{strings.TrimSpace(text)}
		return nil
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		return err
	}
	a.Values = normalizeCronjobStrings(values)
	return nil
}

type cronjobRecord struct {
	ID              string
	Name            string
	Schedule        string
	Prompt          string
	Paused          bool
	CreatedAt       int64
	LastRunUnix     int64
	LastStatus      string
	Repeat          int
	RepeatCompleted int
	Model           string
	Provider        string
	Skills          []string
	EnabledToolsets []string
	Workdir         string
	Script          string
	ContextFrom     []string
}

type cronjobEntry struct {
	value  reflect.Value
	record cronjobRecord
}

func (t *CronjobTool) create(store *reflectedCronStore, in cronjobArgs) json.RawMessage {
	if in.Schedule == nil || strings.TrimSpace(*in.Schedule) == "" {
		return cronjobError("schedule is required for create")
	}
	parsed, err := parseCronjobSchedule(*in.Schedule, t.cfg.Now())
	if err != nil {
		return cronjobError(err.Error())
	}
	prompt := stringValue(in.Prompt)
	if strings.TrimSpace(prompt) == "" {
		return cronjobError("prompt is required for create")
	}
	if finding, blocked := scanCronjobPrompt(prompt); blocked {
		return cronjobError(finding)
	}

	script, errRaw := t.validateScriptValue(in.Script)
	if errRaw != nil {
		return errRaw
	}
	workdir, errRaw := validateCronjobWorkdir(in.Workdir)
	if errRaw != nil {
		return errRaw
	}
	contextFrom, errRaw := validateCronjobContextFrom(store, in.ContextFrom)
	if errRaw != nil {
		return errRaw
	}

	repeat := parsed.DefaultRepeat
	if in.Repeat != nil {
		repeat = normalizeCronjobRepeat(*in.Repeat)
	}
	id := newCronjobID()
	name := strings.TrimSpace(stringValue(in.Name))
	if name == "" {
		name = "cron " + id[:8]
	}

	rec := cronjobRecord{
		ID:              id,
		Name:            name,
		Schedule:        parsed.Display,
		Prompt:          prompt,
		CreatedAt:       t.cfg.Now().Unix(),
		Repeat:          repeat,
		Model:           strings.TrimSpace(modelValue(in.Model)),
		Provider:        strings.TrimSpace(providerValue(in.Model)),
		Skills:          stringSliceValue(in.Skills),
		EnabledToolsets: stringSliceValue(in.EnabledToolsets),
		Workdir:         workdir,
		Script:          script,
		ContextFrom:     contextFrom,
	}
	if err := store.Create(rec); err != nil {
		return cronjobError(err.Error())
	}
	return cronjobJSON(map[string]any{
		"success":  true,
		"job_id":   rec.ID,
		"name":     rec.Name,
		"schedule": rec.Schedule,
		"repeat":   repeatDisplay(rec),
		"job":      summarizeCronjob(rec),
	})
}

func (t *CronjobTool) list(store *reflectedCronStore) json.RawMessage {
	entries, err := store.List()
	if err != nil {
		return cronjobError(err.Error())
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].record.CreatedAt != entries[j].record.CreatedAt {
			return entries[i].record.CreatedAt < entries[j].record.CreatedAt
		}
		if entries[i].record.Name != entries[j].record.Name {
			return entries[i].record.Name < entries[j].record.Name
		}
		return entries[i].record.ID < entries[j].record.ID
	})
	jobs := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		jobs = append(jobs, summarizeCronjob(entry.record))
	}
	return cronjobJSON(map[string]any{
		"success": true,
		"count":   len(jobs),
		"jobs":    jobs,
	})
}

func (t *CronjobTool) update(store *reflectedCronStore, in cronjobArgs) json.RawMessage {
	id := strings.TrimSpace(in.JobID)
	if id == "" {
		return cronjobError("job_id is required for action 'update'")
	}
	entry, errRaw := getCronjobEntry(store, id)
	if errRaw != nil {
		return errRaw
	}
	changed := false

	if in.Prompt != nil {
		prompt := *in.Prompt
		if finding, blocked := scanCronjobPrompt(prompt); blocked {
			return cronjobError(finding)
		}
		setStringField(entry.value, "Prompt", prompt)
		entry.record.Prompt = prompt
		changed = true
	}
	if in.Schedule != nil {
		parsed, err := parseCronjobSchedule(*in.Schedule, t.cfg.Now())
		if err != nil {
			return cronjobError(err.Error())
		}
		setStringField(entry.value, "Schedule", parsed.Display)
		entry.record.Schedule = parsed.Display
		changed = true
	}
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" {
			return cronjobError("name must not be empty")
		}
		if err := ensureCronjobNameAvailable(store, id, name); err != nil {
			return cronjobError(err.Error())
		}
		setStringField(entry.value, "Name", name)
		entry.record.Name = name
		changed = true
	}
	if in.Repeat != nil {
		repeat := normalizeCronjobRepeat(*in.Repeat)
		setIntField(entry.value, "Repeat", repeat)
		entry.record.Repeat = repeat
		changed = true
	}
	if in.Model != nil {
		model := strings.TrimSpace(in.Model.Model)
		provider := strings.TrimSpace(in.Model.Provider)
		setStringField(entry.value, "Model", model)
		entry.record.Model = model
		if provider != "" {
			setStringField(entry.value, "Provider", provider)
			entry.record.Provider = provider
		}
		changed = true
	}
	if in.Skills != nil {
		skills := stringSliceValue(in.Skills)
		setStringSliceField(entry.value, "Skills", skills)
		entry.record.Skills = skills
		changed = true
	}
	if in.EnabledToolsets != nil {
		toolsets := stringSliceValue(in.EnabledToolsets)
		setStringSliceField(entry.value, "EnabledToolsets", toolsets)
		entry.record.EnabledToolsets = toolsets
		changed = true
	}
	if in.Workdir != nil {
		workdir, errRaw := validateCronjobWorkdir(in.Workdir)
		if errRaw != nil {
			return errRaw
		}
		setStringField(entry.value, "Workdir", workdir)
		entry.record.Workdir = workdir
		changed = true
	}
	if in.Script != nil {
		script, errRaw := t.validateScriptValue(in.Script)
		if errRaw != nil {
			return errRaw
		}
		setStringField(entry.value, "Script", script)
		entry.record.Script = script
		changed = true
	}
	if in.ContextFrom != nil {
		contextFrom, errRaw := validateCronjobContextFrom(store, in.ContextFrom)
		if errRaw != nil {
			return errRaw
		}
		setStringSliceField(entry.value, "ContextFrom", contextFrom)
		entry.record.ContextFrom = contextFrom
		changed = true
	}
	if !changed {
		return cronjobError("no updates provided")
	}
	if err := store.Update(entry.value); err != nil {
		return cronjobError(err.Error())
	}
	return cronjobJSON(map[string]any{
		"success": true,
		"job":     summarizeCronjob(entry.record),
	})
}

func (t *CronjobTool) pauseResume(store *reflectedCronStore, in cronjobArgs, paused bool) json.RawMessage {
	action := "resume"
	if paused {
		action = "pause"
	}
	id := strings.TrimSpace(in.JobID)
	if id == "" {
		return cronjobError(fmt.Sprintf("job_id is required for action '%s'", action))
	}
	entry, errRaw := getCronjobEntry(store, id)
	if errRaw != nil {
		return errRaw
	}
	setBoolField(entry.value, "Paused", paused)
	entry.record.Paused = paused
	if err := store.Update(entry.value); err != nil {
		return cronjobError(err.Error())
	}
	return cronjobJSON(map[string]any{
		"success": true,
		"job":     summarizeCronjob(entry.record),
	})
}

func (t *CronjobTool) remove(store *reflectedCronStore, in cronjobArgs) json.RawMessage {
	_ = t
	id := strings.TrimSpace(in.JobID)
	if id == "" {
		return cronjobError("job_id is required for action 'remove'")
	}
	entry, errRaw := getCronjobEntry(store, id)
	if errRaw != nil {
		return errRaw
	}
	if err := store.Delete(id); err != nil {
		return cronjobError(err.Error())
	}
	return cronjobJSON(map[string]any{
		"success":     true,
		"removed_job": summarizeCronjob(entry.record),
	})
}

func (t *CronjobTool) runNow(store *reflectedCronStore, in cronjobArgs) json.RawMessage {
	id := strings.TrimSpace(in.JobID)
	if id == "" {
		return cronjobError("job_id is required for action 'run'")
	}
	entry, errRaw := getCronjobEntry(store, id)
	if errRaw != nil {
		return errRaw
	}
	req := CronjobRunNowRequest{
		Action:     "run_now",
		JobID:      entry.record.ID,
		PromptHash: cronjobPromptHash(entry.record.Prompt),
	}
	return cronjobJSON(map[string]any{
		"success": true,
		"run_now": req,
	})
}

func (t *CronjobTool) validateScriptValue(value *string) (string, json.RawMessage) {
	if value == nil {
		return "", nil
	}
	raw := strings.TrimSpace(*value)
	if raw == "" {
		return "", nil
	}
	clean, finding, blocked := validateCronjobScriptPath(raw, t.cfg.ScriptsRoot)
	if blocked {
		return "", cronjobError(finding)
	}
	return clean, nil
}

func getCronjobEntry(store *reflectedCronStore, id string) (cronjobEntry, json.RawMessage) {
	entry, err := store.Get(id)
	if err != nil {
		return cronjobEntry{}, cronjobError(fmt.Sprintf("job with ID %q not found", id))
	}
	return entry, nil
}

func validateCronjobContextFrom(store *reflectedCronStore, arg *cronjobStringListArg) ([]string, json.RawMessage) {
	if arg == nil {
		return nil, nil
	}
	values := normalizeCronjobStrings(arg.Values)
	for _, id := range values {
		if _, err := store.Get(id); err != nil {
			return nil, cronjobError(fmt.Sprintf("context_from job %q not found", id))
		}
	}
	return values, nil
}

func validateCronjobWorkdir(value *string) (string, json.RawMessage) {
	if value == nil {
		return "", nil
	}
	raw := strings.TrimSpace(*value)
	if raw == "" {
		return "", nil
	}
	if !filepath.IsAbs(raw) {
		return "", cronjobError("workdir must be an absolute path")
	}
	info, err := os.Stat(raw)
	if err != nil {
		return "", cronjobError("workdir does not exist: " + raw)
	}
	if !info.IsDir() {
		return "", cronjobError("workdir is not a directory: " + raw)
	}
	return filepath.Clean(raw), nil
}

func ensureCronjobNameAvailable(store *reflectedCronStore, currentID, name string) error {
	entries, err := store.List()
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.record.ID != currentID && entry.record.Name == name {
			return errors.New("cron: job name already taken")
		}
	}
	return nil
}

func summarizeCronjob(rec cronjobRecord) map[string]any {
	out := map[string]any{
		"job_id":      rec.ID,
		"name":        rec.Name,
		"schedule":    rec.Schedule,
		"repeat":      repeatDisplay(rec),
		"enabled":     !rec.Paused,
		"state":       cronjobState(rec),
		"prompt_hash": cronjobPromptHash(rec.Prompt),
	}
	if rec.Model != "" {
		out["model"] = rec.Model
	}
	if rec.Provider != "" {
		out["provider"] = rec.Provider
	}
	if len(rec.Skills) > 0 {
		out["skills"] = append([]string(nil), rec.Skills...)
	}
	if len(rec.EnabledToolsets) > 0 {
		out["enabled_toolsets"] = append([]string(nil), rec.EnabledToolsets...)
	}
	if rec.Workdir != "" {
		out["workdir"] = rec.Workdir
	}
	if rec.Script != "" {
		out["script"] = rec.Script
	}
	if len(rec.ContextFrom) > 0 {
		out["context_from"] = append([]string(nil), rec.ContextFrom...)
	}
	if rec.LastRunUnix > 0 {
		out["last_run_unix"] = rec.LastRunUnix
	}
	if rec.LastStatus != "" {
		out["last_status"] = rec.LastStatus
	}
	return out
}

func cronjobState(rec cronjobRecord) string {
	if rec.Paused {
		return "paused"
	}
	return "scheduled"
}

func repeatDisplay(rec cronjobRecord) string {
	switch {
	case rec.Repeat <= 0:
		return "forever"
	case rec.Repeat == 1 && rec.RepeatCompleted == 0:
		return "once"
	case rec.Repeat == 1:
		return "1/1"
	case rec.RepeatCompleted > 0:
		return fmt.Sprintf("%d/%d", rec.RepeatCompleted, rec.Repeat)
	default:
		return fmt.Sprintf("%d times", rec.Repeat)
	}
}

func cronjobJSON(v any) json.RawMessage {
	raw, err := json.Marshal(v)
	if err != nil {
		return cronjobError(err.Error())
	}
	return raw
}

func cronjobError(message string) json.RawMessage {
	raw, _ := json.Marshal(map[string]any{
		"success": false,
		"error":   message,
	})
	return raw
}

func validCronjobAction(action string) bool {
	switch action {
	case "create", "list", "update", "pause", "resume", "remove", "run":
		return true
	default:
		return false
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func stringSliceValue(value *[]string) []string {
	if value == nil {
		return nil
	}
	return normalizeCronjobStrings(*value)
}

func modelValue(value *cronjobModelArg) string {
	if value == nil {
		return ""
	}
	return value.Model
}

func providerValue(value *cronjobModelArg) string {
	if value == nil {
		return ""
	}
	return value.Provider
}

func normalizeCronjobStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		text := strings.TrimSpace(value)
		if text == "" {
			continue
		}
		if _, ok := seen[text]; ok {
			continue
		}
		seen[text] = struct{}{}
		out = append(out, text)
	}
	return out
}

func normalizeCronjobRepeat(value int) int {
	if value <= 0 {
		return 0
	}
	return value
}

func newCronjobID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		now := time.Now().UnixNano()
		return fmt.Sprintf("%032x", now)
	}
	return hex.EncodeToString(b[:])
}

func cronjobPromptHash(prompt string) string {
	h := sha256.Sum256([]byte(prompt))
	return hex.EncodeToString(h[:8])
}

type cronjobParsedSchedule struct {
	Display       string
	DefaultRepeat int
}

var (
	cronjobDurationPattern = regexp.MustCompile(`^(\d+)\s*(m|min|mins|minute|minutes|h|hr|hrs|hour|hours|d|day|days)$`)
	cronjobISODatePattern  = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}(?:[T ].*)?$`)
	cronjobCronParser      = rc.NewParser(rc.Minute | rc.Hour | rc.Dom | rc.Month | rc.Dow)
)

func parseCronjobSchedule(input string, now time.Time) (cronjobParsedSchedule, error) {
	display := strings.TrimSpace(input)
	if display == "" {
		return cronjobParsedSchedule{}, errors.New("invalid schedule: schedule is empty")
	}
	lower := strings.ToLower(display)
	if strings.HasPrefix(lower, "every ") {
		if _, err := parseCronjobDurationMinutes(strings.TrimSpace(display[len("every "):])); err != nil {
			return cronjobParsedSchedule{}, fmt.Errorf("invalid schedule: %s", err)
		}
		return cronjobParsedSchedule{Display: display}, nil
	}
	if cronjobISODatePattern.MatchString(display) {
		if err := validateCronjobISO(display, now.Location()); err != nil {
			return cronjobParsedSchedule{}, fmt.Errorf("invalid schedule: %s", err)
		}
		return cronjobParsedSchedule{Display: display, DefaultRepeat: 1}, nil
	}
	if _, err := parseCronjobDurationMinutes(display); err == nil {
		return cronjobParsedSchedule{Display: display, DefaultRepeat: 1}, nil
	}
	if len(strings.Fields(display)) == 5 {
		if _, err := cronjobCronParser.Parse(display); err != nil {
			return cronjobParsedSchedule{}, fmt.Errorf("invalid schedule: %s", err)
		}
		return cronjobParsedSchedule{Display: display}, nil
	}
	return cronjobParsedSchedule{}, errors.New("invalid schedule: use a duration, recurring interval, 5-field cron expression, or ISO timestamp")
}

func parseCronjobDurationMinutes(input string) (int, error) {
	match := cronjobDurationPattern.FindStringSubmatch(strings.ToLower(strings.TrimSpace(input)))
	if match == nil {
		return 0, fmt.Errorf("invalid duration %q", input)
	}
	value, err := strconv.Atoi(match[1])
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("invalid duration %q", input)
	}
	switch match[2][0] {
	case 'm':
		return value, nil
	case 'h':
		return value * 60, nil
	case 'd':
		return value * 1440, nil
	default:
		return 0, fmt.Errorf("invalid duration %q", input)
	}
}

func validateCronjobISO(input string, loc *time.Location) error {
	if loc == nil {
		loc = time.UTC
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04Z07:00"} {
		if _, err := time.Parse(layout, input); err == nil {
			return nil
		}
	}
	for _, layout := range []string{
		"2006-01-02T15:04:05.999999999",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
	} {
		if _, err := time.ParseInLocation(layout, input, loc); err == nil {
			return nil
		}
	}
	return fmt.Errorf("invalid ISO timestamp %q", input)
}

type reflectedCronStore struct {
	value   reflect.Value
	jobType reflect.Type
}

func newReflectedCronStore(store any) (*reflectedCronStore, error) {
	if store == nil {
		return nil, errors.New("cron store disabled")
	}
	value := reflect.ValueOf(store)
	if value.Kind() == reflect.Pointer && value.IsNil() {
		return nil, errors.New("cron store disabled")
	}
	create := value.MethodByName("Create")
	if !create.IsValid() || create.Type().NumIn() != 1 {
		return nil, errors.New("cron store disabled: missing Create")
	}
	jobType := create.Type().In(0)
	for _, method := range []string{"Get", "List", "Update", "Delete"} {
		if !value.MethodByName(method).IsValid() {
			return nil, fmt.Errorf("cron store disabled: missing %s", method)
		}
	}
	return &reflectedCronStore{value: value, jobType: jobType}, nil
}

func (s *reflectedCronStore) Create(rec cronjobRecord) error {
	job := reflect.New(s.jobType).Elem()
	applyCronjobRecord(job, rec)
	return reflectedError(s.value.MethodByName("Create").Call([]reflect.Value{job}))
}

func (s *reflectedCronStore) Get(id string) (cronjobEntry, error) {
	out := s.value.MethodByName("Get").Call([]reflect.Value{reflect.ValueOf(id)})
	if err := reflectedError(out); err != nil {
		return cronjobEntry{}, err
	}
	job := reflect.New(s.jobType).Elem()
	job.Set(out[0])
	return cronjobEntry{value: job, record: cronjobRecordFromValue(job)}, nil
}

func (s *reflectedCronStore) List() ([]cronjobEntry, error) {
	out := s.value.MethodByName("List").Call(nil)
	if err := reflectedError(out); err != nil {
		return nil, err
	}
	slice := out[0]
	entries := make([]cronjobEntry, 0, slice.Len())
	for i := 0; i < slice.Len(); i++ {
		job := reflect.New(s.jobType).Elem()
		job.Set(slice.Index(i))
		entries = append(entries, cronjobEntry{value: job, record: cronjobRecordFromValue(job)})
	}
	return entries, nil
}

func (s *reflectedCronStore) Update(job reflect.Value) error {
	return reflectedError(s.value.MethodByName("Update").Call([]reflect.Value{job}))
}

func (s *reflectedCronStore) Delete(id string) error {
	return reflectedError(s.value.MethodByName("Delete").Call([]reflect.Value{reflect.ValueOf(id)}))
}

func reflectedError(out []reflect.Value) error {
	if len(out) == 0 {
		return nil
	}
	last := out[len(out)-1]
	if !last.IsValid() || last.IsNil() {
		return nil
	}
	err, ok := last.Interface().(error)
	if !ok {
		return nil
	}
	return err
}

func applyCronjobRecord(value reflect.Value, rec cronjobRecord) {
	setStringField(value, "ID", rec.ID)
	setStringField(value, "Name", rec.Name)
	setStringField(value, "Schedule", rec.Schedule)
	setStringField(value, "Prompt", rec.Prompt)
	setBoolField(value, "Paused", rec.Paused)
	setInt64Field(value, "CreatedAt", rec.CreatedAt)
	setInt64Field(value, "LastRunUnix", rec.LastRunUnix)
	setStringField(value, "LastStatus", rec.LastStatus)
	setIntField(value, "Repeat", rec.Repeat)
	setIntField(value, "RepeatCompleted", rec.RepeatCompleted)
	setStringField(value, "Model", rec.Model)
	setStringField(value, "Provider", rec.Provider)
	setStringSliceField(value, "Skills", rec.Skills)
	setStringSliceField(value, "EnabledToolsets", rec.EnabledToolsets)
	setStringField(value, "Workdir", rec.Workdir)
	setStringField(value, "Script", rec.Script)
	setStringSliceField(value, "ContextFrom", rec.ContextFrom)
}

func cronjobRecordFromValue(value reflect.Value) cronjobRecord {
	return cronjobRecord{
		ID:              stringField(value, "ID"),
		Name:            stringField(value, "Name"),
		Schedule:        stringField(value, "Schedule"),
		Prompt:          stringField(value, "Prompt"),
		Paused:          boolField(value, "Paused"),
		CreatedAt:       int64Field(value, "CreatedAt"),
		LastRunUnix:     int64Field(value, "LastRunUnix"),
		LastStatus:      stringField(value, "LastStatus"),
		Repeat:          intField(value, "Repeat"),
		RepeatCompleted: intField(value, "RepeatCompleted"),
		Model:           stringField(value, "Model"),
		Provider:        stringField(value, "Provider"),
		Skills:          stringSliceField(value, "Skills"),
		EnabledToolsets: stringSliceField(value, "EnabledToolsets"),
		Workdir:         stringField(value, "Workdir"),
		Script:          stringField(value, "Script"),
		ContextFrom:     stringSliceField(value, "ContextFrom"),
	}
}

func setStringField(value reflect.Value, name, fieldValue string) {
	field := value.FieldByName(name)
	if field.IsValid() && field.CanSet() && field.Kind() == reflect.String {
		field.SetString(fieldValue)
	}
}

func setBoolField(value reflect.Value, name string, fieldValue bool) {
	field := value.FieldByName(name)
	if field.IsValid() && field.CanSet() && field.Kind() == reflect.Bool {
		field.SetBool(fieldValue)
	}
}

func setIntField(value reflect.Value, name string, fieldValue int) {
	field := value.FieldByName(name)
	if field.IsValid() && field.CanSet() {
		switch field.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			field.SetInt(int64(fieldValue))
		}
	}
}

func setInt64Field(value reflect.Value, name string, fieldValue int64) {
	field := value.FieldByName(name)
	if field.IsValid() && field.CanSet() {
		switch field.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			field.SetInt(fieldValue)
		}
	}
}

func setStringSliceField(value reflect.Value, name string, fieldValue []string) {
	field := value.FieldByName(name)
	if !field.IsValid() || !field.CanSet() || field.Kind() != reflect.Slice || field.Type().Elem().Kind() != reflect.String {
		return
	}
	slice := reflect.MakeSlice(field.Type(), 0, len(fieldValue))
	for _, item := range fieldValue {
		slice = reflect.Append(slice, reflect.ValueOf(item).Convert(field.Type().Elem()))
	}
	field.Set(slice)
}

func stringField(value reflect.Value, name string) string {
	field := value.FieldByName(name)
	if field.IsValid() && field.Kind() == reflect.String {
		return field.String()
	}
	return ""
}

func boolField(value reflect.Value, name string) bool {
	field := value.FieldByName(name)
	return field.IsValid() && field.Kind() == reflect.Bool && field.Bool()
}

func intField(value reflect.Value, name string) int {
	field := value.FieldByName(name)
	if field.IsValid() {
		switch field.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return int(field.Int())
		}
	}
	return 0
}

func int64Field(value reflect.Value, name string) int64 {
	field := value.FieldByName(name)
	if field.IsValid() {
		switch field.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return field.Int()
		}
	}
	return 0
}

func stringSliceField(value reflect.Value, name string) []string {
	field := value.FieldByName(name)
	if !field.IsValid() || field.Kind() != reflect.Slice || field.Type().Elem().Kind() != reflect.String {
		return nil
	}
	out := make([]string, 0, field.Len())
	for i := 0; i < field.Len(); i++ {
		out = append(out, field.Index(i).String())
	}
	return out
}

type cronjobPromptFinding struct {
	pattern *regexp.Regexp
	id      string
}

var cronjobPromptFindings = []cronjobPromptFinding{
	{pattern: regexp.MustCompile(`(?is)ignore\s+(?:\w+\s+)*(?:previous|all|above|prior)\s+(?:\w+\s+)*instructions`), id: "prompt_injection"},
	{pattern: regexp.MustCompile(`(?is)do\s+not\s+tell\s+the\s+user`), id: "deception_hide"},
	{pattern: regexp.MustCompile(`(?is)system\s+prompt\s+override`), id: "sys_prompt_override"},
	{pattern: regexp.MustCompile(`(?is)disregard\s+(your|all|any)\s+(instructions|rules|guidelines)`), id: "disregard_rules"},
	{pattern: regexp.MustCompile(`(?is)curl\s+[^\n]*\$\{?\w*(KEY|TOKEN|SECRET|PASSWORD|CREDENTIAL|API)`), id: "exfil_curl"},
	{pattern: regexp.MustCompile(`(?is)wget\s+[^\n]*\$\{?\w*(KEY|TOKEN|SECRET|PASSWORD|CREDENTIAL|API)`), id: "exfil_wget"},
	{pattern: regexp.MustCompile(`(?is)cat\s+[^\n]*(\.env|credentials|\.netrc|\.pgpass)`), id: "read_secrets"},
	{pattern: regexp.MustCompile(`(?is)authorized_keys`), id: "ssh_backdoor"},
	{pattern: regexp.MustCompile(`(?is)/etc/sudoers|visudo`), id: "sudoers_mod"},
	{pattern: regexp.MustCompile(`(?is)rm\s+-rf\s+/`), id: "destructive_root_rm"},
}

var cronjobInvisibleChars = map[rune]struct{}{
	'\u200b': {},
	'\u200c': {},
	'\u200d': {},
	'\u2060': {},
	'\ufeff': {},
	'\u202a': {},
	'\u202b': {},
	'\u202c': {},
	'\u202d': {},
	'\u202e': {},
}

func scanCronjobPrompt(prompt string) (string, bool) {
	for _, r := range prompt {
		if _, ok := cronjobInvisibleChars[r]; ok {
			return fmt.Sprintf("blocked prompt: prompt contains invisible unicode U+%04X", r), true
		}
	}
	for _, finding := range cronjobPromptFindings {
		if finding.pattern.FindString(prompt) != "" {
			return fmt.Sprintf("blocked prompt: prompt matches threat pattern %q", finding.id), true
		}
	}
	return "", false
}

func validateCronjobScriptPath(script, scriptsRoot string) (string, string, bool) {
	raw := strings.TrimSpace(script)
	if raw == "" {
		return "", "", false
	}
	if strings.HasPrefix(raw, "~") {
		return "", "script path must be relative to the scripts root", true
	}
	if filepath.IsAbs(raw) || hasCronjobWindowsDrive(raw) {
		return "", "script path must not be absolute", true
	}
	clean := filepath.Clean(raw)
	if clean == "." {
		return "", "script path must name a file under the scripts root", true
	}
	if scriptsRoot == "" {
		scriptsRoot = "."
	}
	rootAbs, err := filepath.Abs(scriptsRoot)
	if err != nil {
		return "", "scripts root could not be resolved", true
	}
	targetAbs, err := filepath.Abs(filepath.Join(rootAbs, clean))
	if err != nil {
		return "", "script path could not be resolved under the scripts root", true
	}
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", "script path escapes the scripts root", true
	}
	return filepath.ToSlash(rel), "", false
}

func hasCronjobWindowsDrive(path string) bool {
	return len(path) >= 2 && path[1] == ':'
}
