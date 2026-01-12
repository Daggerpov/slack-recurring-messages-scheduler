package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/daggerpov/slack-recurring-messages-scheduler/internal/types"
)

const (
	CredentialsFileName = ".slack-scheduler-credentials.json"
)

// LoadCredentials loads credentials from the config file in the current directory
func LoadCredentials() (*types.Credentials, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("could not determine current directory: %w", err)
	}

	path := filepath.Join(cwd, CredentialsFileName)
	creds, err := LoadCredentialsFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("credentials file not found at %s\n\n"+
			"Create it with your Slack token:\n"+
			"{\n  \"token\": \"xoxp-your-user-token-here\"\n}\n\n"+
			"To get a token, create a Slack app at https://api.slack.com/apps and add these scopes:\n"+
			"- chat:write (to send messages)\n"+
			"- channels:read (to resolve channel names)\n"+
			"- groups:read (for private channels)\n", path)
	}

	return creds, nil
}

func LoadCredentialsFromFile(path string) (*types.Credentials, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var creds types.Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("failed to parse credentials file: %w", err)
	}

	if creds.Token == "" {
		return nil, fmt.Errorf("token is empty in credentials file")
	}

	return &creds, nil
}

// CreateTemplateCredentials creates a template credentials file in the current directory
func CreateTemplateCredentials() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not determine current directory: %w", err)
	}

	path := filepath.Join(cwd, CredentialsFileName)

	// Don't overwrite existing file
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("credentials file already exists at %s", path)
	}

	template := types.Credentials{
		Token: "xoxp-your-user-token-here",
	}

	data, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write credentials file: %w", err)
	}

	fmt.Printf("Created credentials template at: %s\n", path)
	fmt.Println("Edit this file and replace the token with your actual Slack user token.")
	return nil
}
