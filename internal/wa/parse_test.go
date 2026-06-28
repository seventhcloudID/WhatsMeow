package wa

import (
	"testing"

	"go.mau.fi/whatsmeow/types"
)

func TestParseRecipient(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
		user    string
	}{
		{"6281234567890", false, "6281234567890"},
		{"+6281234567890", false, "6281234567890"},
		{"6281234567890@s.whatsapp.net", false, "6281234567890"},
		{"", true, ""},
		{"   ", true, ""},
		{"invalid@bad", false, "invalid"},
	}

	for _, tt := range tests {
		jid, err := parseRecipient(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseRecipient(%q) expected error", tt.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseRecipient(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if jid.User != tt.user {
			t.Errorf("parseRecipient(%q) user = %q, want %q", tt.input, jid.User, tt.user)
		}
		if jid.Server != types.DefaultUserServer && jid.Server != types.GroupServer {
			// personal JID should use default server when no @ provided
			if !containsAt(tt.input) && jid.Server != types.DefaultUserServer {
				t.Errorf("parseRecipient(%q) server = %q", tt.input, jid.Server)
			}
		}
	}
}

func containsAt(s string) bool {
	for _, c := range s {
		if c == '@' {
			return true
		}
	}
	return false
}
