package wa

import (
	"context"

	"go.mau.fi/whatsmeow/types"
)

func (m *Manager) SendGlobalPresence(ctx context.Context, state string) error {
	client, err := m.getClient()
	if err != nil {
		return err
	}
	var presence types.Presence
	switch state {
	case "available", "online":
		presence = types.PresenceAvailable
	case "unavailable", "offline":
		presence = types.PresenceUnavailable
	default:
		return errInvalidAction("available/unavailable")
	}
	return client.SendPresence(ctx, presence)
}

func (m *Manager) SendTyping(ctx context.Context, jidStr, state, media string) error {
	jid, err := parseRecipient(jidStr)
	if err != nil {
		return err
	}
	client, err := m.getClient()
	if err != nil {
		return err
	}
	var chatPresence types.ChatPresence
	switch state {
	case "composing", "typing":
		chatPresence = types.ChatPresenceComposing
	case "paused":
		chatPresence = types.ChatPresencePaused
	default:
		return errInvalidAction("composing/paused")
	}
	var chatMedia types.ChatPresenceMedia
	switch media {
	case "", "text":
		chatMedia = types.ChatPresenceMediaText
	case "audio":
		chatMedia = types.ChatPresenceMediaAudio
	default:
		chatMedia = types.ChatPresenceMediaText
	}
	return client.SendChatPresence(ctx, jid, chatPresence, chatMedia)
}

func (m *Manager) SubscribePresence(ctx context.Context, jidStr string) error {
	jid, err := parseRecipient(jidStr)
	if err != nil {
		return err
	}
	client, err := m.getClient()
	if err != nil {
		return err
	}
	return client.SubscribePresence(ctx, jid)
}
