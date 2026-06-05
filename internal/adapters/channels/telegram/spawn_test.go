package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	goncho "github.com/TrebuchetDynamics/goncho/dynamicagents"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
)

func TestTelegramSpawnSlash_CreatesTopicAndBindsAgent(t *testing.T) {
	ctx := context.Background()
	client := newMockClient()
	client.UploadFilesFn = func(endpoint string, params tgbotapi.Params, files []tgbotapi.RequestFile) (*tgbotapi.APIResponse, error) {
		if endpoint == "createForumTopic" {
			result, _ := json.Marshal(map[string]any{
				"message_thread_id": 777,
				"name":              params["name"],
			})
			return &tgbotapi.APIResponse{Ok: true, Result: result}, nil
		}
		return client.uploadSuccess(1002), nil
	}
	reg := newTelegramSpawnRegistry(t)
	bot := New(Config{AllowedChatID: -100123}, client, nil)
	cancel := runTelegramSpawnManager(t, bot, reg, gateway.ManagerConfig{
		AllowedChats: map[string]string{"telegram": "-100123"},
		AllowedUsers: map[string]map[string]bool{
			"telegram": {"6586915095": true},
		},
	})
	defer cancel()

	client.updatesCh <- telegramSpawnUpdate(-100123, "supergroup", 6586915095, "/spawn Research literature reviewer")

	waitForTelegramSpawn(t, func() bool {
		agentID, found, err := reg.Resolve(ctx, goncho.BindingMatch{
			Channel:  "telegram",
			PeerKind: "group",
			PeerID:   "-100123",
			ThreadID: "777",
		})
		return err == nil && found && agentID == "research" && len(client.uploadRequests()) >= 2
	})

	uploads := client.uploadRequests()
	if uploads[0].Endpoint != "createForumTopic" {
		t.Fatalf("first upload endpoint = %q, want createForumTopic", uploads[0].Endpoint)
	}
	if uploads[0].Params["chat_id"] != "-100123" || uploads[0].Params["name"] != "Research" {
		t.Fatalf("createForumTopic params = %+v, want chat_id/name", uploads[0].Params)
	}
	ack := uploads[1]
	if ack.Endpoint != "sendMessage" {
		t.Fatalf("ack endpoint = %q, want sendMessage", ack.Endpoint)
	}
	if ack.Params["chat_id"] != "-100123" || ack.Params["message_thread_id"] != "777" {
		t.Fatalf("ack params = %+v, want chat_id/message_thread_id", ack.Params)
	}
	if !strings.Contains(ack.Params["text"], "agent_spawned") || !strings.Contains(ack.Params["text"], "Research") {
		t.Fatalf("ack text = %q, want spawned evidence with agent name", ack.Params["text"])
	}

	rec, found, err := reg.Get(ctx, "research")
	if err != nil || !found {
		t.Fatalf("Get(research) = %+v, %v, %v; want record", rec, found, err)
	}
	if rec.Persona != "literature reviewer" {
		t.Fatalf("persona = %q, want literature reviewer", rec.Persona)
	}
}

func TestTelegramSpawnSlash_RejectsNonForumChat(t *testing.T) {
	client := newMockClient()
	reg := newTelegramSpawnRegistry(t)
	bot := New(Config{AllowedChatID: -100123}, client, nil)
	cancel := runTelegramSpawnManager(t, bot, reg, gateway.ManagerConfig{
		AllowedChats: map[string]string{"telegram": "-100123"},
		AllowedUsers: map[string]map[string]bool{
			"telegram": {"6586915095": true},
		},
	})
	defer cancel()

	client.updatesCh <- telegramSpawnUpdate(-100123, "group", 6586915095, "/spawn Research literature reviewer")

	waitForTelegramSpawn(t, func() bool {
		return sentTextContains(client, "agent_spawn_requires_forum")
	})
	if len(client.uploadRequests()) != 0 {
		t.Fatalf("uploads = %+v, want no topic or ack calls for non-forum chat", client.uploadRequests())
	}
	assertTelegramSpawnRegistryEmpty(t, reg)
}

