package main

import (
	"fmt"
	"time"

	"github.com/slack-go/slack"
)

// SlackClient wraps the Slack API client
type SlackClient struct {
	api *slack.Client
}

// NewSlackClient creates a new Slack client with the given token
func NewSlackClient(token string) *SlackClient {
	return &SlackClient{
		api: slack.New(token),
	}
}

// SendMessage sends a message to the specified channel
func (c *SlackClient) SendMessage(channel, message string) error {
	_, _, err := c.api.PostMessage(
		channel,
		slack.MsgOptionText(message, false), // false = parse markdown/mentions
		slack.MsgOptionAsUser(true),         // Send as the authenticated user
	)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	return nil
}

// ScheduleMessage schedules a message to be sent at a specific time
func (c *SlackClient) ScheduleMessage(channel, message string, postAt time.Time) (string, error) {
	// Slack API expects Unix timestamp as string
	// Note: ScheduleMessage returns (channelID, scheduledTime, error)
	// For scheduled messages, the timestamp may be empty until the message is sent
	// The scheduled_message_id is in the API response but not exposed by the library
	respChannel, scheduledTime, err := c.api.ScheduleMessage(
		channel,
		fmt.Sprintf("%d", postAt.Unix()),
		slack.MsgOptionText(message, false),
		slack.MsgOptionAsUser(true),
	)
	if err != nil {
		return "", fmt.Errorf("failed to schedule message: %w", err)
	}
	
	// Log the scheduling result
	// Note: For scheduled messages, scheduledTime may be empty (this is normal)
	// The message is still scheduled even if the timestamp is empty
	if scheduledTime != "" {
		fmt.Printf("Scheduled message ID: %s in channel: %s\n", scheduledTime, respChannel)
	} else {
		// For scheduled messages, Slack may return empty timestamp
		// The message is still scheduled - you can verify in Slack with /schedule list
		fmt.Printf("Scheduled message for: %s in channel: %s\n", 
			postAt.Format("2006-01-02 15:04 MST"), respChannel)
		fmt.Printf("Note: Scheduled message ID not available until message is sent. Use '/schedule list' in Slack to verify.\n")
	}
	
	// Return the scheduled timestamp (or postAt timestamp if empty) as identifier
	if scheduledTime != "" {
		return scheduledTime, nil
	}
	// Return the postAt timestamp as a fallback identifier
	return fmt.Sprintf("%d", postAt.Unix()), nil
}

// ValidateCredentials checks if the token is valid by testing auth
func (c *SlackClient) ValidateCredentials() error {
	_, err := c.api.AuthTest()
	if err != nil {
		return fmt.Errorf("invalid credentials: %w", err)
	}
	return nil
}

// GetChannelID resolves a channel name to its ID
func (c *SlackClient) GetChannelID(channelName string) (string, error) {
	// If it already looks like an ID, return it
	if len(channelName) > 0 && (channelName[0] == 'C' || channelName[0] == 'D' || channelName[0] == 'G') {
		return channelName, nil
	}

	// Remove # prefix if present
	if len(channelName) > 0 && channelName[0] == '#' {
		channelName = channelName[1:]
	}

	// List channels to find the ID
	channels, _, err := c.api.GetConversations(&slack.GetConversationsParameters{
		Types: []string{"public_channel", "private_channel"},
		Limit: 1000,
	})
	if err != nil {
		return "", fmt.Errorf("failed to list channels: %w", err)
	}

	for _, ch := range channels {
		if ch.Name == channelName {
			return ch.ID, nil
		}
	}

	return "", fmt.Errorf("channel not found: %s", channelName)
}
