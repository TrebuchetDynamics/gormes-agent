package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	slackapi "github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

type realClient struct {
	api    *slackapi.Client
	socket *socketmode.Client
	events <-chan socketmode.Event
	runFn  func(context.Context) error
	ackFn  func(socketmode.Request) error

	mu      sync.Mutex
	pending map[string]socketmode.Request
}

var _ Client = (*realClient)(nil)

func NewRealClient(botToken, appToken string) Client {
	api := slackapi.New(botToken, slackapi.OptionAppLevelToken(appToken))
	socket := socketmode.New(api)
	return &realClient{
		api:     api,
		socket:  socket,
		events:  socket.Events,
		runFn:   socket.RunContext,
		ackFn:   func(req socketmode.Request) error { return socket.Ack(req) },
		pending: make(map[string]socketmode.Request),
	}
}

func (c *realClient) AuthTest(ctx context.Context) (string, error) {
	resp, err := c.api.AuthTestContext(ctx)
	if err != nil {
		return "", err
	}
	return resp.UserID, nil
}

func (c *realClient) Run(ctx context.Context, fn func(Event)) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- c.runFn(ctx)
	}()

	for {
		select {
		case <-ctx.Done():
			err := <-errCh
			if err == nil || ctx.Err() != nil {
				return nil
			}
			return err
		case err := <-errCh:
			if ctx.Err() != nil {
				return nil
			}
			return err
		case evt, ok := <-c.events:
			if !ok {
				err := <-errCh
				if ctx.Err() != nil {
					return nil
				}
				return err
			}
			c.handleSocketEvent(evt, fn)
		}
	}
}

func (c *realClient) Ack(requestID string) error {
	if requestID == "" {
		return nil
	}

	c.mu.Lock()
	req, ok := c.pending[requestID]
	c.mu.Unlock()
	if !ok {
		return nil
	}
	if err := c.ackFn(req); err != nil {
		return err
	}
	c.mu.Lock()
	delete(c.pending, requestID)
	c.mu.Unlock()
	return nil
}

func (c *realClient) PostMessage(ctx context.Context, channelID, threadTS, text string) (string, error) {
	opts := []slackapi.MsgOption{slackapi.MsgOptionText(text, false)}
	if threadTS != "" {
		opts = append(opts, slackapi.MsgOptionTS(threadTS))
	}
	_, ts, err := c.api.PostMessageContext(ctx, channelID, opts...)
	if err != nil {
		return "", err
	}
	return ts, nil
}

func (c *realClient) PostBlockMessage(ctx context.Context, channelID, threadTS, text string, blocks []SlackBlock) (string, error) {
	apiBlocks, err := slackBlocksToAPI(blocks)
	if err != nil {
		return "", err
	}
	opts := []slackapi.MsgOption{
		slackapi.MsgOptionText(text, false),
		slackapi.MsgOptionBlocks(apiBlocks...),
	}
	if threadTS != "" {
		opts = append(opts, slackapi.MsgOptionTS(threadTS))
	}
	_, ts, err := c.api.PostMessageContext(ctx, channelID, opts...)
	if err != nil {
		return "", err
	}
	return ts, nil
}

func (c *realClient) UpdateMessage(ctx context.Context, channelID, ts, text string) error {
	_, _, _, err := c.api.UpdateMessageContext(ctx, channelID, ts, slackapi.MsgOptionText(text, false))
	return err
}

func (c *realClient) UpdateBlockMessage(ctx context.Context, channelID, ts, text string, blocks []SlackBlock) error {
	apiBlocks, err := slackBlocksToAPI(blocks)
	if err != nil {
		return err
	}
	_, _, _, err = c.api.UpdateMessageContext(
		ctx,
		channelID,
		ts,
		slackapi.MsgOptionText(text, false),
		slackapi.MsgOptionBlocks(apiBlocks...),
	)
	return err
}

func (c *realClient) UploadFile(ctx context.Context, channelID, threadTS, filePath string) (string, error) {
	mediaPath := strings.TrimSpace(filePath)
	if mediaPath == "" {
		return "", fmt.Errorf("slack: media path is required")
	}
	if _, err := os.Stat(mediaPath); err != nil {
		return "", fmt.Errorf("slack: media unavailable")
	}
	file, err := c.api.UploadFileContext(ctx, slackapi.UploadFileParameters{
		File:            mediaPath,
		Filename:        filepath.Base(mediaPath),
		Title:           filepath.Base(mediaPath),
		Channel:         channelID,
		ThreadTimestamp: threadTS,
	})
	if err != nil {
		return "", err
	}
	if file == nil {
		return "", nil
	}
	return file.ID, nil
}

func (c *realClient) handleSocketEvent(evt socketmode.Event, fn func(Event)) {
	switch evt.Type {
	case socketmode.EventTypeEventsAPI:
		c.handleEventsAPI(evt, fn)
	case socketmode.EventTypeSlashCommand:
		c.handleSlashCommand(evt, fn)
	case socketmode.EventTypeInteractive:
		c.handleInteractive(evt, fn)
	}
}

