package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/daggerpov/slack-recurring-messages-scheduler/internal/types"
)

func TestLoadCredentials_ValidFile(t *testing.T) {
	// Create temp directory and credentials file
	tmpDir := t.TempDir()
	credsPath := filepath.Join(tmpDir, CredentialsFileName)

	creds := types.Credentials{Token: "xoxp-test-token-12345"}
	data, err := json.Marshal(creds)
	if err != nil {
		t.Fatalf("failed to marshal test credentials: %v", err)
	}

	if err := os.WriteFile(credsPath, data, 0600); err != nil {
		t.Fatalf("failed to write test credentials: %v", err)
	}

	// Change to temp directory and load credentials
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	loaded, err := LoadCredentials()
	if err != nil {
		t.Fatalf("LoadCredentials() error = %v", err)
	}

	if loaded.Token != creds.Token {
		t.Errorf("loaded token = %s, want %s", loaded.Token, creds.Token)
	}
}

func TestLoadCredentials_MissingFile(t *testing.T) {
	// Create empty temp directory
	tmpDir := t.TempDir()

	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	_, err := LoadCredentials()
	if err == nil {
		t.Error("LoadCredentials() expected error for missing file, got nil")
	}
}

func TestLoadCredentials_EmptyToken(t *testing.T) {
	tmpDir := t.TempDir()
	credsPath := filepath.Join(tmpDir, CredentialsFileName)

	// Write credentials with empty token
	creds := types.Credentials{Token: ""}
	data, _ := json.Marshal(creds)
	os.WriteFile(credsPath, data, 0600)

	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tmpDir)

	_, err := LoadCredentials()
	if err == nil {
		t.Error("LoadCredentials() expected error for empty token, got nil")
	}
}

func TestLoadCredentials_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	credsPath := filepath.Join(tmpDir, CredentialsFileName)

	// Write invalid JSON
	os.WriteFile(credsPath, []byte("not valid json{"), 0600)

	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tmpDir)

	_, err := LoadCredentials()
	if err == nil {
		t.Error("LoadCredentials() expected error for invalid JSON, got nil")
	}
}

func TestCreateTemplateCredentials_Success(t *testing.T) {
	tmpDir := t.TempDir()

	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tmpDir)

	err := CreateTemplateCredentials()
	if err != nil {
		t.Fatalf("CreateTemplateCredentials() error = %v", err)
	}

	// Verify file was created
	credsPath := filepath.Join(tmpDir, CredentialsFileName)
	data, err := os.ReadFile(credsPath)
	if err != nil {
		t.Fatalf("failed to read created credentials file: %v", err)
	}

	var creds types.Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		t.Fatalf("failed to parse created credentials file: %v", err)
	}

	if creds.Token != "xoxp-your-user-token-here" {
		t.Errorf("template token = %s, want placeholder", creds.Token)
	}

	// Verify file permissions (should be 0600)
	info, err := os.Stat(credsPath)
	if err != nil {
		t.Fatalf("failed to stat credentials file: %v", err)
	}

	// On Unix, check permissions (skip on Windows)
	if info.Mode().Perm() != 0600 {
		t.Errorf("file permissions = %v, want 0600", info.Mode().Perm())
	}
}

func TestCreateTemplateCredentials_FileExists(t *testing.T) {
	tmpDir := t.TempDir()
	credsPath := filepath.Join(tmpDir, CredentialsFileName)

	// Create existing file
	os.WriteFile(credsPath, []byte(`{"token":"existing"}`), 0600)

	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tmpDir)

	err := CreateTemplateCredentials()
	if err == nil {
		t.Error("CreateTemplateCredentials() expected error when file exists, got nil")
	}

	// Verify existing file was not overwritten
	data, _ := os.ReadFile(credsPath)
	var creds types.Credentials
	json.Unmarshal(data, &creds)
	if creds.Token != "existing" {
		t.Error("existing credentials file was overwritten")
	}
}

func TestLoadCredentialsFromFile_ValidTokenFormats(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{"user token", "xoxp-123-456-789-abc"},
		{"bot token", "xoxb-123-456-789-def"},
		{"arbitrary token", "some-other-token-format"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			credsPath := filepath.Join(tmpDir, "test-creds.json")

			data, _ := json.Marshal(types.Credentials{Token: tt.token})
			os.WriteFile(credsPath, data, 0600)

			creds, err := LoadCredentialsFromFile(credsPath)
			if err != nil {
				t.Fatalf("LoadCredentialsFromFile() error = %v", err)
			}

			if creds.Token != tt.token {
				t.Errorf("token = %s, want %s", creds.Token, tt.token)
			}
		})
	}
}
