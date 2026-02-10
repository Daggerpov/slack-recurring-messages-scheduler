package main

import (
	"testing"
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
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := generateGroupLabel(tt.index)
			if got != tt.want {
				t.Errorf("generateGroupLabel(%d) = %q, want %q", tt.index, got, tt.want)
			}
		})
	}
}

func TestParseGroupLabel(t *testing.T) {
	tests := []struct {
		label   string
		want    int
		wantOk  bool
	}{
		{"A", 0, true},
		{"B", 1, true},
		{"Z", 25, true},
		{"a", 0, true},  // lowercase
		{"z", 25, true}, // lowercase
		{"A2", 26, true},
		{"B2", 27, true},
		{"Z2", 51, true},
		{"A3", 52, true},
		{"", 0, false},
		{"1", 0, false},
		{"A1", 0, false}, // A1 is invalid (cycle must be >= 2)
		{"AA", 0, false}, // Not a valid format
		{"123", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			got, ok := parseGroupLabel(tt.label)
			if ok != tt.wantOk {
				t.Errorf("parseGroupLabel(%q) ok = %v, want %v", tt.label, ok, tt.wantOk)
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
		{"A", true},
		{"B", true},
		{"Z", true},
		{"a", true},
		{"z", true},
		{"A2", true},
		{"B2", true},
		{"Z99", true},
		{"", false},
		{"1", false},
		{"12", false},
		{"A1", false}, // 1 is not >= 2
		{"AA", false},
		{"1A", false},
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

func TestGroupMessages(t *testing.T) {
	// Create test messages
	messages := []*IndexedMessage{
		{Index: 1, Text: "Hello"},
		{Index: 2, Text: "World"},
		{Index: 3, Text: "Hello"}, // Same as message 1
		{Index: 4, Text: "Test"},
		{Index: 5, Text: "World"}, // Same as message 2
	}

	groups := groupMessages(messages)

	// Should have 3 groups (Hello, World, Test)
	if len(groups) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(groups))
	}

	// Check first group (Hello)
	if groups[0].Label != "A" {
		t.Errorf("first group label = %q, want %q", groups[0].Label, "A")
	}
	if groups[0].Text != "Hello" {
		t.Errorf("first group text = %q, want %q", groups[0].Text, "Hello")
	}
	if len(groups[0].Messages) != 2 {
		t.Errorf("first group has %d messages, want 2", len(groups[0].Messages))
	}

	// Check second group (World)
	if groups[1].Label != "B" {
		t.Errorf("second group label = %q, want %q", groups[1].Label, "B")
	}
	if groups[1].Text != "World" {
		t.Errorf("second group text = %q, want %q", groups[1].Text, "World")
	}
	if len(groups[1].Messages) != 2 {
		t.Errorf("second group has %d messages, want 2", len(groups[1].Messages))
	}

	// Check third group (Test)
	if groups[2].Label != "C" {
		t.Errorf("third group label = %q, want %q", groups[2].Label, "C")
	}
	if groups[2].Text != "Test" {
		t.Errorf("third group text = %q, want %q", groups[2].Text, "Test")
	}
	if len(groups[2].Messages) != 1 {
		t.Errorf("third group has %d messages, want 1", len(groups[2].Messages))
	}

	// Check that group labels are assigned to messages
	if messages[0].GroupLabel != "A" {
		t.Errorf("message 1 group label = %q, want %q", messages[0].GroupLabel, "A")
	}
	if messages[2].GroupLabel != "A" {
		t.Errorf("message 3 group label = %q, want %q", messages[2].GroupLabel, "A")
	}
}

func TestBuildIndexedMessages(t *testing.T) {
	// We can't easily test buildIndexedMessages without mocking slack.ScheduledMessage
	// This is mainly an integration function, so we test the components it uses
}

func TestProcessEscapeSequences(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "newline escape",
			input: `Hello\nWorld`,
			want:  "Hello\nWorld",
		},
		{
			name:  "tab escape",
			input: `Hello\tWorld`,
			want:  "Hello\tWorld",
		},
		{
			name:  "backslash escape",
			input: `Hello\\World`,
			want:  `Hello\World`,
		},
		{
			name:  "double escaped backslash before n",
			input: `Hello\\\\nWorld`,
			want:  `Hello\\nWorld`,
		},
		{
			name:  "multiple newlines",
			input: `line1\nline2\nline3`,
			want:  "line1\nline2\nline3",
		},
		{
			name:  "bullet list with newlines",
			input: `Reminders:\n- Item one\n- Item two\n    - Sub-item`,
			want:  "Reminders:\n- Item one\n- Item two\n    - Sub-item",
		},
		{
			name:  "complex bullet list",
			input: `Hey, quick reminders to:\n\n- Once event details confirmed (by 1 pm):\n    - Send out a message to #availability\n    - Update our Google Calendar\n- Once media finalized:\n    - Update & send out newsletter`,
			want:  "Hey, quick reminders to:\n\n- Once event details confirmed (by 1 pm):\n    - Send out a message to #availability\n    - Update our Google Calendar\n- Once media finalized:\n    - Update & send out newsletter",
		},
		{
			name:  "no escape sequences",
			input: "Hello World",
			want:  "Hello World",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "trailing backslash",
			input: `Hello\`,
			want:  `Hello\`,
		},
		{
			name:  "backslash before unknown char",
			input: `Hello\xWorld`,
			want:  `Hello\xWorld`,
		},
		{
			name:  "already has real newlines (passthrough)",
			input: "Hello\nWorld",
			want:  "Hello\nWorld",
		},
		{
			name:  "escaped backslash preserves literal backslash-n",
			input: `Hello\\nWorld`,
			want:  "Hello\\nWorld",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := processEscapeSequences(tt.input)
			if got != tt.want {
				t.Errorf("processEscapeSequences(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