func TestTelegramSpawnSlash_RollsBackOnTopicCreationFailure(t *testing.T) {
	client := newMockClient()
	client.UploadFilesFn = func(endpoint string, params tgbotapi.Params, files []tgbotapi.RequestFile) (*tgbotapi.APIResponse, error) {
		if endpoint == "createForumTopic" {
			return nil, errors.New("Bad Request: not enough rights to manage topics")
		}
		return client.uploadSuccess(1002), nil
	}
	reg := newTelegramSpawnRegistry(t)
	bot := New(Config{AllowedChatID: -100123}, client, nil)
	cancel := runTelegramSpawnManager(t, bot, reg, gateway.ManagerConfig{
		AllowedChats: map[string]string{"telegram": "-100123"},
		AllowedUsers: map[string]map[string]bool{
			"telegram": {"6586915095": true},
		},
	})
	defer cancel()

	client.updatesCh <- telegramSpawnUpdate(-100123, "supergroup", 6586915095, "/spawn Research literature reviewer")

	waitForTelegramSpawn(t, func() bool {
		return sentTextContains(client, "agent_spawn_topic_failed") && sentTextContains(client, "not enough rights")
	})
	uploads := client.uploadRequests()
	if len(uploads) != 1 || uploads[0].Endpoint != "createForumTopic" {
		t.Fatalf("uploads = %+v, want only failing createForumTopic call", uploads)
	}
	assertTelegramSpawnRegistryEmpty(t, reg)
}

func TestTelegramSpawnSlash_RejectsNonOperatorTier(t *testing.T) {
	client := newMockClient()
	reg := newTelegramSpawnRegistry(t)
	bot := New(Config{AllowedChatID: -100123}, client, nil)
	cancel := runTelegramSpawnManager(t, bot, reg, gateway.ManagerConfig{
		AllowedChats: map[string]string{"telegram": "-100123"},
		AllowedUsers: map[string]map[string]bool{
			"telegram": {"6586915095": true},
		},
	})
	defer cancel()

	client.updatesCh <- telegramSpawnUpdate(-100123, "supergroup", 111, "/spawn Research literature reviewer")

	waitForTelegramSpawn(t, func() bool {
		return sentTextContains(client, "agent_spawn_not_authorized")
	})
	if len(client.uploadRequests()) != 0 {
		t.Fatalf("uploads = %+v, want no topic or ack calls for non-operator", client.uploadRequests())
	}
	assertTelegramSpawnRegistryEmpty(t, reg)
}

func newTelegramSpawnRegistry(t *testing.T) *goncho.DynamicAgentRegistry {
	t.Helper()
	store, err := memory.OpenSqlite(t.TempDir()+"/memory.db", 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	t.Cleanup(func() { _ = store.Close(context.Background()) })
	reg, err := goncho.NewDynamicAgentRegistry(store.DB())
	if err != nil {
		t.Fatalf("NewDynamicAgentRegistry: %v", err)
	}
	return reg
}

func runTelegramSpawnManager(t *testing.T, bot *Bot, reg *goncho.DynamicAgentRegistry, cfg gateway.ManagerConfig) context.CancelFunc {
	t.Helper()
	cfg.DynamicAgentRegistry = reg
	m := gateway.NewManagerWithSubmitter(cfg, nil, slog.Default())
	if err := m.Register(bot); err != nil {
		t.Fatalf("Register: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() { _ = m.Run(ctx) }()
	t.Cleanup(cancel)
	return cancel
}

func telegramSpawnUpdate(chatID int64, chatType string, userID int64, text string) tgbotapi.Update {
	return tgbotapi.Update{
		UpdateID: 1,
		Message: &tgbotapi.Message{
			MessageID: 10,
			Text:      text,
			Chat:      &tgbotapi.Chat{ID: chatID, Type: chatType},
			From:      &tgbotapi.User{ID: userID, FirstName: "operator"},
		},
	}
}

func waitForTelegramSpawn(t *testing.T, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if ok() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
}

func sentTextContains(client *mockClient, needle string) bool {
	for _, sent := range client.sentMessages() {
		msg, ok := sent.(tgbotapi.MessageConfig)
		if ok && strings.Contains(msg.Text, needle) {
			return true
		}
	}
	return false
}

func assertTelegramSpawnRegistryEmpty(t *testing.T, reg *goncho.DynamicAgentRegistry) {
	t.Helper()
	records, err := reg.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("registry records = %+v, want empty", records)
	}
}
