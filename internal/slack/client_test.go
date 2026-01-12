package slack

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	token := "xoxp-test-token"
	client := NewClient(token)

	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.api == nil {
		t.Error("Client.api should not be nil")
	}
}

// TestGetChannelID_AlreadyID tests that channel IDs are returned as-is
// This tests the logic without making API calls
func TestGetChannelID_AlreadyID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"public channel ID", "C1234567890", "C1234567890"},
		{"private channel ID", "G1234567890", "G1234567890"},
		{"DM channel ID", "D1234567890", "D1234567890"},
	}

	client := NewClient("fake-token")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// GetChannelID returns the ID directly if it looks like an ID
			// (starts with C, G, or D)
			got, err := client.GetChannelID(tt.input)
			if err != nil {
				t.Fatalf("GetChannelID() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("GetChannelID() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetChannelID_ChannelNameResolution documents behavior that requires API calls.
// These tests would need a mock Slack API interface to test properly.
//
// Note: There's a subtle behavior in GetChannelID where the ID check happens
// before the hash stripping. This means "#general" gets the hash stripped
// then looks up "general", but "#C123..." also strips the hash and then
// tries to look up "C123..." (which would require an API call).
//
// For proper testing, we'd refactor to use an interface:
//
//   type SlackAPI interface {
//       GetConversations(params *GetConversationsParameters) ([]Channel, string, error)
//       // ...
//   }
//
// Then inject a mock in tests.

// Note: Testing functions that require Slack API calls (ValidateCredentials,
// ListScheduledMessages, etc.) would require either:
// 1. A mock/fake Slack client interface
// 2. Integration tests with a real Slack token
//
// For a production app, consider refactoring Client to use an interface:
//
// type SlackAPI interface {
//     AuthTest() (*slack.AuthTestResponse, error)
//     PostMessage(channel string, options ...slack.MsgOption) (string, string, error)
//     // ... etc
// }
//
// This would allow injecting a mock for testing.

// Benchmark for channel ID detection (since it's called frequently)
func BenchmarkGetChannelID_AlreadyID(b *testing.B) {
	client := NewClient("fake-token")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		client.GetChannelID("C1234567890")
	}
}
