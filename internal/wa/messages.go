package wa

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"google.golang.org/protobuf/proto"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
)

type TextOpts struct {
	To       string
	Text     string
	ReplyTo  string // message ID
	ReplyJID string // chat JID for reply context
}

type LocationOpts struct {
	To        string
	Latitude  float64
	Longitude float64
	Name      string
	Address   string
}

type ContactOpts struct {
	To          string
	DisplayName string
	VCard       string
}

type PollOpts struct {
	To                    string
	Name                  string
	Options               []string
	SelectableOptionCount int
}

type PollVoteOpts struct {
	Chat      string
	Sender    string
	MessageID string
	Options   []string
}

type ReactionOpts struct {
	To        string
	MessageID string
	Sender    string // required for groups
	Emoji     string
}

type RevokeOpts struct {
	To        string
	MessageID string
	Sender    string
}

type EditOpts struct {
	To        string
	MessageID string
	Text      string
}

type DisappearingOpts struct {
	To     string
	Timer  time.Duration
}

func (m *Manager) SendText(ctx context.Context, to, text string) (*SendResult, error) {
	return m.SendTextAdvanced(ctx, TextOpts{To: to, Text: text})
}

func (m *Manager) SendTextAdvanced(ctx context.Context, opts TextOpts) (*SendResult, error) {
	jid, err := parseRecipient(opts.To)
	if err != nil {
		return nil, err
	}
	var msg *waProto.Message
	if opts.ReplyTo != "" {
		chatJID := jid
		if opts.ReplyJID != "" {
			chatJID, err = parseRecipient(opts.ReplyJID)
			if err != nil {
				return nil, err
			}
		}
		client, err := m.getClient()
		if err != nil {
			return nil, err
		}
		msg = &waProto.Message{ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(opts.Text),
			ContextInfo: &waProto.ContextInfo{
				StanzaID:    proto.String(opts.ReplyTo),
				Participant: proto.String(chatJID.String()),
			},
		}}
		_ = client
	} else {
		msg = &waProto.Message{Conversation: proto.String(opts.Text)}
	}
	return m.sendMessage(ctx, jid, msg)
}

func (m *Manager) sendMedia(ctx context.Context, to string, data []byte, mediaType whatsmeow.MediaType, build func(uploaded whatsmeow.UploadResponse, data []byte) *waProto.Message) (*SendResult, error) {
	jid, err := parseRecipient(to)
	if err != nil {
		return nil, err
	}
	client, err := m.getClient()
	if err != nil {
		return nil, err
	}
	uploaded, err := client.Upload(ctx, data, mediaType)
	if err != nil {
		return nil, fmt.Errorf("upload: %w", err)
	}
	return m.sendMessage(ctx, jid, build(uploaded, data))
}

func (m *Manager) SendImage(ctx context.Context, to string, data []byte, caption string) (*SendResult, error) {
	return m.sendMedia(ctx, to, data, whatsmeow.MediaImage, func(u whatsmeow.UploadResponse, data []byte) *waProto.Message {
		return &waProto.Message{ImageMessage: &waProto.ImageMessage{
			Caption:       proto.String(caption),
			URL:           proto.String(u.URL),
			DirectPath:    proto.String(u.DirectPath),
			MediaKey:      u.MediaKey,
			Mimetype:      proto.String(http.DetectContentType(data)),
			FileEncSHA256: u.FileEncSHA256,
			FileSHA256:    u.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(data))),
		}}
	})
}

func (m *Manager) SendVideo(ctx context.Context, to string, data []byte, caption string) (*SendResult, error) {
	return m.sendMedia(ctx, to, data, whatsmeow.MediaVideo, func(u whatsmeow.UploadResponse, data []byte) *waProto.Message {
		return &waProto.Message{VideoMessage: &waProto.VideoMessage{
			Caption:       proto.String(caption),
			URL:           proto.String(u.URL),
			DirectPath:    proto.String(u.DirectPath),
			MediaKey:      u.MediaKey,
			Mimetype:      proto.String(http.DetectContentType(data)),
			FileEncSHA256: u.FileEncSHA256,
			FileSHA256:    u.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(data))),
		}}
	})
}

func (m *Manager) SendAudio(ctx context.Context, to string, data []byte, ptt bool, seconds uint32) (*SendResult, error) {
	return m.sendMedia(ctx, to, data, whatsmeow.MediaAudio, func(u whatsmeow.UploadResponse, data []byte) *waProto.Message {
		mime := http.DetectContentType(data)
		if ptt {
			mime = "audio/ogg; codecs=opus"
		}
		return &waProto.Message{AudioMessage: &waProto.AudioMessage{
			URL:           proto.String(u.URL),
			DirectPath:    proto.String(u.DirectPath),
			MediaKey:      u.MediaKey,
			Mimetype:      proto.String(mime),
			FileEncSHA256: u.FileEncSHA256,
			FileSHA256:    u.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(data))),
			PTT:           proto.Bool(ptt),
			Seconds:       proto.Uint32(seconds),
		}}
	})
}

