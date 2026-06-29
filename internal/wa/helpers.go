package wa

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
)

func (m *Manager) getClient() (*whatsmeow.Client, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.client.Store.ID == nil {
		return nil, fmt.Errorf("belum login")
	}
	if !m.client.IsConnected() {
		return nil, fmt.Errorf("tidak terhubung ke WhatsApp")
	}
	return m.client, nil
}

func (m *Manager) sendMessage(ctx context.Context, to types.JID, msg *waProto.Message) (*SendResult, error) {
	client, err := m.getClient()
	if err != nil {
		return nil, err
	}
	resp, err := client.SendMessage(ctx, to, msg)
	if err != nil {
		return nil, err
	}
	return &SendResult{MessageID: resp.ID, Timestamp: resp.Timestamp}, nil
}

func parseJIDOrRecipient(s string) (types.JID, error) {
	return parseRecipient(s)
}

func parseOptionalJID(s string) (types.JID, error) {
	if s == "" {
		return types.EmptyJID, fmt.Errorf("jid kosong")
	}
	return parseRecipient(s)
}
