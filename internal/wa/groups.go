package wa

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

type CreateGroupOpts struct {
	Name         string   `json:"name"`
	Participants []string `json:"participants"`
}

type GroupParticipantsOpts struct {
	GroupJID     string   `json:"group_jid"`
	Participants []string `json:"participants"`
	Action       string   `json:"action"` // add, remove, promote, demote
}

func (m *Manager) ListGroups(ctx context.Context) ([]*types.GroupInfo, error) {
	client, err := m.getClient()
	if err != nil {
		return nil, err
	}
	return client.GetJoinedGroups(ctx)
}

func (m *Manager) GetGroupInfo(ctx context.Context, jidStr string) (*types.GroupInfo, error) {
	jid, err := parseRecipient(jidStr)
	if err != nil {
		return nil, err
	}
	client, err := m.getClient()
	if err != nil {
		return nil, err
	}
	return client.GetGroupInfo(ctx, jid)
}

func (m *Manager) CreateGroup(ctx context.Context, opts CreateGroupOpts) (*types.GroupInfo, error) {
	client, err := m.getClient()
	if err != nil {
		return nil, err
	}
	participants := make([]types.JID, 0, len(opts.Participants))
	for _, p := range opts.Participants {
		j, err := parseRecipient(p)
		if err != nil {
			return nil, err
		}
		participants = append(participants, j)
	}
	return client.CreateGroup(ctx, whatsmeow.ReqCreateGroup{
		Name:         opts.Name,
		Participants: participants,
	})
}

func (m *Manager) LeaveGroup(ctx context.Context, jidStr string) error {
	jid, err := parseRecipient(jidStr)
	if err != nil {
		return err
	}
	client, err := m.getClient()
	if err != nil {
		return err
	}
	return client.LeaveGroup(ctx, jid)
}

func (m *Manager) GetGroupInviteLink(ctx context.Context, jidStr string, reset bool) (string, error) {
	jid, err := parseRecipient(jidStr)
	if err != nil {
		return "", err
	}
	client, err := m.getClient()
	if err != nil {
		return "", err
	}
	return client.GetGroupInviteLink(ctx, jid, reset)
}

func (m *Manager) GetGroupInfoFromLink(ctx context.Context, code string) (*types.GroupInfo, error) {
	client, err := m.getClient()
	if err != nil {
		return nil, err
	}
	return client.GetGroupInfoFromLink(ctx, code)
}

func (m *Manager) JoinGroupWithLink(ctx context.Context, code string) (types.JID, error) {
	client, err := m.getClient()
	if err != nil {
		return types.EmptyJID, err
	}
	return client.JoinGroupWithLink(ctx, code)
}

func (m *Manager) UpdateGroupParticipants(ctx context.Context, opts GroupParticipantsOpts) ([]types.GroupParticipant, error) {
	jid, err := parseRecipient(opts.GroupJID)
	if err != nil {
		return nil, err
	}
	client, err := m.getClient()
	if err != nil {
		return nil, err
	}
	participants := make([]types.JID, 0, len(opts.Participants))
	for _, p := range opts.Participants {
		j, err := parseRecipient(p)
		if err != nil {
			return nil, err
		}
		participants = append(participants, j)
	}
	action := whatsmeow.ParticipantChange(opts.Action)
	if action != whatsmeow.ParticipantChangeAdd &&
		action != whatsmeow.ParticipantChangeRemove &&
		action != whatsmeow.ParticipantChangePromote &&
		action != whatsmeow.ParticipantChangeDemote {
		return nil, fmt.Errorf("action tidak valid: %s", opts.Action)
	}
	return client.UpdateGroupParticipants(ctx, jid, participants, action)
}

func (m *Manager) SetGroupName(ctx context.Context, jidStr, name string) error {
	jid, err := parseRecipient(jidStr)
	if err != nil {
		return err
	}
	client, err := m.getClient()
	if err != nil {
		return err
	}
	return client.SetGroupName(ctx, jid, name)
}

func (m *Manager) SetGroupDescription(ctx context.Context, jidStr, description string) error {
	jid, err := parseRecipient(jidStr)
	if err != nil {
		return err
	}
	client, err := m.getClient()
	if err != nil {
		return err
	}
	return client.SetGroupDescription(ctx, jid, description)
}

func (m *Manager) SetGroupPhoto(ctx context.Context, jidStr string, photo []byte) (string, error) {
	jid, err := parseRecipient(jidStr)
	if err != nil {
		return "", err
	}
	client, err := m.getClient()
	if err != nil {
		return "", err
	}
	return client.SetGroupPhoto(ctx, jid, photo)
}

func (m *Manager) SetGroupLocked(ctx context.Context, jidStr string, locked bool) error {
	jid, err := parseRecipient(jidStr)
	if err != nil {
		return err
	}
	client, err := m.getClient()
	if err != nil {
		return err
	}
	return client.SetGroupLocked(ctx, jid, locked)
}

func (m *Manager) SetGroupAnnounce(ctx context.Context, jidStr string, announce bool) error {
	jid, err := parseRecipient(jidStr)
	if err != nil {
		return err
	}
	client, err := m.getClient()
	if err != nil {
		return err
	}
	return client.SetGroupAnnounce(ctx, jid, announce)
}