func (m *Manager) SendDocument(ctx context.Context, to string, data []byte, filename, mimetype string) (*SendResult, error) {
	if mimetype == "" {
		mimetype = http.DetectContentType(data)
	}
	if filename == "" {
		filename = "document"
	}
	return m.sendMedia(ctx, to, data, whatsmeow.MediaDocument, func(u whatsmeow.UploadResponse, _ []byte) *waProto.Message {
		return &waProto.Message{DocumentMessage: &waProto.DocumentMessage{
			URL:           proto.String(u.URL),
			DirectPath:    proto.String(u.DirectPath),
			MediaKey:      u.MediaKey,
			Mimetype:      proto.String(mimetype),
			FileEncSHA256: u.FileEncSHA256,
			FileSHA256:    u.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(data))),
			FileName:      proto.String(filename),
		}}
	})
}

func (m *Manager) SendSticker(ctx context.Context, to string, data []byte) (*SendResult, error) {
	return m.sendMedia(ctx, to, data, whatsmeow.MediaImage, func(u whatsmeow.UploadResponse, data []byte) *waProto.Message {
		return &waProto.Message{StickerMessage: &waProto.StickerMessage{
			URL:           proto.String(u.URL),
			DirectPath:    proto.String(u.DirectPath),
			MediaKey:      u.MediaKey,
			Mimetype:      proto.String(http.DetectContentType(data)),
			FileEncSHA256: u.FileEncSHA256,
			FileSHA256:    u.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(data))),
		}}
	})
}

func (m *Manager) SendLocation(ctx context.Context, opts LocationOpts) (*SendResult, error) {
	jid, err := parseRecipient(opts.To)
	if err != nil {
		return nil, err
	}
	msg := &waProto.Message{LocationMessage: &waProto.LocationMessage{
		DegreesLatitude:  proto.Float64(opts.Latitude),
		DegreesLongitude: proto.Float64(opts.Longitude),
		Name:             proto.String(opts.Name),
		Address:          proto.String(opts.Address),
	}}
	return m.sendMessage(ctx, jid, msg)
}

func (m *Manager) SendContact(ctx context.Context, opts ContactOpts) (*SendResult, error) {
	jid, err := parseRecipient(opts.To)
	if err != nil {
		return nil, err
	}
	msg := &waProto.Message{ContactMessage: &waProto.ContactMessage{
		DisplayName: proto.String(opts.DisplayName),
		Vcard:       proto.String(opts.VCard),
	}}
	return m.sendMessage(ctx, jid, msg)
}

func (m *Manager) SendPoll(ctx context.Context, opts PollOpts) (*SendResult, error) {
	jid, err := parseRecipient(opts.To)
	if err != nil {
		return nil, err
	}
	client, err := m.getClient()
	if err != nil {
		return nil, err
	}
	count := opts.SelectableOptionCount
	if count <= 0 {
		count = 1
	}
	msg := client.BuildPollCreation(opts.Name, opts.Options, count)
	return m.sendMessage(ctx, jid, msg)
}

func (m *Manager) SendPollVote(ctx context.Context, opts PollVoteOpts) (*SendResult, error) {
	chat, err := parseRecipient(opts.Chat)
	if err != nil {
		return nil, err
	}
	sender := chat
	if opts.Sender != "" {
		sender, err = parseRecipient(opts.Sender)
		if err != nil {
			return nil, err
		}
	}
	client, err := m.getClient()
	if err != nil {
		return nil, err
	}
	info := &types.MessageInfo{
		MessageSource: types.MessageSource{
			Chat:   chat,
			Sender: sender,
		},
		ID: opts.MessageID,
	}
	msg, err := client.BuildPollVote(ctx, info, opts.Options)
	if err != nil {
		return nil, fmt.Errorf("build poll vote: %w", err)
	}
	return m.sendMessage(ctx, chat, msg)
}

func (m *Manager) SendReaction(ctx context.Context, opts ReactionOpts) (*SendResult, error) {
	chat, err := parseRecipient(opts.To)
	if err != nil {
		return nil, err
	}
	sender := chat
	if opts.Sender != "" {
		sender, err = parseRecipient(opts.Sender)
		if err != nil {
			return nil, err
		}
	}
	client, err := m.getClient()
	if err != nil {
		return nil, err
	}
	msg := client.BuildReaction(chat, sender, opts.MessageID, opts.Emoji)
	return m.sendMessage(ctx, chat, msg)
}

func (m *Manager) SendRevoke(ctx context.Context, opts RevokeOpts) (*SendResult, error) {
	chat, err := parseRecipient(opts.To)
	if err != nil {
		return nil, err
	}
	sender := chat
	if opts.Sender != "" {
		sender, err = parseRecipient(opts.Sender)
		if err != nil {
			return nil, err
		}
	}
	client, err := m.getClient()
	if err != nil {
		return nil, err
	}
	msg := client.BuildRevoke(chat, sender, opts.MessageID)
	return m.sendMessage(ctx, chat, msg)
}

func (m *Manager) SendEdit(ctx context.Context, opts EditOpts) (*SendResult, error) {
	chat, err := parseRecipient(opts.To)
	if err != nil {
		return nil, err
	}
	client, err := m.getClient()
	if err != nil {
		return nil, err
	}
	newContent := &waProto.Message{Conversation: proto.String(opts.Text)}
	msg := client.BuildEdit(chat, opts.MessageID, newContent)
	return m.sendMessage(ctx, chat, msg)
}

func (m *Manager) SetDisappearing(ctx context.Context, opts DisappearingOpts) error {
	jid, err := parseRecipient(opts.To)
	if err != nil {
		return err
	}
	client, err := m.getClient()
	if err != nil {
		return err
	}
	return client.SetDisappearingTimer(ctx, jid, opts.Timer, time.Now())
}
