package wa

import (
	"context"
	"time"

	"go.mau.fi/whatsmeow/appstate"
	"go.mau.fi/whatsmeow/types"
)

type MarkReadOpts struct {
	Chat      string   `json:"chat"`
	Sender    string   `json:"sender"`
	MessageIDs []string `json:"message_ids"`
	Timestamp time.Time `json:"timestamp"`
}

type ChatActionOpts struct {
	Chat   string `json:"chat"`
	Mute   *bool  `json:"mute"`
	Hours  int    `json:"hours"`
	Archive *bool `json:"archive"`
	Pin    *bool  `json:"pin"`
}

func (m *Manager) MarkRead(ctx context.Context, opts MarkReadOpts) error {
	chat, err := parseRecipient(opts.Chat)
	if err != nil {
		return err
	}
	sender := chat
	if opts.Sender != "" {
		sender, err = parseRecipient(opts.Sender)
		if err != nil {
			return err
		}
	}
	client, err := m.getClient()
	if err != nil {
		return err
	}
	ts := opts.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}
	ids := make([]types.MessageID, len(opts.MessageIDs))
	copy(ids, opts.MessageIDs)
	return client.MarkRead(ctx, ids, ts, chat, sender)
}

func (m *Manager) ChatAction(ctx context.Context, opts ChatActionOpts) error {
	jid, err := parseRecipient(opts.Chat)
	if err != nil {
		return err
	}
	client, err := m.getClient()
	if err != nil {
		return err
	}
	if opts.Mute != nil {
		duration := time.Duration(opts.Hours) * time.Hour
		if opts.Hours <= 0 && *opts.Mute {
			duration = 8 * time.Hour
		}
		patch := appstate.BuildMute(jid, *opts.Mute, duration)
		if err := client.SendAppState(ctx, patch); err != nil {
			return err
		}
	}
	if opts.Archive != nil {
		patch := appstate.BuildArchive(jid, *opts.Archive, time.Time{}, nil)
		if err := client.SendAppState(ctx, patch); err != nil {
			return err
		}
	}
	if opts.Pin != nil {
		patch := appstate.BuildPin(jid, *opts.Pin)
		if err := client.SendAppState(ctx, patch); err != nil {
			return err
		}
	}
	return nil
}
