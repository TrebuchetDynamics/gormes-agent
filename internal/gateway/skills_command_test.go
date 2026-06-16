package gateway

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
)

func TestHandleSkillsCommandSearchAndBrowseHubReadModel(t *testing.T) {
	providers := []skills.HubRegistryProvider{skills.NewInMemoryRegistryProvider([]skills.HubSearchResult{
		{
			Name:        "planner-pro",
			Description: "Plan safely with API_KEY=sk-live-secret and local path /home/xel/.gormes/secret.txt before editing.",
			Source:      "hermes-index",
			InstallID:   "github/openai/planner-pro",
			Score:       0.95,
			TrustLevel:  "community",
			Tags:        []string{"planning", "safe"},
		},
		{
			Name:        "browser-helper",
			Description: "Browse pages without mutating installed skills.",
			Source:      "skills.sh",
			InstallID:   "skills/browser-helper",
			Score:       0.50,
			TrustLevel:  "trusted",
		},
	}, nil)}
	opts := SkillsCommandOptions{HubProviders: providers, PageSize: 1, MaxDescriptionRunes: 90}

	search := HandleSkillsCommandWithOptions(context.Background(), "/skills search planner", opts)
	for _, want := range []string{
		"Skill Hub Search",
		"query: planner",
		"1 result(s)",
		"planner-pro",
		"source=hermes-index",
		"trust=community",
		"planning,safe",
	} {
		if !strings.Contains(search, want) {
			t.Fatalf("/skills search output missing %q:\n%s", want, search)
		}
	}
	for _, forbidden := range []string{"sk-live-secret", "/home/xel/.gormes", "secret.txt", "SKILL.md"} {
		if strings.Contains(search, forbidden) {
			t.Fatalf("/skills search leaked forbidden content %q:\n%s", forbidden, search)
		}
	}

	browse := HandleSkillsCommandWithOptions(context.Background(), "/skills browse --page 2 --page-size 1", opts)
	for _, want := range []string{
		"Skill Hub Browse",
		"page 2/2",
		"2 total",
		"browser-helper",
		"source=skills.sh",
		"trust=trusted",
	} {
		if !strings.Contains(browse, want) {
			t.Fatalf("/skills browse output missing %q:\n%s", want, browse)
		}
	}
	if strings.Contains(browse, "planner-pro") {
		t.Fatalf("/skills browse page 2 should not include page 1 result:\n%s", browse)
	}
}

