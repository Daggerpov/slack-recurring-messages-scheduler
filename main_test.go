package main

import (
	"testing"
	"time"

	"github.com/slack-go/slack"
)

func TestGenerateGroupLabel(t *testing.T) {
	tests := []struct {
		index int
		want  string
	}{
		{0, "A"},
		{1, "B"},
		{25, "Z"},
		{26, "A2"},
		{27, "B2"},
		{51, "Z2"},
		{52, "A3"},
		{77, "Z3"},
		{78, "A4"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := generateGroupLabel(tt.index)
			if got != tt.want {
				t.Errorf("generateGroupLabel(%d) = %s, want %s", tt.index, got, tt.want)
			}
		})
	}
}

func TestParseGroupLabel(t *testing.T) {
	tests := []struct {
		label   string
		want    int
		wantOK  bool
	}{
		// Valid labels
		{"A", 0, true},
		{"B", 1, true},
		{"Z", 25, true},
		{"a", 0, true}, // lowercase
		{"A2", 26, true},
		{"B2", 27, true},
		{"Z2", 51, true},
		{"A3", 52, true},
		{"z3", 77, true}, // lowercase with number

		// Invalid labels
		{"", 0, false},
		{"1", 0, false},
		{"AA", 0, false},  // double letter without number
		{"A1", 0, false},  // A1 is invalid (numbers start at 2)
		{"A0", 0, false},
		{"AB", 0, false},
		{"1A", 0, false},
		{"@", 0, false},
		{"A-1", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			got, ok := parseGroupLabel(tt.label)
			if ok != tt.wantOK {
				t.Errorf("parseGroupLabel(%q) ok = %v, want %v", tt.label, ok, tt.wantOK)
			}
			if ok && got != tt.want {
				t.Errorf("parseGroupLabel(%q) = %d, want %d", tt.label, got, tt.want)
			}
		})
	}
}

