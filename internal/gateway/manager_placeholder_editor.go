package gateway

import (
	"context"
	"errors"
)

type hookedPlaceholderEditor struct {
	base         placeholderEditor
	manager      *Manager
	platform     string
	threadID     string
	replyToMsgID string
}

func (h hookedPlaceholderEditor) SendPlaceholder(ctx context.Context, chatID string) (string, error) {
	const placeholderText = "⏳"

	h.manager.fireHook(ctx, HookEvent{
		Point:            HookBeforeSend,
		Platform:         h.platform,
		ChatID:           chatID,
		ThreadID:         h.threadID,
		ReplyToMessageID: h.replyToMsgID,
		Text:             placeholderText,
	})

	var (
		msgID string
		err   error
	)
	if h.replyToMsgID != "" {
		if h.threadID != "" {
			if replySender, ok := h.base.(ThreadReplyPlaceholderCapable); ok {
				msgID, err = replySender.SendThreadReplyPlaceholder(ctx, chatID, h.threadID, h.replyToMsgID)
			} else if placeholder, ok := h.base.(ThreadPlaceholderCapable); ok {
				msgID, err = placeholder.SendThreadPlaceholder(ctx, chatID, h.threadID)
			} else if replySender, ok := h.base.(ReplyPlaceholderCapable); ok {
				msgID, err = replySender.SendReplyPlaceholder(ctx, chatID, h.replyToMsgID)
			} else {
				msgID, err = h.base.SendPlaceholder(ctx, chatID)
			}
		} else if replySender, ok := h.base.(ReplyPlaceholderCapable); ok {
			msgID, err = replySender.SendReplyPlaceholder(ctx, chatID, h.replyToMsgID)
		} else {
			msgID, err = h.base.SendPlaceholder(ctx, chatID)
		}
	} else if h.threadID != "" {
		if placeholder, ok := h.base.(ThreadPlaceholderCapable); ok {
			msgID, err = placeholder.SendThreadPlaceholder(ctx, chatID, h.threadID)
		} else {
			msgID, err = h.base.SendPlaceholder(ctx, chatID)
		}
	} else {
		msgID, err = h.base.SendPlaceholder(ctx, chatID)
	}
	if err != nil {
		h.manager.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{
			Platform:      h.platform,
			PlatformState: PlatformStateFailed,
			ErrorMessage:  err.Error(),
		})
		h.manager.fireHook(ctx, HookEvent{
			Point:            HookOnError,
			Platform:         h.platform,
			ChatID:           chatID,
			ThreadID:         h.threadID,
			ReplyToMessageID: h.replyToMsgID,
			Text:             placeholderText,
			Err:              err,
		})
		return "", err
	}

	h.manager.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{
		Platform:      h.platform,
		PlatformState: PlatformStateRunning,
	})
	h.manager.fireHook(ctx, HookEvent{
		Point:            HookAfterSend,
		Platform:         h.platform,
		ChatID:           chatID,
		ThreadID:         h.threadID,
		MsgID:            msgID,
		ReplyToMessageID: h.replyToMsgID,
		Text:             placeholderText,
	})
	return msgID, nil
}

func (h hookedPlaceholderEditor) Send(ctx context.Context, chatID, text string) (string, error) {
	sender, ok := h.base.(coalescerMessageSender)
	if !ok {
		return "", errors.New("gateway: channel does not support fresh final send")
	}

	h.manager.fireHook(ctx, HookEvent{
		Point:            HookBeforeSend,
		Platform:         h.platform,
		ChatID:           chatID,
		ThreadID:         h.threadID,
		ReplyToMessageID: h.replyToMsgID,
		Text:             text,
	})

	var (
		msgID string
		err   error
	)
	if h.replyToMsgID != "" {
		if h.threadID != "" {
			if replySender, ok := h.base.(ThreadReplySender); ok {
				msgID, err = replySender.SendThreadReply(ctx, chatID, h.threadID, h.replyToMsgID, text)
			} else if threadSender, ok := h.base.(ThreadSender); ok {
				msgID, err = threadSender.SendThread(ctx, chatID, h.threadID, text)
			} else if replySender, ok := h.base.(ReplySender); ok {
				msgID, err = replySender.SendReply(ctx, chatID, h.replyToMsgID, text)
			} else {
				msgID, err = sender.Send(ctx, chatID, text)
			}
		} else if replySender, ok := h.base.(ReplySender); ok {
			msgID, err = replySender.SendReply(ctx, chatID, h.replyToMsgID, text)
		} else {
			msgID, err = sender.Send(ctx, chatID, text)
		}
	} else if h.threadID != "" {
		if threadSender, ok := h.base.(ThreadSender); ok {
			msgID, err = threadSender.SendThread(ctx, chatID, h.threadID, text)
		} else {
			msgID, err = sender.Send(ctx, chatID, text)
		}
	} else {
		msgID, err = sender.Send(ctx, chatID, text)
	}
	if err != nil {
		h.manager.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{
			Platform:      h.platform,
			PlatformState: PlatformStateFailed,
			ErrorMessage:  err.Error(),
		})
		h.manager.fireHook(ctx, HookEvent{
			Point:            HookOnError,
			Platform:         h.platform,
			ChatID:           chatID,
			ThreadID:         h.threadID,
			ReplyToMessageID: h.replyToMsgID,
			Text:             text,
			Err:              err,
		})
		return "", err
	}

	h.manager.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{
		Platform:      h.platform,
		PlatformState: PlatformStateRunning,
	})
	h.manager.fireHook(ctx, HookEvent{
		Point:            HookAfterSend,
		Platform:         h.platform,
		ChatID:           chatID,
		ThreadID:         h.threadID,
		MsgID:            msgID,
		ReplyToMessageID: h.replyToMsgID,
		Text:             text,
	})
	return msgID, nil
}

func (h hookedPlaceholderEditor) EditMessage(ctx context.Context, chatID, msgID, text string) error {
	return h.base.EditMessage(ctx, chatID, msgID, text)
}

func (h hookedPlaceholderEditor) EditMessageFinal(ctx context.Context, chatID, msgID, text string, finalize bool) error {
	if finalizer, ok := h.base.(FinalizingMessageEditor); ok {
		return finalizer.EditMessageFinal(ctx, chatID, msgID, text, finalize)
	}
	return h.base.EditMessage(ctx, chatID, msgID, text)
}

func (h hookedPlaceholderEditor) DeleteMessage(ctx context.Context, chatID, msgID string) error {
	if deleter, ok := h.base.(MessageDeleter); ok {
		return deleter.DeleteMessage(ctx, chatID, msgID)
	}
	return nil
}