func TestHandleSkillsCommandTelegramUsesPlainTextReply(t *testing.T) {
	kind, body := ParseInboundText("/skills search planner")
	if kind != EventSkills || body != "/skills search planner" {
		t.Fatalf("ParseInboundText(/skills search planner) = (%v, %q), want EventSkills with body", kind, body)
	}
	ch := newFakePlainSkillsChannel("telegram")
	fk := &fakeKernel{}
	m := NewManagerWithSubmitter(ManagerConfig{AllowedChats: map[string]string{"telegram": "42"}}, fk, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	if err := m.handleInbound(context.Background(), InboundEvent{
		Platform: "telegram",
		ChatID:   "42",
		UserID:   "u",
		MsgID:    "m-skills-search",
		Kind:     kind,
		Text:     body,
	}); err != nil {
		t.Fatalf("handleInbound(/skills search): %v", err)
	}

	plain := ch.plainSnapshot()
	if len(plain) != 1 {
		t.Fatalf("plain skills sends = %d, want 1", len(plain))
	}
	if !strings.Contains(strings.ToLower(plain[0].Text), "skill hub search") {
		t.Fatalf("plain skills reply missing search evidence:\n%s", plain[0].Text)
	}
	if sent := ch.sentSnapshot(); len(sent) != 0 {
		t.Fatalf("skills command used Markdown Send path, sent=%+v", sent)
	}
	if submits := fk.submitsSnapshot(); len(submits) != 0 {
		t.Fatalf("/skills command must not submit to kernel, got submits=%+v", submits)
	}
}

func TestHandleSkillsCommandMutatingActionsUnavailable(t *testing.T) {
	for _, action := range []string{"edit", "disable", "review"} {
		t.Run(action, func(t *testing.T) {
			out := HandleSkillsCommandWithOptions(context.Background(), "/skills "+action+" planner-pro", SkillsCommandOptions{
				HubProviders: []skills.HubRegistryProvider{skills.NewInMemoryRegistryProvider([]skills.HubSearchResult{{Name: "planner-pro", InstallID: "planner-pro"}}, nil)},
			})
			for _, want := range []string{"skills_manage_unavailable", action, "row 6.F", "read-only", "no skill store was changed"} {
				if !strings.Contains(out, want) {
					t.Fatalf("/skills %s output missing %q:\n%s", action, want, out)
				}
			}
		})
	}
}

func TestHandleSkillsCommandExternalDirs(t *testing.T) {
	root := t.TempDir()
	external := t.TempDir()
	writeGatewaySkill(t, root, "active/ops/shared", "shared", "Local shared", "Use local shared.")
	writeGatewaySkill(t, external, "research/external-skill", "external-skill", "External skill", "Use external safely.")
	writeGatewaySkill(t, external, "research/shared", "shared", "External shared", "Use external shared.")

	opts := SkillsCommandOptions{SkillsRoot: root, ExternalDirs: []string{external}}
	list := HandleSkillsCommandWithOptions(context.Background(), "/skills list --source external", opts)
	for _, want := range []string{"Installed Skills", "external-skill", "research", "external", "operator", "1 external"} {
		if !strings.Contains(list, want) {
			t.Fatalf("/skills list external missing %q:\n%s", want, list)
		}
	}
	for _, forbidden := range []string{"External shared", "shared  research", external} {
		if strings.Contains(list, forbidden) {
			t.Fatalf("/skills list external leaked/duplicated forbidden %q:\n%s", forbidden, list)
		}
	}

	inspect := HandleSkillsCommandWithOptions(context.Background(), "/skills inspect external-skill", opts)
	for _, want := range []string{"Skill: external-skill", "Description: External skill", "Use external safely.", "Source: external", "Trust: operator"} {
		if !strings.Contains(inspect, want) {
			t.Fatalf("/skills inspect external missing %q:\n%s", want, inspect)
		}
	}
	if strings.Contains(inspect, external) {
		t.Fatalf("/skills inspect leaked external root path:\n%s", inspect)
	}

	shared := HandleSkillsCommandWithOptions(context.Background(), "/skills inspect shared", opts)
	if !strings.Contains(shared, "Local shared") || strings.Contains(shared, "External shared") {
		t.Fatalf("local skill should win duplicate inspect, got:\n%s", shared)
	}
}

func TestHandleSkillsCommandInstallDirectURL(t *testing.T) {
	rawURL := "https://example.com/SKILL.md"
	fetcher := &fakeSkillsInstallFetcher{body: gatewaySkillURLDoc("")}
	scanner := &fakeSkillsInstallScanner{ok: true, evidence: "scan_clean"}
	store := newFakeSkillsInstallStore(t.TempDir())
	opts := SkillsCommandOptions{URLInstall: skills.URLInstallPolicy{Fetcher: fetcher, Scanner: scanner, Store: store}}

	out := HandleSkillsCommandWithOptions(context.Background(), "/skills install "+rawURL+" --name my-url-skill", opts)
	for _, want := range []string{"url_skill_installed", "installed my-url-skill"} {
		if !strings.Contains(out, want) {
			t.Fatalf("install output missing %q:\n%s", want, out)
		}
	}
	if len(fetcher.calls) != 1 || fetcher.calls[0] != rawURL {
		t.Fatalf("fetcher.calls = %#v, want one fetch for %s", fetcher.calls, rawURL)
	}
	if scanner.calls != 1 {
		t.Fatalf("scanner.calls = %d, want 1", scanner.calls)
	}
	if len(store.files) != 1 {
		t.Fatalf("store.files = %d, want 1", len(store.files))
	}
	for path := range store.files {
		if !strings.Contains(path, filepath.Join("active", "my-url-skill", "SKILL.md")) {
			t.Fatalf("written path = %q, want active/my-url-skill/SKILL.md", path)
		}
		if strings.Contains(out, store.root) || strings.Contains(out, path) {
			t.Fatalf("install output leaked local path %q:\n%s", path, out)
		}
	}

	missingName := HandleSkillsCommandWithOptions(context.Background(), "/skills install "+rawURL, opts)
	if !strings.Contains(missingName, "url_skill_missing_name") || !strings.Contains(missingName, "gormes skills install "+rawURL+" --name <your-name>") {
		t.Fatalf("missing-name output missing retry hint:\n%s", missingName)
	}
	if len(store.files) != 1 {
		t.Fatalf("missing-name path mutated store, files=%d", len(store.files))
	}

	unsafeFetcher := &fakeSkillsInstallFetcher{body: gatewaySkillURLDoc("")}
	unsafeStore := newFakeSkillsInstallStore(t.TempDir())
	unsafe := HandleSkillsCommandWithOptions(context.Background(), "/skills install "+rawURL+" --name SKILL", SkillsCommandOptions{URLInstall: skills.URLInstallPolicy{Fetcher: unsafeFetcher, Store: unsafeStore}})
	if !strings.Contains(unsafe, "url_skill_invalid_name") {
		t.Fatalf("unsafe-name output missing typed evidence:\n%s", unsafe)
	}
	if len(unsafeFetcher.calls) != 0 || len(unsafeStore.files) != 0 {
		t.Fatalf("unsafe-name mutated dependencies: fetches=%d files=%d", len(unsafeFetcher.calls), len(unsafeStore.files))
	}

	blockedStore := newFakeSkillsInstallStore(t.TempDir())
	blocked := HandleSkillsCommandWithOptions(context.Background(), "/skills install "+rawURL+" --name blocked-skill", SkillsCommandOptions{URLInstall: skills.URLInstallPolicy{
		Fetcher: &fakeSkillsInstallFetcher{body: gatewaySkillURLDoc("")},
		Scanner: &fakeSkillsInstallScanner{ok: false, evidence: "scan_blocked_secret"},
		Store:   blockedStore,
	}})
	if !strings.Contains(blocked, "url_skill_quarantine_failed") || !strings.Contains(blocked, "scan_blocked_secret") {
		t.Fatalf("blocked scan output missing evidence:\n%s", blocked)
	}
	if len(blockedStore.files) != 0 {
		t.Fatalf("blocked scan wrote %d files, want 0", len(blockedStore.files))
	}
}

func TestManagerSkillsCommandOptionsAreSnapshotted(t *testing.T) {
	root := t.TempDir()
	writeGatewaySkill(t, root, "active/ops/planner", "planner", "Plan work", "Body")
	disabled := map[string]struct{}{}
	ch := newFakeChannel("slack")
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"slack": "C42"},
		SkillsCommandOptions: SkillsCommandOptions{
			SkillsRoot: root,
			Disabled:   disabled,
		},
	}, &fakeKernel{}, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}
	disabled["planner"] = struct{}{}

	if err := m.handleInbound(context.Background(), InboundEvent{Platform: "slack", ChatID: "C42", UserID: "u", MsgID: "m-skills-list", Kind: EventSkills, Text: "/skills list"}); err != nil {
		t.Fatalf("handleInbound(/skills list): %v", err)
	}

	sent := ch.sentSnapshot()
	if len(sent) != 1 {
		t.Fatalf("skills list sends = %d, want 1", len(sent))
	}
	if strings.Contains(sent[0].Text, "planner  -  local  trusted  disabled") || strings.Contains(sent[0].Text, "0 enabled, 1 disabled") {
		t.Fatalf("manager observed caller mutation to SkillsCommandOptions.Disabled:\n%s", sent[0].Text)
	}
	if !strings.Contains(sent[0].Text, "planner  ops  local  local  enabled") {
		t.Fatalf("skills list missing originally enabled skill after config mutation:\n%s", sent[0].Text)
	}
}