func (c *realClient) handleEventsAPI(evt socketmode.Event, fn func(Event)) {
	if evt.Request == nil {
		return
	}

	eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
	if !ok {
		_ = c.autoAck(evt.Request)
		return
	}
	if eventsAPIEvent.Type != slackevents.CallbackEvent {
		_ = c.autoAck(evt.Request)
		return
	}

	msg, ok := eventsAPIEvent.InnerEvent.Data.(*slackevents.MessageEvent)
	if !ok || msg == nil {
		_ = c.autoAck(evt.Request)
		return
	}

	requestID := evt.Request.EnvelopeID
	c.mu.Lock()
	c.pending[requestID] = *evt.Request
	c.mu.Unlock()

	fn(Event{
		RequestID: requestID,
		ChannelID: msg.Channel,
		TeamID:    eventsAPIEvent.TeamID,
		UserID:    msg.User,
		Text:      msg.Text,
		ChatType:  msg.ChannelType,
		Timestamp: msg.TimeStamp,
		ThreadTS:  msg.ThreadTimeStamp,
		SubType:   msg.SubType,
		BotID:     msg.BotID,
	})
}

func (c *realClient) handleSlashCommand(evt socketmode.Event, fn func(Event)) {
	if evt.Request == nil {
		return
	}

	cmd, ok := evt.Data.(slackapi.SlashCommand)
	if !ok {
		_ = c.autoAck(evt.Request)
		return
	}

	requestID := evt.Request.EnvelopeID
	c.mu.Lock()
	c.pending[requestID] = *evt.Request
	c.mu.Unlock()

	fn(Event{
		RequestID: requestID,
		ChannelID: cmd.ChannelID,
		TeamID:    cmd.TeamID,
		UserID:    cmd.UserID,
		Text:      strings.TrimSpace(cmd.Command + " " + cmd.Text),
		ChatType:  cmd.ChannelName,
	})
}

func (c *realClient) handleInteractive(evt socketmode.Event, fn func(Event)) {
	if evt.Request == nil {
		return
	}

	callback, ok := evt.Data.(slackapi.InteractionCallback)
	if !ok {
		_ = c.autoAck(evt.Request)
		return
	}
	action := firstSlackApprovalBlockAction(callback)
	if action == nil || fn == nil {
		_ = c.autoAck(evt.Request)
		return
	}

	requestID := evt.Request.EnvelopeID
	c.mu.Lock()
	c.pending[requestID] = *evt.Request
	c.mu.Unlock()

	channelID := firstNonEmpty(callback.Channel.ID, callback.Container.ChannelID)
	messageTS := firstNonEmpty(callback.Message.Timestamp, callback.MessageTs, callback.Container.MessageTs)
	userName := firstNonEmpty(callback.User.Name, callback.User.ID)
	fn(Event{
		RequestID: requestID,
		ChannelID: channelID,
		TeamID:    callback.Team.ID,
		UserID:    callback.User.ID,
		Timestamp: messageTS,
		ThreadTS:  callback.Container.ThreadTs,
		ApprovalAction: &ApprovalAction{
			ActionID:   action.ActionID,
			SessionKey: action.Value,
			MessageTS:  messageTS,
			ChannelID:  channelID,
			UserID:     callback.User.ID,
			UserName:   userName,
		},
	})
}

func firstSlackApprovalBlockAction(callback slackapi.InteractionCallback) *slackapi.BlockAction {
	if callback.Type != slackapi.InteractionTypeBlockActions {
		return nil
	}
	for _, action := range callback.ActionCallback.BlockActions {
		if action != nil && isSlackApprovalAction(action.ActionID) {
			return action
		}
	}
	return nil
}

func (c *realClient) autoAck(req *socketmode.Request) error {
	if req == nil {
		return nil
	}
	return c.ackFn(*req)
}

type rawSlackAPIBlock SlackBlock

func (b rawSlackAPIBlock) BlockType() slackapi.MessageBlockType {
	block := SlackBlock(b)
	blockType, _ := block["type"].(string)
	return slackapi.MessageBlockType(blockType)
}

func (b rawSlackAPIBlock) ID() string {
	block := SlackBlock(b)
	blockID, _ := block["block_id"].(string)
	return blockID
}

func (b rawSlackAPIBlock) MarshalJSON() ([]byte, error) {
	return json.Marshal(SlackBlock(b))
}

func slackBlocksToAPI(blocks []SlackBlock) ([]slackapi.Block, error) {
	if blocks == nil {
		return nil, nil
	}
	out := make([]slackapi.Block, 0, len(blocks))
	for _, block := range blocks {
		if _, err := json.Marshal(block); err != nil {
			return nil, err
		}
		out = append(out, rawSlackAPIBlock(block))
	}
	return out, nil
}
