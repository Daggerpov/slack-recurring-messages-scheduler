package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/daggerpov/slack-recurring-messages-scheduler/internal/config"
	"github.com/daggerpov/slack-recurring-messages-scheduler/internal/scheduler"
	"github.com/daggerpov/slack-recurring-messages-scheduler/internal/slack"
	"github.com/daggerpov/slack-recurring-messages-scheduler/internal/types"
	"github.com/spf13/cobra"
)

const (
	groupsFileName = ".slack-scheduler-groups.json"
)

// MessageGroup represents a group of related scheduled messages
type MessageGroup struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Channel    string    `json:"channel"`
	ChannelID  string    `json:"channel_id"`
	Message    string    `json:"message"`
	Interval   string    `json:"interval"`
	Days       []string  `json:"days,omitempty"`
	SendTime   string    `json:"send_time"`
	MessageIDs []string  `json:"message_ids"`
	CreatedAt  time.Time `json:"created_at"`
}

// GroupsFile holds all message groups
type GroupsFile struct {
	Groups []MessageGroup `json:"groups"`
}

var (
	// CLI flags
	message     string
	channel     string
	startDate   string
	sendTime    string
	interval    string
	repeatCount int
	endDate     string
	days        string
	groupName   string

	// List command flags
	listChannel string

	// Delete command flags
	deleteChannel string
	deleteID      string
	deleteGroup   string
	deleteAll     bool

	// Modify command flags
	modifyGroup      string
	modifyID         string
	modifyChannel    string
	modifyMessage    string
	modifyTime       string
	modifyStartDate  string
	modifyInterval   string
	modifyDays       string
	modifyCount      int
	modifyEndDate    string
	modifyKeepFuture bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "./slack-scheduler",
		Short: "Schedule Slack messages to be sent at specific times",
		Long: `A CLI tool to schedule Slack messages with support for:
- One-time scheduled messages
- Recurring messages (daily, weekly, monthly)
- Specific days of the week for weekly schedules
- Full Slack formatting support (@mentions, emoji, etc.)

Messages are scheduled using your system's local timezone.

IMPORTANT: @channel, @here, and @everyone mentions are automatically converted
to the proper Slack API format to ensure notifications are sent.`,
		Example: `  # Send a one-time message
  ./slack-scheduler -m "Hello team!" -c general -d 2025-01-17 -t 14:00

  # Send weekly until end date (start date defaults to today)
  ./slack-scheduler -m "Weekly reminder!" -c general -t 14:00 -i weekly -e 2025-04-01

  # Send every Friday from today until end date
  ./slack-scheduler -m "TGIF!" -c general -t 14:00 -i weekly --days fri -e 2025-04-01

  # Send on Monday and Friday at 9am for 8 occurrences
  ./slack-scheduler -m "Standup time!" -c engineering -d 2025-01-13 -t 09:00 -i weekly -n 8 --days mon,fri

  # Send @channel notification that actually works
  ./slack-scheduler -m "@channel Don't forget standup!" -c general -d 2025-01-17 -t 09:00`,
		RunE: runSchedule,
	}

	// Required flags
	rootCmd.Flags().StringVarP(&message, "message", "m", "", "Message to send (supports @mentions, emoji, Slack formatting)")
	rootCmd.Flags().StringVarP(&channel, "channel", "c", "", "Channel name or ID to send to")
	rootCmd.Flags().StringVarP(&startDate, "date", "d", "", "Start date (YYYY-MM-DD)")
	rootCmd.Flags().StringVarP(&sendTime, "time", "t", "", "Time to send (HH:MM, 24-hour format, local time)")

	rootCmd.MarkFlagRequired("message")
	rootCmd.MarkFlagRequired("channel")
	// Note: --date is optional; defaults to today when --days or --end-date is specified
	rootCmd.MarkFlagRequired("time")

	// Optional flags
	rootCmd.Flags().StringVarP(&interval, "interval", "i", "none", "Repeat interval: none, daily, weekly, monthly")
	rootCmd.Flags().IntVarP(&repeatCount, "count", "n", 0, "Max number of times to send (0 = unlimited, use --end-date to limit)")
	rootCmd.Flags().StringVarP(&endDate, "end-date", "e", "", "End date (YYYY-MM-DD). Recurrence stops on or before this date")
	rootCmd.Flags().StringVar(&days, "days", "", "Days of week for weekly schedule (comma-separated: mon,tue,wed,thu,fri,sat,sun)")
	rootCmd.Flags().StringVarP(&groupName, "group", "g", "", "Name for this group of messages (for easier management)")

	// Init command
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Create a credentials template file",
		Long:  "Creates a template credentials file in the current directory that you can edit with your Slack token.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return config.CreateTemplateCredentials()
		},
	}
	rootCmd.AddCommand(initCmd)

	// List command
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List scheduled messages",
		Long: `Lists all scheduled messages. Optionally filter by channel.

Note: This shows messages scheduled via the API. Messages scheduled through
Slack's native UI are managed separately and won't appear here.`,
		Example: `  # List all scheduled messages
  ./slack-scheduler list

  # List scheduled messages for a specific channel
  ./slack-scheduler list -c general

  # List message groups
  ./slack-scheduler list --groups`,
		RunE: runList,
	}
	listCmd.Flags().StringVarP(&listChannel, "channel", "c", "", "Filter by channel name or ID")
	var listGroups bool
	listCmd.Flags().BoolVar(&listGroups, "groups", false, "List message groups instead of individual messages")
	rootCmd.AddCommand(listCmd)

	// Delete command
	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete scheduled messages",
		Long: `Delete scheduled messages by ID, group, or all messages in a channel.

Use 'list' command to find message IDs and groups first.`,
		Example: `  # Delete a specific message by ID
  ./slack-scheduler delete -c general --id Q0A7Z0QMWAF

  # Delete all messages in a group
  ./slack-scheduler delete --group "standup-reminders"

  # Delete ALL scheduled messages in a channel
  ./slack-scheduler delete -c general --all

  # Delete ALL scheduled messages across all channels
  ./slack-scheduler delete --all`,
		RunE: runDelete,
	}
	deleteCmd.Flags().StringVarP(&deleteChannel, "channel", "c", "", "Channel name or ID")
	deleteCmd.Flags().StringVar(&deleteID, "id", "", "Specific scheduled message ID to delete")
	deleteCmd.Flags().StringVar(&deleteGroup, "group", "", "Delete all messages in a group")
	deleteCmd.Flags().BoolVar(&deleteAll, "all", false, "Delete ALL scheduled messages in the channel")
	rootCmd.AddCommand(deleteCmd)

	// Modify command
	modifyCmd := &cobra.Command{
		Use:   "modify",
		Short: "Modify scheduled messages",
		Long: `Modify scheduled messages by deleting and recreating them with new parameters.

Since the Slack API doesn't support updating scheduled messages directly,
this command deletes the existing messages and creates new ones with the
updated parameters.

You can modify by message ID (for a single message) or by group name
(for all messages in a group).`,
		Example: `  # Change the channel for all messages in a group
  ./slack-scheduler modify --group "standup" --channel new-channel

  # Change the message text for a group
  ./slack-scheduler modify --group "standup" --message "New message text"

  # Change the time for all messages in a group
  ./slack-scheduler modify --group "standup" --time 10:00

  # Modify a single scheduled message
  ./slack-scheduler modify --id Q0A7Z0QMWAF --channel general --message "Updated text"`,
		RunE: runModify,
	}
	modifyCmd.Flags().StringVar(&modifyGroup, "group", "", "Group name to modify")
	modifyCmd.Flags().StringVar(&modifyID, "id", "", "Specific message ID to modify (requires --channel)")
	modifyCmd.Flags().StringVarP(&modifyChannel, "channel", "c", "", "New channel for the messages")
	modifyCmd.Flags().StringVarP(&modifyMessage, "message", "m", "", "New message text")
	modifyCmd.Flags().StringVarP(&modifyTime, "time", "t", "", "New time (HH:MM, 24-hour format)")
	modifyCmd.Flags().StringVarP(&modifyStartDate, "date", "d", "", "New start date (YYYY-MM-DD) - only for recreating")
	modifyCmd.Flags().StringVarP(&modifyInterval, "interval", "i", "", "New interval (none, daily, weekly, monthly)")
	modifyCmd.Flags().StringVar(&modifyDays, "days", "", "New days of week (comma-separated)")
	modifyCmd.Flags().IntVarP(&modifyCount, "count", "n", 0, "New repeat count")
	modifyCmd.Flags().StringVarP(&modifyEndDate, "end-date", "e", "", "New end date (YYYY-MM-DD)")
	modifyCmd.Flags().BoolVar(&modifyKeepFuture, "keep-future", true, "Only modify future scheduled times (default: true)")
	rootCmd.AddCommand(modifyCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runSchedule(cmd *cobra.Command, args []string) error {
	// Validate interval
	intervalType := types.Interval(interval)
	if !intervalType.IsValid() {
		return fmt.Errorf("invalid interval: %s (use: none, daily, weekly, monthly)", interval)
	}

	// Parse days of week
	parsedDays, err := types.ParseDaysOfWeek(days)
	if err != nil {
		return err
	}

	// If days specified but interval is not weekly, warn user
	if len(parsedDays) > 0 && intervalType != types.IntervalWeekly {
		fmt.Println("Warning: --days flag is only used with weekly interval")
	}

	// If start date not provided, default to today
	if startDate == "" {
		// Only allow this if we have days or end-date specified (for recurring schedules)
		if len(parsedDays) > 0 || endDate != "" || intervalType != types.IntervalNone {
			startDate = time.Now().In(scheduler.LocalTZ).Format("2006-01-02")
			fmt.Printf("Note: Using today (%s) as start date\n", startDate)
		} else {
			return fmt.Errorf("--date is required for one-time messages (or specify --days/--end-date for recurring)")
		}
	}

	// Validate date format
	if _, err := time.Parse("2006-01-02", startDate); err != nil {
		return fmt.Errorf("invalid date format: %s (use YYYY-MM-DD)", startDate)
	}

	// Validate time format
	if _, err := time.Parse("15:04", sendTime); err != nil {
		return fmt.Errorf("invalid time format: %s (use HH:MM, 24-hour)", sendTime)
	}

	// Validate end date if provided
	if endDate != "" {
		if _, err := time.Parse("2006-01-02", endDate); err != nil {
			return fmt.Errorf("invalid end date format: %s (use YYYY-MM-DD)", endDate)
		}
	}

	// Build config
	scheduleConfig := &types.ScheduleConfig{
		Message:     message,
		Channel:     channel,
		StartDate:   startDate,
		SendTime:    sendTime,
		Interval:    intervalType,
		RepeatCount: repeatCount,
		EndDate:     endDate,
		Days:        parsedDays,
	}

	// Load credentials
	creds, err := config.LoadCredentials()
	if err != nil {
		return err
	}

	// Create Slack client and validate
	client := slack.NewClient(creds.Token)
	if err := client.ValidateCredentials(); err != nil {
		return err
	}
	fmt.Println("✓ Credentials validated")

	// Resolve channel ID early for group tracking
	channelID, err := client.GetChannelID(channel)
	if err != nil {
		return err
	}

	// Create scheduler and run
	sched := scheduler.New(client, scheduleConfig)

	// Preview what will be scheduled
	times, err := sched.CalculateScheduleTimes()
	if err != nil {
		return err
	}

	// Display message with any mention conversions noted
	displayMessage := message
	convertedMessage := slack.ConvertMentions(message)
	if displayMessage != convertedMessage {
		fmt.Printf("\nNote: @channel/@here/@everyone mentions will be converted to Slack API format\n")
		fmt.Printf("      for proper notification delivery.\n")
	}

	fmt.Printf("\nScheduling %d message(s) to #%s:\n", len(times), channel)
	fmt.Printf("Message: %s\n\n", message)

	for i, t := range times {
		fmt.Printf("  %d. %s\n", i+1, t.Format("Mon Jan 02, 2006 at 03:04 PM MST"))
	}
	fmt.Println()

	// Schedule the messages
	scheduledIDs, err := sched.Schedule()
	if err != nil {
		return fmt.Errorf("scheduling failed: %w", err)
	}

	// Save group information if we scheduled multiple messages or a group name was provided
	if len(scheduledIDs) > 0 {
		group := MessageGroup{
			ID:         generateGroupID(),
			Name:       groupName,
			Channel:    channel,
			ChannelID:  channelID,
			Message:    message,
			Interval:   interval,
			SendTime:   sendTime,
			MessageIDs: scheduledIDs,
			CreatedAt:  time.Now(),
		}

		// Convert days to strings for storage
		for _, d := range parsedDays {
			group.Days = append(group.Days, string(d))
		}

		// Generate a default name if not provided
		if group.Name == "" {
			group.Name = fmt.Sprintf("%s-%s", channel, group.ID[:8])
		}

		if err := saveGroup(group); err != nil {
			fmt.Printf("Warning: Could not save group info: %v\n", err)
		} else {
			fmt.Printf("Group saved as: %s\n", group.Name)
		}
	}

	fmt.Printf("\n✓ Successfully scheduled %d message(s)\n", len(scheduledIDs))
	return nil
}

func runList(cmd *cobra.Command, args []string) error {
	showGroups, _ := cmd.Flags().GetBool("groups")

	if showGroups {
		return listGroups()
	}

	// Load credentials
	creds, err := config.LoadCredentials()
	if err != nil {
		return err
	}

	client := slack.NewClient(creds.Token)

	// Resolve channel ID if provided
	var channelID string
	if listChannel != "" {
		channelID, err = client.GetChannelID(listChannel)
		if err != nil {
			return err
		}
	}

	// Get scheduled messages
	messages, err := client.ListScheduledMessages(channelID)
	if err != nil {
		return err
	}

	if len(messages) == 0 {
		fmt.Println("No scheduled messages found.")
		return nil
	}

	// Get channel name map for display
	channelNames, _ := client.GetChannelNameMap()

	// Load groups for matching
	groups, _ := loadGroups()
	messageToGroup := make(map[string]string)
	for _, g := range groups.Groups {
		for _, msgID := range g.MessageIDs {
			messageToGroup[msgID] = g.Name
		}
	}

	fmt.Printf("Found %d scheduled message(s):\n\n", len(messages))

	for _, msg := range messages {
		postAt := time.Unix(int64(msg.PostAt), 0)
		channelName := channelNames[msg.Channel]
		if channelName == "" {
			channelName = msg.Channel
		}

		// Convert message back to human-readable format for display
		displayText := slack.ConvertMentionsBack(msg.Text)
		if len(displayText) > 60 {
			displayText = displayText[:57] + "..."
		}

		groupInfo := ""
		if gName, ok := messageToGroup[msg.ID]; ok {
			groupInfo = fmt.Sprintf(" [group: %s]", gName)
		}

		fmt.Printf("ID: %s%s\n", msg.ID, groupInfo)
		fmt.Printf("  Channel: #%s (%s)\n", channelName, msg.Channel)
		fmt.Printf("  Time: %s\n", postAt.Format("Mon Jan 02, 2006 at 03:04 PM MST"))
		fmt.Printf("  Message: %s\n\n", displayText)
	}

	return nil
}

func listGroups() error {
	groups, err := loadGroups()
	if err != nil {
		return fmt.Errorf("failed to load groups: %w", err)
	}

	if len(groups.Groups) == 0 {
		fmt.Println("No message groups found.")
		return nil
	}

	fmt.Printf("Found %d message group(s):\n\n", len(groups.Groups))

	for _, g := range groups.Groups {
		fmt.Printf("Group: %s\n", g.Name)
		fmt.Printf("  ID: %s\n", g.ID)
		fmt.Printf("  Channel: #%s\n", g.Channel)
		fmt.Printf("  Time: %s\n", g.SendTime)
		fmt.Printf("  Interval: %s\n", g.Interval)
		if len(g.Days) > 0 {
			fmt.Printf("  Days: %s\n", strings.Join(g.Days, ", "))
		}
		fmt.Printf("  Messages: %d scheduled\n", len(g.MessageIDs))
		fmt.Printf("  Created: %s\n", g.CreatedAt.Format("2006-01-02 15:04"))

		// Show message preview
		displayMsg := g.Message
		if len(displayMsg) > 50 {
			displayMsg = displayMsg[:47] + "..."
		}
		fmt.Printf("  Message: %s\n\n", displayMsg)
	}

	return nil
}

func runDelete(cmd *cobra.Command, args []string) error {
	// Validate flags
	if deleteGroup == "" && deleteID == "" && !deleteAll {
		return fmt.Errorf("must specify --id, --group, or --all")
	}

	if deleteID != "" && deleteChannel == "" {
		return fmt.Errorf("--channel is required when deleting by --id")
	}

	// deleteAll can work without channel - it will delete ALL scheduled messages across all channels

	// Load credentials
	creds, err := config.LoadCredentials()
	if err != nil {
		return err
	}

	client := slack.NewClient(creds.Token)

	// Handle group deletion
	if deleteGroup != "" {
		return deleteByGroup(client, deleteGroup)
	}

	// Resolve channel ID only if channel is specified
	var channelID string
	if deleteChannel != "" {
		channelID, err = client.GetChannelID(deleteChannel)
		if err != nil {
			return err
		}
	}

	// Delete specific ID
	if deleteID != "" {
		fmt.Printf("Deleting scheduled message %s from channel %s...\n", deleteID, deleteChannel)
		if err := client.DeleteScheduledMessage(channelID, deleteID); err != nil {
			return err
		}
		fmt.Println("✓ Message deleted successfully")

		// Remove from groups
		removeMessageFromGroups(deleteID)
		return nil
	}

	// Delete all
	if deleteAll {
		// If channel specified, delete only from that channel
		// Otherwise delete ALL scheduled messages across all channels
		if deleteChannel != "" {
			messages, err := client.ListScheduledMessages(channelID)
			if err != nil {
				return err
			}

			if len(messages) == 0 {
				fmt.Println("No scheduled messages found in this channel.")
				return nil
			}

			fmt.Printf("Deleting %d scheduled message(s) from #%s...\n", len(messages), deleteChannel)

			deleted := 0
			for _, msg := range messages {
				if err := client.DeleteScheduledMessage(channelID, msg.ID); err != nil {
					fmt.Printf("  ✗ Failed to delete %s: %v\n", msg.ID, err)
				} else {
					fmt.Printf("  ✓ Deleted %s\n", msg.ID)
					deleted++
					removeMessageFromGroups(msg.ID)
				}
			}

			fmt.Printf("\n✓ Deleted %d of %d messages\n", deleted, len(messages))
		} else {
			// Delete ALL scheduled messages across all channels
			messages, err := client.ListScheduledMessages("")
			if err != nil {
				return err
			}

			if len(messages) == 0 {
				fmt.Println("No scheduled messages found.")
				return nil
			}

			// Get channel names for display
			channelNames, _ := client.GetChannelNameMap()

			fmt.Printf("Deleting %d scheduled message(s) across all channels...\n", len(messages))

			deleted := 0
			for _, msg := range messages {
				channelName := channelNames[msg.Channel]
				if channelName == "" {
					channelName = msg.Channel
				}
				if err := client.DeleteScheduledMessage(msg.Channel, msg.ID); err != nil {
					fmt.Printf("  ✗ Failed to delete %s from #%s: %v\n", msg.ID, channelName, err)
				} else {
					fmt.Printf("  ✓ Deleted %s from #%s\n", msg.ID, channelName)
					deleted++
					removeMessageFromGroups(msg.ID)
				}
			}

			// Also clear all groups
			if err := saveGroups(&GroupsFile{Groups: []MessageGroup{}}); err != nil {
				fmt.Printf("Warning: Could not clear groups file: %v\n", err)
			}

			fmt.Printf("\n✓ Deleted %d of %d messages\n", deleted, len(messages))
		}
	}

	return nil
}

func deleteByGroup(client *slack.Client, groupName string) error {
	groups, err := loadGroups()
	if err != nil {
		return fmt.Errorf("failed to load groups: %w", err)
	}

	var targetGroup *MessageGroup
	var targetIndex int
	for i, g := range groups.Groups {
		if g.Name == groupName || g.ID == groupName {
			targetGroup = &groups.Groups[i]
			targetIndex = i
			break
		}
	}

	if targetGroup == nil {
		return fmt.Errorf("group not found: %s", groupName)
	}

	fmt.Printf("Deleting %d message(s) from group '%s'...\n", len(targetGroup.MessageIDs), targetGroup.Name)

	deleted := 0
	for _, msgID := range targetGroup.MessageIDs {
		if err := client.DeleteScheduledMessage(targetGroup.ChannelID, msgID); err != nil {
			// Message might already be sent or deleted
			fmt.Printf("  ✗ Could not delete %s: %v\n", msgID, err)
		} else {
			fmt.Printf("  ✓ Deleted %s\n", msgID)
			deleted++
		}
	}

	// Remove the group from storage
	groups.Groups = append(groups.Groups[:targetIndex], groups.Groups[targetIndex+1:]...)
	if err := saveGroups(groups); err != nil {
		fmt.Printf("Warning: Could not update groups file: %v\n", err)
	}

	fmt.Printf("\n✓ Deleted %d of %d messages, group removed\n", deleted, len(targetGroup.MessageIDs))
	return nil
}

func runModify(cmd *cobra.Command, args []string) error {
	// Validate flags
	if modifyGroup == "" && modifyID == "" {
		return fmt.Errorf("must specify --group or --id")
	}

	if modifyID != "" && modifyChannel == "" {
		// Try to find from groups or require channel
		return fmt.Errorf("--channel is required when modifying by --id (need to know current channel)")
	}

	// Check that at least one modification is specified
	hasModification := modifyChannel != "" || modifyMessage != "" || modifyTime != "" ||
		modifyStartDate != "" || modifyInterval != "" || modifyDays != "" ||
		modifyCount > 0 || modifyEndDate != ""

	if !hasModification {
		return fmt.Errorf("must specify at least one attribute to modify (--channel, --message, --time, etc.)")
	}

	// Load credentials
	creds, err := config.LoadCredentials()
	if err != nil {
		return err
	}

	client := slack.NewClient(creds.Token)

	if modifyGroup != "" {
		return modifyByGroup(client, modifyGroup)
	}

	return modifyByID(client, modifyID, modifyChannel)
}

func modifyByGroup(client *slack.Client, groupName string) error {
	groups, err := loadGroups()
	if err != nil {
		return fmt.Errorf("failed to load groups: %w", err)
	}

	var targetGroup *MessageGroup
	var targetIndex int
	for i, g := range groups.Groups {
		if g.Name == groupName || g.ID == groupName {
			targetGroup = &groups.Groups[i]
			targetIndex = i
			break
		}
	}

	if targetGroup == nil {
		return fmt.Errorf("group not found: %s", groupName)
	}

	fmt.Printf("Modifying group '%s' (%d messages)...\n\n", targetGroup.Name, len(targetGroup.MessageIDs))

	// Get current scheduled messages to find their times
	currentMessages, err := client.ListScheduledMessages(targetGroup.ChannelID)
	if err != nil {
		return fmt.Errorf("failed to list current messages: %w", err)
	}

	// Build a map of message ID to scheduled time
	messageIDSet := make(map[string]bool)
	for _, id := range targetGroup.MessageIDs {
		messageIDSet[id] = true
	}

	var scheduledTimes []time.Time
	var messageIDsToDelete []string
	now := time.Now()

	for _, msg := range currentMessages {
		if messageIDSet[msg.ID] {
			postAt := time.Unix(int64(msg.PostAt), 0)

			// Only include future messages if --keep-future is true (default)
			if !modifyKeepFuture || postAt.After(now) {
				scheduledTimes = append(scheduledTimes, postAt)
				messageIDsToDelete = append(messageIDsToDelete, msg.ID)
			}
		}
	}

	if len(messageIDsToDelete) == 0 {
		fmt.Println("No modifiable messages found (all may have been sent already).")
		return nil
	}

	// Sort times for consistent ordering
	sort.Slice(scheduledTimes, func(i, j int) bool {
		return scheduledTimes[i].Before(scheduledTimes[j])
	})

	// Build new configuration
	newChannel := targetGroup.Channel
	newChannelID := targetGroup.ChannelID
	if modifyChannel != "" {
		newChannel = modifyChannel
		newChannelID, err = client.GetChannelID(modifyChannel)
		if err != nil {
			return fmt.Errorf("failed to resolve new channel: %w", err)
		}
	}

	newMessage := targetGroup.Message
	if modifyMessage != "" {
		newMessage = modifyMessage
	}

	newTime := targetGroup.SendTime
	if modifyTime != "" {
		// Validate time format
		if _, err := time.Parse("15:04", modifyTime); err != nil {
			return fmt.Errorf("invalid time format: %s (use HH:MM)", modifyTime)
		}
		newTime = modifyTime
	}

	newInterval := targetGroup.Interval
	if modifyInterval != "" {
		intervalType := types.Interval(modifyInterval)
		if !intervalType.IsValid() {
			return fmt.Errorf("invalid interval: %s", modifyInterval)
		}
		newInterval = modifyInterval
	}

	newDays := targetGroup.Days
	if modifyDays != "" {
		parsedDays, err := types.ParseDaysOfWeek(modifyDays)
		if err != nil {
			return err
		}
		newDays = make([]string, len(parsedDays))
		for i, d := range parsedDays {
			newDays[i] = string(d)
		}
	}

	// Preview changes
	fmt.Println("Changes to apply:")
	if modifyChannel != "" {
		fmt.Printf("  Channel: #%s → #%s\n", targetGroup.Channel, newChannel)
	}
	if modifyMessage != "" {
		oldMsg := targetGroup.Message
		if len(oldMsg) > 30 {
			oldMsg = oldMsg[:27] + "..."
		}
		newMsg := newMessage
		if len(newMsg) > 30 {
			newMsg = newMsg[:27] + "..."
		}
		fmt.Printf("  Message: \"%s\" → \"%s\"\n", oldMsg, newMsg)
	}
	if modifyTime != "" {
		fmt.Printf("  Time: %s → %s\n", targetGroup.SendTime, newTime)
	}
	if modifyInterval != "" {
		fmt.Printf("  Interval: %s → %s\n", targetGroup.Interval, newInterval)
	}
	if modifyDays != "" {
		fmt.Printf("  Days: %s → %s\n", strings.Join(targetGroup.Days, ","), strings.Join(newDays, ","))
	}
	fmt.Printf("\nAffected messages: %d\n\n", len(messageIDsToDelete))

	// Step 1: Delete old messages
	fmt.Println("Step 1: Deleting old scheduled messages...")
	deleted := 0
	for _, msgID := range messageIDsToDelete {
		if err := client.DeleteScheduledMessage(targetGroup.ChannelID, msgID); err != nil {
			fmt.Printf("  ✗ Could not delete %s: %v\n", msgID, err)
		} else {
			fmt.Printf("  ✓ Deleted %s\n", msgID)
			deleted++
		}
	}

	if deleted == 0 {
		return fmt.Errorf("could not delete any messages, aborting modification")
	}

	// Step 2: Create new messages with modified parameters
	fmt.Println("\nStep 2: Creating new scheduled messages...")

	var newMessageIDs []string

	// If time was modified, we need to adjust the scheduled times
	if modifyTime != "" {
		// Parse the new time
		newTimeHour, newTimeMin := parseTime(newTime)

		for i := range scheduledTimes {
			// Keep the date but change the time
			oldTime := scheduledTimes[i]
			scheduledTimes[i] = time.Date(
				oldTime.Year(), oldTime.Month(), oldTime.Day(),
				newTimeHour, newTimeMin, 0, 0, oldTime.Location(),
			)
		}
	}

	for _, schedTime := range scheduledTimes {
		// Skip if the time is now in the past after adjustment
		if schedTime.Before(time.Now()) {
			fmt.Printf("  ⚠ Skipping past time: %s\n", schedTime.Format("2006-01-02 15:04"))
			continue
		}

		msgID, err := client.ScheduleMessage(newChannelID, newMessage, schedTime)
		if err != nil {
			fmt.Printf("  ✗ Failed to schedule for %s: %v\n", schedTime.Format("2006-01-02 15:04"), err)
		} else {
			newMessageIDs = append(newMessageIDs, msgID)
		}
	}

	if len(newMessageIDs) == 0 {
		return fmt.Errorf("could not create any new scheduled messages")
	}

	// Step 3: Update group info
	targetGroup.Channel = newChannel
	targetGroup.ChannelID = newChannelID
	targetGroup.Message = newMessage
	targetGroup.SendTime = newTime
	targetGroup.Interval = newInterval
	targetGroup.Days = newDays
	targetGroup.MessageIDs = newMessageIDs

	groups.Groups[targetIndex] = *targetGroup
	if err := saveGroups(groups); err != nil {
		fmt.Printf("Warning: Could not save group updates: %v\n", err)
	}

	fmt.Printf("\n✓ Successfully modified group '%s'\n", targetGroup.Name)
	fmt.Printf("  Deleted: %d messages\n", deleted)
	fmt.Printf("  Created: %d messages\n", len(newMessageIDs))

	return nil
}

func modifyByID(client *slack.Client, msgID string, channelName string) error {
	channelID, err := client.GetChannelID(channelName)
	if err != nil {
		return fmt.Errorf("failed to resolve channel: %w", err)
	}

	// Get the current message info
	messages, err := client.ListScheduledMessages(channelID)
	if err != nil {
		return fmt.Errorf("failed to list messages: %w", err)
	}

	var targetMsg *struct {
		ID     string
		PostAt int
		Text   string
	}

	for _, msg := range messages {
		if msg.ID == msgID {
			targetMsg = &struct {
				ID     string
				PostAt int
				Text   string
			}{
				ID:     msg.ID,
				PostAt: msg.PostAt,
				Text:   msg.Text,
			}
			break
		}
	}

	if targetMsg == nil {
		return fmt.Errorf("scheduled message not found: %s", msgID)
	}

	// Determine new values
	newChannelID := channelID
	newChannelName := channelName
	if modifyChannel != "" {
		newChannelID, err = client.GetChannelID(modifyChannel)
		if err != nil {
			return fmt.Errorf("failed to resolve new channel: %w", err)
		}
		newChannelName = modifyChannel
	}

	newMessage := slack.ConvertMentionsBack(targetMsg.Text)
	if modifyMessage != "" {
		newMessage = modifyMessage
	}

	schedTime := time.Unix(int64(targetMsg.PostAt), 0)
	if modifyTime != "" {
		hour, min := parseTime(modifyTime)
		schedTime = time.Date(
			schedTime.Year(), schedTime.Month(), schedTime.Day(),
			hour, min, 0, 0, schedTime.Location(),
		)
	}

	fmt.Printf("Modifying message %s...\n", msgID)
	fmt.Printf("  Channel: %s → %s\n", channelName, newChannelName)
	fmt.Printf("  Time: %s\n", schedTime.Format("2006-01-02 15:04 MST"))

	// Delete old message
	fmt.Println("\nDeleting old message...")
	if err := client.DeleteScheduledMessage(channelID, msgID); err != nil {
		return fmt.Errorf("failed to delete old message: %w", err)
	}
	fmt.Println("✓ Deleted")

	// Create new message
	fmt.Println("Creating new message...")
	newID, err := client.ScheduleMessage(newChannelID, newMessage, schedTime)
	if err != nil {
		return fmt.Errorf("failed to create new message: %w", err)
	}

	// Update groups if this message was in one
	updateMessageIDInGroups(msgID, newID)

	fmt.Printf("\n✓ Message modified successfully\n")
	fmt.Printf("  New ID: %s\n", newID)

	return nil
}

// Helper functions

func parseTime(timeStr string) (int, int) {
	parts := strings.Split(timeStr, ":")
	hour, _ := strconv.Atoi(parts[0])
	min, _ := strconv.Atoi(parts[1])
	return hour, min
}

func generateGroupID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func getGroupsFilePath() string {
	cwd, err := os.Getwd()
	if err != nil {
		return groupsFileName
	}
	return filepath.Join(cwd, groupsFileName)
}

func loadGroups() (*GroupsFile, error) {
	path := getGroupsFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &GroupsFile{Groups: []MessageGroup{}}, nil
		}
		return nil, err
	}

	var groups GroupsFile
	if err := json.Unmarshal(data, &groups); err != nil {
		return nil, err
	}

	return &groups, nil
}

func saveGroups(groups *GroupsFile) error {
	path := getGroupsFilePath()
	data, err := json.MarshalIndent(groups, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func saveGroup(group MessageGroup) error {
	groups, err := loadGroups()
	if err != nil {
		groups = &GroupsFile{Groups: []MessageGroup{}}
	}

	groups.Groups = append(groups.Groups, group)
	return saveGroups(groups)
}

func removeMessageFromGroups(msgID string) {
	groups, err := loadGroups()
	if err != nil {
		return
	}

	modified := false
	for i, g := range groups.Groups {
		for j, id := range g.MessageIDs {
			if id == msgID {
				groups.Groups[i].MessageIDs = append(g.MessageIDs[:j], g.MessageIDs[j+1:]...)
				modified = true
				break
			}
		}
	}

	if modified {
		saveGroups(groups)
	}
}

func updateMessageIDInGroups(oldID, newID string) {
	groups, err := loadGroups()
	if err != nil {
		return
	}

	modified := false
	for i, g := range groups.Groups {
		for j, id := range g.MessageIDs {
			if id == oldID {
				groups.Groups[i].MessageIDs[j] = newID
				modified = true
				break
			}
		}
	}

	if modified {
		saveGroups(groups)
	}
}
