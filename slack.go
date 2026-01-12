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
	// Slack API expects Unix timestamp as string (UTC)
	// Convert local time to UTC for the API call
	postAtUTC := postAt.UTC()
	postAtUnix := postAtUTC.Unix()
	
	respChannel, scheduledTime, err := c.api.ScheduleMessage(
		channel,
		fmt.Sprintf("%d", postAtUnix),
		slack.MsgOptionText(message, false),
		slack.MsgOptionAsUser(true),
	)
	if err != nil {
		return "", fmt.Errorf("failed to schedule message: %w", err)
	}
	
	// Log the scheduling result
	fmt.Printf("Scheduled message for: %s (UTC: %s) in channel: %s\n", 
		postAt.Format("2006-01-02 15:04 MST"), 
		postAtUTC.Format("2006-01-02 15:04 UTC"),
		respChannel)
	
	if scheduledTime != "" {
		fmt.Printf("Scheduled message timestamp: %s\n", scheduledTime)
	}
	
	// Return the scheduled timestamp (or postAt timestamp if empty) as identifier
	if scheduledTime != "" {
		return scheduledTime, nil
	}
	// Return the postAt timestamp as a fallback identifier
	return fmt.Sprintf("%d", postAtUnix), nil
}

// ListScheduledMessages lists all scheduled messages, optionally filtered by channel
func (c *SlackClient) ListScheduledMessages(channelID string) ([]slack.ScheduledMessage, error) {
	params := &slack.GetScheduledMessagesParameters{
		Limit: 100,
	}
	if channelID != "" {
		params.Channel = channelID
	}
	
	messages, _, err := c.api.GetScheduledMessages(params)
	if err != nil {
		return nil, fmt.Errorf("failed to list scheduled messages: %w", err)
	}
	
	return messages, nil
}

// DeleteScheduledMessage deletes a scheduled message by its ID
func (c *SlackClient) DeleteScheduledMessage(channelID, scheduledMsgID string) error {
	_, err := c.api.DeleteScheduledMessage(&slack.DeleteScheduledMessageParameters{
		Channel:            channelID,
		ScheduledMessageID: scheduledMsgID,
		AsUser:             true,
	})
	if err != nil {
		return fmt.Errorf("failed to delete scheduled message: %w", err)
	}
	return nil
}

// ValidateCredentials checks if the token is valid by testing auth
func (c *SlackClient) ValidateCredentials() error {
	resp, err := c.api.AuthTest()
	if err != nil {
		return fmt.Errorf("invalid credentials: %w", err)
	}
	
	// Print auth info for debugging
	fmt.Printf("  Authenticated as: %s\n", resp.User)
	fmt.Printf("  Team: %s\n", resp.Team)
	if resp.BotID != "" {
		fmt.Printf("  ⚠️  WARNING: This is a BOT token (Bot ID: %s)\n", resp.BotID)
		fmt.Printf("     Scheduled messages from bot tokens WON'T appear in your Slack UI!\n")
		fmt.Printf("     Use a User OAuth Token (xoxp-...) instead of a Bot Token (xoxb-...)\n")
	} else {
		fmt.Printf("  Token type: User token ✓\n")
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

// GetChannelName resolves a channel ID to its human-readable name
func (c *SlackClient) GetChannelName(channelID string) (string, error) {
	// List channels to find the name
	channels, _, err := c.api.GetConversations(&slack.GetConversationsParameters{
		Types: []string{"public_channel", "private_channel"},
		Limit: 1000,
	})
	if err != nil {
		return "", fmt.Errorf("failed to list channels: %w", err)
	}

	for _, ch := range channels {
		if ch.ID == channelID {
			return ch.Name, nil
		}
	}

	// Return the ID if we can't find the name
	return channelID, nil
}

// GetChannelNameMap returns a map of channel IDs to names
func (c *SlackClient) GetChannelNameMap() (map[string]string, error) {
	channels, _, err := c.api.GetConversations(&slack.GetConversationsParameters{
		Types: []string{"public_channel", "private_channel"},
		Limit: 1000,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list channels: %w", err)
	}

	nameMap := make(map[string]string)
	for _, ch := range channels {
		nameMap[ch.ID] = ch.Name
	}
	return nameMap, nil
}
