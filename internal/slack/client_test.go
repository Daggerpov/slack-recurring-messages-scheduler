package slack

import (
	"testing"
)

// TestConvertMentions tests the conversion of human-readable mentions to Slack API format
func TestConvertMentions(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "convert @channel",
			input: "Hello @channel, please check this",
			want:  "Hello <!channel>, please check this",
		},
		{
			name:  "convert @here",
			input: "@here urgent meeting now",
			want:  "<!here> urgent meeting now",
		},
		{
			name:  "convert @everyone",
			input: "@everyone please read this",
			want:  "<!everyone> please read this",
		},
		{
			name:  "convert multiple mentions",
			input: "@channel and @here please pay attention",
			want:  "<!channel> and <!here> please pay attention",
		},
		{
			name:  "case insensitive @CHANNEL",
			input: "@CHANNEL important!",
			want:  "<!channel> important!",
		},
		{
			name:  "case insensitive @Here",
			input: "@Here please respond",
			want:  "<!here> please respond",
		},
		{
			name:  "don't convert @channels (plural)",
			input: "Check the @channels setting",
			want:  "Check the @channels setting",
		},
		{
			name:  "don't convert email-like patterns",
			input: "Email me at user@here.com",
			want:  "Email me at user@here.com",
		},
		{
			name:  "no mentions - pass through",
			input: "Hello team, this is a regular message",
			want:  "Hello team, this is a regular message",
		},
		{
			name:  "preserve other @ mentions",
			input: "@channel @john please review",
			want:  "<!channel> @john please review",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "@channel at end of message",
			input: "Please respond @channel",
			want:  "Please respond <!channel>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertMentions(tt.input)
			if got != tt.want {
				t.Errorf("ConvertMentions(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestConvertMentionsBack tests the reverse conversion from Slack API format to human-readable
func TestConvertMentionsBack(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "convert <!channel>",
			input: "Hello <!channel>, please check this",
			want:  "Hello @channel, please check this",
		},
		{
			name:  "convert <!here>",
			input: "<!here> urgent meeting now",
			want:  "@here urgent meeting now",
		},
		{
			name:  "convert <!everyone>",
			input: "<!everyone> please read this",
			want:  "@everyone please read this",
		},
		{
			name:  "convert multiple mentions",
			input: "<!channel> and <!here> please pay attention",
			want:  "@channel and @here please pay attention",
		},
		{
			name:  "no special mentions - pass through",
			input: "Hello team, this is a regular message",
			want:  "Hello team, this is a regular message",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertMentionsBack(tt.input)
			if got != tt.want {
				t.Errorf("ConvertMentionsBack(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestMentionConversionRoundTrip tests that conversion is reversible
func TestMentionConversionRoundTrip(t *testing.T) {
	testCases := []string{
		"@channel please respond",
		"@here urgent",
		"@everyone announcement",
		"@channel and @here and @everyone",
		"No mentions here",
	}

	for _, original := range testCases {
		t.Run(original, func(t *testing.T) {
			converted := ConvertMentions(original)
			backToOriginal := ConvertMentionsBack(converted)
			if backToOriginal != original {
				t.Errorf("Round trip failed: %q -> %q -> %q", original, converted, backToOriginal)
			}
		})
	}
}

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
