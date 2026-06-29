package wa

import (
	"context"
	"encoding/base64"
	"fmt"

	"google.golang.org/protobuf/proto"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
)

type DownloadMediaOpts struct {
	URL        string `json:"url"`
	DirectPath string `json:"direct_path"`
	MediaKey   string `json:"media_key"` // base64
	Mimetype   string `json:"mimetype"`
	FileSHA256 string `json:"file_sha256"` // base64
	FileEncSHA256 string `json:"file_enc_sha256"` // base64
	FileLength uint64 `json:"file_length"`
	MediaType  string `json:"media_type"` // image, video, audio, document
}

func (m *Manager) DownloadMedia(ctx context.Context, opts DownloadMediaOpts) ([]byte, string, error) {
	client, err := m.getClient()
	if err != nil {
		return nil, "", err
	}
	mediaKey, err := base64.StdEncoding.DecodeString(opts.MediaKey)
	if err != nil {
		return nil, "", fmt.Errorf("media_key invalid: %w", err)
	}
	fileSHA, err := base64.StdEncoding.DecodeString(opts.FileSHA256)
	if err != nil {
		return nil, "", fmt.Errorf("file_sha256 invalid: %w", err)
	}
	fileEncSHA, err := base64.StdEncoding.DecodeString(opts.FileEncSHA256)
	if err != nil {
		return nil, "", fmt.Errorf("file_enc_sha256 invalid: %w", err)
	}
	fl := opts.FileLength
	msg := &waProto.Message{
		ImageMessage: &waProto.ImageMessage{
			URL:           proto.String(opts.URL),
			DirectPath:    proto.String(opts.DirectPath),
			MediaKey:      mediaKey,
			Mimetype:      proto.String(opts.Mimetype),
			FileSHA256:    fileSHA,
			FileEncSHA256: fileEncSHA,
			FileLength:    proto.Uint64(fl),
		},
	}
	var downloadable whatsmeow.DownloadableMessage = msg.GetImageMessage()
	switch opts.MediaType {
	case "video":
		msg = &waProto.Message{VideoMessage: &waProto.VideoMessage{
			URL: proto.String(opts.URL), DirectPath: proto.String(opts.DirectPath), MediaKey: mediaKey,
			Mimetype: proto.String(opts.Mimetype), FileSHA256: fileSHA, FileEncSHA256: fileEncSHA,
			FileLength: proto.Uint64(fl),
		}}
		downloadable = msg.GetVideoMessage()
	case "audio":
		msg = &waProto.Message{AudioMessage: &waProto.AudioMessage{
			URL: proto.String(opts.URL), DirectPath: proto.String(opts.DirectPath), MediaKey: mediaKey,
			Mimetype: proto.String(opts.Mimetype), FileSHA256: fileSHA, FileEncSHA256: fileEncSHA,
			FileLength: proto.Uint64(fl),
		}}
		downloadable = msg.GetAudioMessage()
	case "document":
		msg = &waProto.Message{DocumentMessage: &waProto.DocumentMessage{
			URL: proto.String(opts.URL), DirectPath: proto.String(opts.DirectPath), MediaKey: mediaKey,
			Mimetype: proto.String(opts.Mimetype), FileSHA256: fileSHA, FileEncSHA256: fileEncSHA,
			FileLength: proto.Uint64(fl),
		}}
		downloadable = msg.GetDocumentMessage()
	}
	data, err := client.Download(ctx, downloadable)
	if err != nil {
		return nil, "", fmt.Errorf("download: %w", err)
	}
	mime := opts.Mimetype
	if mime == "" {
		mime = "application/octet-stream"
	}
	return data, mime, nil
}