func TestIsGroupLabel(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		// Valid group labels
		{"A", true},
		{"B", true},
		{"Z", true},
		{"a", true},
		{"A2", true},
		{"B2", true},
		{"Z99", true},

		// Invalid group labels
		{"", false},
		{"1", false},
		{"12", false},
		{"A1", false},  // 1 is invalid (must be >= 2)
		{"AA", false},
		{"AB", false},
		{"1A", false},
		{"A-2", false},
		{"A.2", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isGroupLabel(tt.input)
			if got != tt.want {
				t.Errorf("isGroupLabel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestBuildIndexedMessages(t *testing.T) {
	// Test building indexed messages from Slack messages
	now := time.Now()
	messages := []slack.ScheduledMessage{
		{ID: "msg3", Channel: "C123", Text: "Third", PostAt: int(now.Add(2 * time.Hour).Unix())},
		{ID: "msg1", Channel: "C123", Text: "First", PostAt: int(now.Unix())},
		{ID: "msg2", Channel: "C456", Text: "Second", PostAt: int(now.Add(1 * time.Hour).Unix())},
	}

	channelMap := map[string]string{
		"C123": "general",
		"C456": "random",
	}

	indexed := buildIndexedMessages(messages, channelMap)

	// Should be sorted by PostAt time
	if len(indexed) != 3 {
		t.Fatalf("expected 3 indexed messages, got %d", len(indexed))
	}

	// Verify sorting (first should be earliest)
	if indexed[0].SlackID != "msg1" {
		t.Errorf("first message should be msg1, got %s", indexed[0].SlackID)
	}
	if indexed[1].SlackID != "msg2" {
		t.Errorf("second message should be msg2, got %s", indexed[1].SlackID)
	}
	if indexed[2].SlackID != "msg3" {
		t.Errorf("third message should be msg3, got %s", indexed[2].SlackID)
	}

	// Verify 1-based indexing
	if indexed[0].Index != 1 {
		t.Errorf("first index should be 1, got %d", indexed[0].Index)
	}
	if indexed[2].Index != 3 {
		t.Errorf("third index should be 3, got %d", indexed[2].Index)
	}

	// Verify channel name resolution
	if indexed[0].ChannelName != "general" {
		t.Errorf("channel name should be 'general', got %s", indexed[0].ChannelName)
	}
	if indexed[1].ChannelName != "random" {
		t.Errorf("channel name should be 'random', got %s", indexed[1].ChannelName)
	}
}

func TestBuildIndexedMessages_UnknownChannel(t *testing.T) {
	messages := []slack.ScheduledMessage{
		{ID: "msg1", Channel: "C999", Text: "Test", PostAt: int(time.Now().Unix())},
	}

	channelMap := map[string]string{} // Empty map

	indexed := buildIndexedMessages(messages, channelMap)

	// Should fall back to channel ID
	if indexed[0].ChannelName != "C999" {
		t.Errorf("channel name should fall back to ID 'C999', got %s", indexed[0].ChannelName)
	}
}

func TestGroupMessages(t *testing.T) {
	messages := []*IndexedMessage{
		{Index: 1, Text: "Hello", SlackID: "m1"},
		{Index: 2, Text: "World", SlackID: "m2"},
		{Index: 3, Text: "Hello", SlackID: "m3"},
		{Index: 4, Text: "Hello", SlackID: "m4"},
		{Index: 5, Text: "World", SlackID: "m5"},
	}

	groups := groupMessages(messages)

	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}

	// First group (Hello) - should have label A
	if groups[0].Label != "A" {
		t.Errorf("first group label should be 'A', got %s", groups[0].Label)
	}
	if groups[0].Text != "Hello" {
		t.Errorf("first group text should be 'Hello', got %s", groups[0].Text)
	}
	if len(groups[0].Messages) != 3 {
		t.Errorf("first group should have 3 messages, got %d", len(groups[0].Messages))
	}

	// Second group (World) - should have label B
	if groups[1].Label != "B" {
		t.Errorf("second group label should be 'B', got %s", groups[1].Label)
	}
	if groups[1].Text != "World" {
		t.Errorf("second group text should be 'World', got %s", groups[1].Text)
	}
	if len(groups[1].Messages) != 2 {
		t.Errorf("second group should have 2 messages, got %d", len(groups[1].Messages))
	}

	// Verify group labels are assigned to individual messages
	for _, msg := range groups[0].Messages {
		if msg.GroupLabel != "A" {
			t.Errorf("message %d should have group label 'A', got %s", msg.Index, msg.GroupLabel)
		}
	}
	for _, msg := range groups[1].Messages {
		if msg.GroupLabel != "B" {
			t.Errorf("message %d should have group label 'B', got %s", msg.Index, msg.GroupLabel)
		}
	}
}

func TestGroupMessages_SingleMessage(t *testing.T) {
	messages := []*IndexedMessage{
		{Index: 1, Text: "Solo", SlackID: "m1"},
	}

	groups := groupMessages(messages)

	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if len(groups[0].Messages) != 1 {
		t.Errorf("group should have 1 message, got %d", len(groups[0].Messages))
	}
}

func TestGroupMessages_EmptyInput(t *testing.T) {
	groups := groupMessages([]*IndexedMessage{})

	if len(groups) != 0 {
		t.Errorf("expected 0 groups for empty input, got %d", len(groups))
	}
}

func TestGroupMessages_ManyGroups(t *testing.T) {
	// Create 30 messages with unique text to test group label generation beyond Z
	var messages []*IndexedMessage
	for i := 0; i < 30; i++ {
		messages = append(messages, &IndexedMessage{
			Index:   i + 1,
			Text:    string(rune('a' + i)), // unique text for each
			SlackID: "m",
		})
	}

	groups := groupMessages(messages)

	if len(groups) != 30 {
		t.Fatalf("expected 30 groups, got %d", len(groups))
	}

	// Check labels wrap correctly
	if groups[0].Label != "A" {
		t.Errorf("group 0 label should be 'A', got %s", groups[0].Label)
	}
	if groups[25].Label != "Z" {
		t.Errorf("group 25 label should be 'Z', got %s", groups[25].Label)
	}
	if groups[26].Label != "A2" {
		t.Errorf("group 26 label should be 'A2', got %s", groups[26].Label)
	}
	if groups[29].Label != "D2" {
		t.Errorf("group 29 label should be 'D2', got %s", groups[29].Label)
	}
}

// Roundtrip test: generate label then parse it back
func TestGroupLabel_Roundtrip(t *testing.T) {
	for i := 0; i < 100; i++ {
		label := generateGroupLabel(i)
		parsed, ok := parseGroupLabel(label)
		if !ok {
			t.Errorf("failed to parse generated label %q for index %d", label, i)
		}
		if parsed != i {
			t.Errorf("roundtrip failed: index %d -> label %q -> parsed %d", i, label, parsed)
		}
	}
}
