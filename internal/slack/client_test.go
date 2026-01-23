package slack

import (
	"fmt"
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

// TestConvertChannelLinks tests the conversion of #channel-name to Slack API format
func TestConvertChannelLinks(t *testing.T) {
	// Mock channel lookup function
	mockLookup := func(name string) (string, error) {
		channels := map[string]string{
			"general":  "C1234567890",
			"meetings": "C0987654321",
			"dev-team": "C1111111111",
			"test_ch":  "C2222222222",
		}
		if id, ok := channels[name]; ok {
			return id, nil
		}
		return "", fmt.Errorf("channel not found: %s", name)
	}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "convert #general",
			input: "Check out #general for updates",
			want:  "Check out <#C1234567890|general> for updates",
		},
		{
			name:  "convert #meetings at start",
			input: "#meetings is where we meet",
			want:  "<#C0987654321|meetings> is where we meet",
		},
		{
			name:  "convert channel with hyphen",
			input: "Join #dev-team for discussions",
			want:  "Join <#C1111111111|dev-team> for discussions",
		},
		{
			name:  "convert channel with underscore",
			input: "See #test_ch for tests",
			want:  "See <#C2222222222|test_ch> for tests",
		},
		{
			name:  "multiple channel links",
			input: "#general and #meetings are important",
			want:  "<#C1234567890|general> and <#C0987654321|meetings> are important",
		},
		{
			name:  "unknown channel preserved",
			input: "Check #unknown-channel for info",
			want:  "Check #unknown-channel for info",
		},
		{
			name:  "don't convert numbers only",
			input: "Issue #123 is fixed",
			want:  "Issue #123 is fixed",
		},
		{
			name:  "don't convert in email/URL context",
			input: "Visit example.com#general for info",
			want:  "Visit example.com#general for info",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "no channel links",
			input: "Hello team, this is a regular message",
			want:  "Hello team, this is a regular message",
		},
		{
			name:  "channel at end of message",
			input: "Please post in #general",
			want:  "Please post in <#C1234567890|general>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertChannelLinks(tt.input, mockLookup)
			if got != tt.want {
				t.Errorf("ConvertChannelLinks(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestConvertChannelLinksBack tests the reverse conversion from Slack API format to human-readable
func TestConvertChannelLinksBack(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "convert <#ID|name> format",
			input: "Check out <#C1234567890|general> for updates",
			want:  "Check out #general for updates",
		},
		{
			name:  "convert <#ID> format (no name)",
			input: "Check out <#C1234567890> for updates",
			want:  "Check out #C1234567890 for updates",
		},
		{
			name:  "multiple channel links",
			input: "<#C123|general> and <#C456|meetings> are important",
			want:  "#general and #meetings are important",
		},
		{
			name:  "no channel links",
			input: "Hello team, this is a regular message",
			want:  "Hello team, this is a regular message",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "channel with hyphen in name",
			input: "Join <#C111|dev-team> for discussions",
			want:  "Join #dev-team for discussions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertChannelLinksBack(tt.input)
			if got != tt.want {
				t.Errorf("ConvertChannelLinksBack(%q) = %q, want %q", tt.input, got, tt.want)
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
