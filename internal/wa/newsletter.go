package wa

import (
	"context"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

func (m *Manager) ListNewsletters(ctx context.Context) ([]*types.NewsletterMetadata, error) {
	client, err := m.getClient()
	if err != nil {
		return nil, err
	}
	return client.GetSubscribedNewsletters(ctx)
}

func (m *Manager) GetNewsletterInfo(ctx context.Context, jidStr string) (*types.NewsletterMetadata, error) {
	jid, err := parseRecipient(jidStr)
	if err != nil {
		return nil, err
	}
	client, err := m.getClient()
	if err != nil {
		return nil, err
	}
	return client.GetNewsletterInfo(ctx, jid)
}

func (m *Manager) GetNewsletterByInvite(ctx context.Context, inviteKey string) (*types.NewsletterMetadata, error) {
	client, err := m.getClient()
	if err != nil {
		return nil, err
	}
	return client.GetNewsletterInfoWithInvite(ctx, inviteKey)
}

func (m *Manager) FollowNewsletter(ctx context.Context, jidStr string) error {
	jid, err := parseRecipient(jidStr)
	if err != nil {
		return err
	}
	client, err := m.getClient()
	if err != nil {
		return err
	}
	return client.FollowNewsletter(ctx, jid)
}

func (m *Manager) UnfollowNewsletter(ctx context.Context, jidStr string) error {
	jid, err := parseRecipient(jidStr)
	if err != nil {
		return err
	}
	client, err := m.getClient()
	if err != nil {
		return err
	}
	return client.UnfollowNewsletter(ctx, jid)
}

func (m *Manager) CreateNewsletter(ctx context.Context, name, description string, picture []byte) (*types.NewsletterMetadata, error) {
	client, err := m.getClient()
	if err != nil {
		return nil, err
	}
	return client.CreateNewsletter(ctx, whatsmeow.CreateNewsletterParams{
		Name:        name,
		Description: description,
		Picture:     picture,
	})
}

func (m *Manager) NewsletterMessages(ctx context.Context, jidStr string, count int) ([]*types.NewsletterMessage, error) {
	jid, err := parseRecipient(jidStr)
	if err != nil {
		return nil, err
	}
	client, err := m.getClient()
	if err != nil {
		return nil, err
	}
	if count <= 0 {
		count = 10
	}
	return client.GetNewsletterMessages(ctx, jid, &whatsmeow.GetNewsletterMessagesParams{
		Count: count,
	})
}

func (m *Manager) NewsletterToggleMute(ctx context.Context, jidStr string, mute bool) error {
	jid, err := parseRecipient(jidStr)
	if err != nil {
		return err
	}
	client, err := m.getClient()
	if err != nil {
		return err
	}
	return client.NewsletterToggleMute(ctx, jid, mute)
}