func TestHandleSkillsCommandTelegramInstallUsesPlainTextReply(t *testing.T) {
	kind, body := ParseInboundText("/skills install https://example.com/SKILL.md --name telegram-skill")
	if kind != EventSkills {
		t.Fatalf("ParseInboundText kind = %v, want EventSkills", kind)
	}
	ch := newFakePlainSkillsChannel("telegram")
	fk := &fakeKernel{}
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		SkillsCommandOptions: SkillsCommandOptions{URLInstall: skills.URLInstallPolicy{
			Fetcher: &fakeSkillsInstallFetcher{body: gatewaySkillURLDoc("")},
			Scanner: &fakeSkillsInstallScanner{ok: true, evidence: "scan_clean"},
			Store:   newFakeSkillsInstallStore(t.TempDir()),
		}},
	}, fk, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	if err := m.handleInbound(context.Background(), InboundEvent{Platform: "telegram", ChatID: "42", UserID: "u", MsgID: "m-skills-install", Kind: kind, Text: body}); err != nil {
		t.Fatalf("handleInbound(/skills install): %v", err)
	}

	plain := ch.plainSnapshot()
	if len(plain) != 1 {
		t.Fatalf("plain skills sends = %d, want 1", len(plain))
	}
	if !strings.Contains(plain[0].Text, "url_skill_installed") || !strings.Contains(plain[0].Text, "telegram-skill") {
		t.Fatalf("plain skills install reply missing evidence:\n%s", plain[0].Text)
	}
	if sent := ch.sentSnapshot(); len(sent) != 0 {
		t.Fatalf("skills install used Markdown Send path, sent=%+v", sent)
	}
	if submits := fk.submitsSnapshot(); len(submits) != 0 {
		t.Fatalf("/skills install must not submit to kernel, got submits=%+v", submits)
	}
}

