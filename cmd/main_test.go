package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGenerateGroupID(t *testing.T) {
	id1 := generateGroupID()
	// Small delay to ensure different timestamps
	time.Sleep(time.Millisecond)
	id2 := generateGroupID()

	if id1 == "" {
		t.Error("generateGroupID should not return empty string")
	}
	if id1 == id2 {
		t.Error("generateGroupID should generate unique IDs")
	}
}

func TestParseTime(t *testing.T) {
	tests := []struct {
		input    string
		wantHour int
		wantMin  int
	}{
		{"09:00", 9, 0},
		{"14:30", 14, 30},
		{"00:00", 0, 0},
		{"23:59", 23, 59},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			hour, min := parseTime(tt.input)
			if hour != tt.wantHour || min != tt.wantMin {
				t.Errorf("parseTime(%q) = (%d, %d), want (%d, %d)",
					tt.input, hour, min, tt.wantHour, tt.wantMin)
			}
		})
	}
}

func TestGroupsFileOperations(t *testing.T) {
	// Use a temp directory for testing
	tempDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tempDir)

	// Test loading from non-existent file
	groups, err := loadGroups()
	if err != nil {
		t.Errorf("loadGroups should not error on non-existent file: %v", err)
	}
	if len(groups.Groups) != 0 {
		t.Errorf("expected 0 groups, got %d", len(groups.Groups))
	}

	// Test saving a group
	testGroup := MessageGroup{
		ID:         "test123",
		Name:       "test-group",
		Channel:    "general",
		ChannelID:  "C123456",
		Message:    "Test message @channel",
		Interval:   "daily",
		SendTime:   "09:00",
		MessageIDs: []string{"MSG1", "MSG2", "MSG3"},
		CreatedAt:  time.Now(),
	}

	err = saveGroup(testGroup)
	if err != nil {
		t.Fatalf("saveGroup failed: %v", err)
	}

	// Verify file was created
	groupsPath := filepath.Join(tempDir, groupsFileName)
	if _, err := os.Stat(groupsPath); os.IsNotExist(err) {
		t.Error("groups file was not created")
	}

	// Test loading saved groups
	groups, err = loadGroups()
	if err != nil {
		t.Fatalf("loadGroups failed: %v", err)
	}

	if len(groups.Groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups.Groups))
	}

	loaded := groups.Groups[0]
	if loaded.ID != testGroup.ID {
		t.Errorf("ID mismatch: got %s, want %s", loaded.ID, testGroup.ID)
	}
	if loaded.Name != testGroup.Name {
		t.Errorf("Name mismatch: got %s, want %s", loaded.Name, testGroup.Name)
	}
	if loaded.Message != testGroup.Message {
		t.Errorf("Message mismatch: got %s, want %s", loaded.Message, testGroup.Message)
	}
	if len(loaded.MessageIDs) != len(testGroup.MessageIDs) {
		t.Errorf("MessageIDs length mismatch: got %d, want %d",
			len(loaded.MessageIDs), len(testGroup.MessageIDs))
	}
}

func TestRemoveMessageFromGroups(t *testing.T) {
	// Use a temp directory for testing
	tempDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tempDir)

	// Save a group with multiple messages
	testGroup := MessageGroup{
		ID:         "test123",
		Name:       "test-group",
		Channel:    "general",
		ChannelID:  "C123456",
		Message:    "Test",
		MessageIDs: []string{"MSG1", "MSG2", "MSG3"},
		CreatedAt:  time.Now(),
	}
	saveGroup(testGroup)

	// Remove a message
	removeMessageFromGroups("MSG2")

	// Verify it was removed
	groups, _ := loadGroups()
	if len(groups.Groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups.Groups))
	}

	msgIDs := groups.Groups[0].MessageIDs
	if len(msgIDs) != 2 {
		t.Errorf("expected 2 message IDs after removal, got %d", len(msgIDs))
	}

	// Verify MSG2 is not in the list
	for _, id := range msgIDs {
		if id == "MSG2" {
			t.Error("MSG2 should have been removed")
		}
	}
}

func TestUpdateMessageIDInGroups(t *testing.T) {
	// Use a temp directory for testing
	tempDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tempDir)

	// Save a group with messages
	testGroup := MessageGroup{
		ID:         "test123",
		Name:       "test-group",
		Channel:    "general",
		ChannelID:  "C123456",
		Message:    "Test",
		MessageIDs: []string{"OLD_ID"},
		CreatedAt:  time.Now(),
	}
	saveGroup(testGroup)

	// Update the message ID
	updateMessageIDInGroups("OLD_ID", "NEW_ID")

	// Verify it was updated
	groups, _ := loadGroups()
	if len(groups.Groups[0].MessageIDs) != 1 {
		t.Fatalf("expected 1 message ID, got %d", len(groups.Groups[0].MessageIDs))
	}

	if groups.Groups[0].MessageIDs[0] != "NEW_ID" {
		t.Errorf("expected message ID to be NEW_ID, got %s", groups.Groups[0].MessageIDs[0])
	}
}
