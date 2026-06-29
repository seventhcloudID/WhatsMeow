package wa

import (
	"context"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func (m *Manager) CheckWhatsApp(ctx context.Context, phones []string) ([]types.IsOnWhatsAppResponse, error) {
	client, err := m.getClient()
	if err != nil {
		return nil, err
	}
	return client.IsOnWhatsApp(ctx, phones)
}

func (m *Manager) GetUserInfo(ctx context.Context, jids []string) (map[types.JID]types.UserInfo, error) {
	client, err := m.getClient()
	if err != nil {
		return nil, err
	}
	parsed := make([]types.JID, 0, len(jids))
	for _, j := range jids {
		pj, err := parseRecipient(j)
		if err != nil {
			return nil, err
		}
		parsed = append(parsed, pj)
	}
	return client.GetUserInfo(ctx, parsed)
}

func (m *Manager) GetProfilePicture(ctx context.Context, jidStr string, preview bool) (*types.ProfilePictureInfo, error) {
	jid, err := parseRecipient(jidStr)
	if err != nil {
		return nil, err
	}
	client, err := m.getClient()
	if err != nil {
		return nil, err
	}
	return client.GetProfilePictureInfo(ctx, jid, &whatsmeow.GetProfilePictureParams{
		Preview: preview,
	})
}

func (m *Manager) SetAboutStatus(ctx context.Context, msg string) error {
	client, err := m.getClient()
	if err != nil {
		return err
	}
	return client.SetStatusMessage(ctx, msg)
}

func (m *Manager) GetBusinessProfile(ctx context.Context, jidStr string) (*types.BusinessProfile, error) {
	jid, err := parseRecipient(jidStr)
	if err != nil {
		return nil, err
	}
	client, err := m.getClient()
	if err != nil {
		return nil, err
	}
	return client.GetBusinessProfile(ctx, jid)
}

func (m *Manager) GetBlocklist(ctx context.Context) (*types.Blocklist, error) {
	client, err := m.getClient()
	if err != nil {
		return nil, err
	}
	return client.GetBlocklist(ctx)
}

func (m *Manager) UpdateBlocklist(ctx context.Context, jidStr, action string) error {
	jid, err := parseRecipient(jidStr)
	if err != nil {
		return err
	}
	client, err := m.getClient()
	if err != nil {
		return err
	}
	var act events.BlocklistChangeAction
	switch action {
	case "block":
		act = events.BlocklistChangeActionBlock
	case "unblock":
		act = events.BlocklistChangeActionUnblock
	default:
		return errInvalidAction("block/unblock")
	}
	_, err = client.UpdateBlocklist(ctx, jid, act)
	return err
}

func errInvalidAction(allowed string) error {
	return &invalidActionError{allowed: allowed}
}

type invalidActionError struct{ allowed string }

func (e *invalidActionError) Error() string {
	return "action tidak valid, gunakan: " + e.allowed
}

func (m *Manager) GetPrivacySettings(ctx context.Context) (types.PrivacySettings, error) {
	client, err := m.getClient()
	if err != nil {
		return types.PrivacySettings{}, err
	}
	return client.GetPrivacySettings(ctx), nil
}

func (m *Manager) GetUserDevices(ctx context.Context, jids []string) ([]types.JID, error) {
	client, err := m.getClient()
	if err != nil {
		return nil, err
	}
	parsed := make([]types.JID, 0, len(jids))
	for _, j := range jids {
		pj, err := parseRecipient(j)
		if err != nil {
			return nil, err
		}
		parsed = append(parsed, pj)
	}
	return client.GetUserDevices(ctx, parsed)
}