func writeGatewaySkill(t *testing.T, root, rel, name, description, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel), "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir skill: %v", err)
	}
	raw := "---\nname: " + name + "\ndescription: " + description + "\n---\n\n" + body + "\n"
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}
}

type fakeSkillsInstallFetcher struct {
	body  []byte
	calls []string
}

func (f *fakeSkillsInstallFetcher) Fetch(_ context.Context, url string) ([]byte, error) {
	f.calls = append(f.calls, url)
	return append([]byte(nil), f.body...), nil
}

type fakeSkillsInstallScanner struct {
	ok       bool
	evidence string
	calls    int
}

func (s *fakeSkillsInstallScanner) Scan(_ context.Context, _ []byte) (bool, string, error) {
	s.calls++
	return s.ok, s.evidence, nil
}

type fakeSkillsInstallStore struct {
	root  string
	files map[string][]byte
}

func newFakeSkillsInstallStore(root string) *fakeSkillsInstallStore {
	return &fakeSkillsInstallStore{root: root, files: map[string][]byte{}}
}

func (s *fakeSkillsInstallStore) ActiveDir() string { return filepath.Join(s.root, "active") }

func (s *fakeSkillsInstallStore) WriteSkill(_ context.Context, dir string, file string, body []byte) (string, error) {
	full := filepath.Join(dir, file)
	s.files[full] = append([]byte(nil), body...)
	return full, nil
}

func gatewaySkillURLDoc(name string) []byte {
	lines := []string{"---"}
	if strings.TrimSpace(name) != "" {
		lines = append(lines, "name: "+name)
	}
	lines = append(lines, "description: Gateway installed skill", "---", "", "Use this skill from gateway.")
	return []byte(strings.Join(lines, "\n"))
}

type fakePlainSkillsChannel struct {
	*fakeChannel
	plainMu sync.Mutex
	plain   []fakeSent
}

func newFakePlainSkillsChannel(name string) *fakePlainSkillsChannel {
	return &fakePlainSkillsChannel{fakeChannel: newFakeChannel(name)}
}

func (f *fakePlainSkillsChannel) SendPlain(_ context.Context, chatID, text string) (string, error) {
	f.plainMu.Lock()
	defer f.plainMu.Unlock()
	id := strconv.Itoa(f.nextMsgID)
	f.nextMsgID++
	f.plain = append(f.plain, fakeSent{ChatID: chatID, Text: text, MsgID: id})
	return id, nil
}

func (f *fakePlainSkillsChannel) SendPlainReply(_ context.Context, chatID, replyToMsgID, text string) (string, error) {
	f.plainMu.Lock()
	defer f.plainMu.Unlock()
	id := strconv.Itoa(f.nextMsgID)
	f.nextMsgID++
	f.plain = append(f.plain, fakeSent{ChatID: chatID, Text: text, MsgID: id + ":reply:" + replyToMsgID})
	return id, nil
}

func (f *fakePlainSkillsChannel) plainSnapshot() []fakeSent {
	f.plainMu.Lock()
	defer f.plainMu.Unlock()
	return cloneSlice(f.plain)
}
