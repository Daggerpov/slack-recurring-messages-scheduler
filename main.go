package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/slack-go/slack"
	"github.com/spf13/cobra"
)

var (
	// CLI flags for schedule command
	message     string
	channel     string
	startDate   string
	sendTime    string
	interval    string
	repeatCount int
	endDate     string
	days        string

	// CLI flags for list command
	listChannel string

	// CLI flags for delete command
	deleteAll bool
)

// IndexedMessage wraps a scheduled message with a simple integer ID
type IndexedMessage struct {
	Index      int
	SlackID    string
	ChannelID  string
	ChannelName string
	Text       string
	PostAt     time.Time
	GroupLabel string
}

// MessageGroup represents a group of messages with the same text
type MessageGroup struct {
	Label    string
	Text     string
	Messages []*IndexedMessage
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "slack-scheduler",
		Short: "Schedule Slack messages to be sent at specific times",
		Long: `A CLI tool to schedule Slack messages with support for:
- One-time scheduled messages
- Recurring messages (daily, weekly, monthly)
- Specific days of the week for weekly schedules
- Full Slack formatting support (@mentions, emoji, etc.)

Messages are scheduled using your system's local timezone.`,
		Example: `  # Send a one-time message
  slack-scheduler -m "Hello team!" -c general -d 2025-01-17 -t 14:00

  # Send every Friday at 2pm for 4 weeks
  slack-scheduler -m "Weekly reminder!" -c general -d 2025-01-17 -t 14:00 -i weekly -n 4

  # Send on Monday and Friday at 9am for 8 occurrences
  slack-scheduler -m "Standup time!" -c engineering -d 2025-01-13 -t 09:00 -i weekly -n 8 --days mon,fri`,
		RunE: runSchedule,
	}

	// Required flags
	rootCmd.Flags().StringVarP(&message, "message", "m", "", "Message to send (supports @mentions, emoji, Slack formatting)")
	rootCmd.Flags().StringVarP(&channel, "channel", "c", "", "Channel name or ID to send to")
	rootCmd.Flags().StringVarP(&startDate, "date", "d", "", "Start date (YYYY-MM-DD)")
	rootCmd.Flags().StringVarP(&sendTime, "time", "t", "", "Time to send (HH:MM, 24-hour format, local time)")

	rootCmd.MarkFlagRequired("message")
	rootCmd.MarkFlagRequired("channel")
	rootCmd.MarkFlagRequired("date")
	rootCmd.MarkFlagRequired("time")

	// Optional flags
	rootCmd.Flags().StringVarP(&interval, "interval", "i", "none", "Repeat interval: none, daily, weekly, monthly")
	rootCmd.Flags().IntVarP(&repeatCount, "count", "n", 1, "Number of times to send (for repeating schedules)")
	rootCmd.Flags().StringVarP(&endDate, "end-date", "e", "", "End date (YYYY-MM-DD). Recurrence stops on or before this date")
	rootCmd.Flags().StringVar(&days, "days", "", "Days of week for weekly schedule (comma-separated: mon,tue,wed,thu,fri,sat,sun)")

	// Init command to create credentials template
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Create a credentials template file",
		Long:  "Creates a template credentials file in your home directory that you can edit with your Slack token.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return CreateTemplateCredentials()
		},
	}
	rootCmd.AddCommand(initCmd)

	// List command to show scheduled messages
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all scheduled messages",
		Long: `List all messages scheduled via the Slack API.

Note: Messages scheduled via the API don't appear in Slack's UI "Scheduled Messages" view.
Use this command to see and manage API-scheduled messages.`,
		Example: `  # List all scheduled messages
  slack-scheduler list

  # List scheduled messages for a specific channel
  slack-scheduler list -c general`,
		RunE: runList,
	}
	listCmd.Flags().StringVarP(&listChannel, "channel", "c", "", "Filter by channel name or ID (optional)")
	rootCmd.AddCommand(listCmd)

	// Delete command to cancel scheduled messages
	deleteCmd := &cobra.Command{
		Use:   "delete [IDs or Groups...]",
		Short: "Delete scheduled messages",
		Long: `Delete (cancel) scheduled messages by ID or group.

Use 'slack-scheduler list' to find message IDs and groups.
You can delete multiple messages at once using IDs (integers) or group labels (letters).`,
		Example: `  # Delete specific messages by ID
  slack-scheduler delete 1 2 4

  # Delete all messages in a group
  slack-scheduler delete A

  # Mix IDs and groups
  slack-scheduler delete A 4 5

  # Delete all scheduled messages
  slack-scheduler delete --all`,
		RunE: runDelete,
	}
	deleteCmd.Flags().BoolVar(&deleteAll, "all", false, "Delete all scheduled messages")
	rootCmd.AddCommand(deleteCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runSchedule(cmd *cobra.Command, args []string) error {
	// Validate interval
	intervalType := Interval(interval)
	if !intervalType.IsValid() {
		return fmt.Errorf("invalid interval: %s (use: none, daily, weekly, monthly)", interval)
	}

	// Check if user specified -n or -e without -i
	if intervalType == IntervalNone && (repeatCount > 1 || endDate != "") {
		return fmt.Errorf("to create a recurring schedule, you must specify -i (interval)\n" +
			"Example: slack-scheduler -m \"Hello\" -c general -d 2025-01-17 -t 14:00 -i daily -n 5\n" +
			"Use -i with: daily, weekly, or monthly")
	}

	// Parse days of week
	parsedDays, err := ParseDaysOfWeek(days)
	if err != nil {
		return err
	}

	// If days specified but interval is not weekly, warn user
	if len(parsedDays) > 0 && intervalType != IntervalWeekly {
		fmt.Println("Warning: --days flag is only used with weekly interval")
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
		// Check that end date is after start date
		start, _ := time.Parse("2006-01-02", startDate)
		end, _ := time.Parse("2006-01-02", endDate)
		if end.Before(start) {
			return fmt.Errorf("end date (%s) must be after start date (%s)", endDate, startDate)
		}
	}

	// Build config
	config := &ScheduleConfig{
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
	creds, err := LoadCredentials()
	if err != nil {
		return err
	}

	// Create Slack client and validate
	client := NewSlackClient(creds.Token)
	if err := client.ValidateCredentials(); err != nil {
		return err
	}
	fmt.Println("✓ Credentials validated")

	// Create scheduler and run
	scheduler := NewScheduler(client, config)

	// Preview what will be scheduled
	times, err := scheduler.CalculateScheduleTimes()
	if err != nil {
		return err
	}

	fmt.Printf("\nScheduling %d message(s) to #%s:\n", len(times), channel)
	fmt.Printf("Message: %s\n\n", message)

	for i, t := range times {
		fmt.Printf("  %d. %s\n", i+1, t.Format("Mon Jan 02, 2006 at 03:04 PM MST"))
	}
	fmt.Println()

	// Schedule the messages
	scheduledIDs, err := scheduler.Schedule()
	if err != nil {
		return fmt.Errorf("scheduling failed: %w", err)
	}

	fmt.Printf("\n✓ Successfully scheduled %d message(s)\n", len(scheduledIDs))
	return nil
}

// generateGroupLabel generates a group label like A, B, ..., Z, A2, B2, ...
func generateGroupLabel(index int) string {
	letter := 'A' + rune(index%26)
	cycle := index / 26
	if cycle == 0 {
		return string(letter)
	}
	return fmt.Sprintf("%c%d", letter, cycle+1)
}

// parseGroupLabel parses a group label like "A", "B2" into an index
func parseGroupLabel(label string) (int, bool) {
	label = strings.ToUpper(strings.TrimSpace(label))
	if len(label) == 0 {
		return 0, false
	}

	letter := label[0]
	if letter < 'A' || letter > 'Z' {
		return 0, false
	}

	letterIndex := int(letter - 'A')
	
	if len(label) == 1 {
		return letterIndex, true
	}

	// Parse the cycle number (e.g., "2" from "A2")
	cycleStr := label[1:]
	cycle, err := strconv.Atoi(cycleStr)
	if err != nil || cycle < 2 {
		return 0, false
	}

	return letterIndex + (cycle-1)*26, true
}

// buildIndexedMessages creates IndexedMessage list from Slack messages
func buildIndexedMessages(messages []slack.ScheduledMessage, channelNameMap map[string]string) []*IndexedMessage {
	// Sort messages by post time for consistent ordering
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].PostAt < messages[j].PostAt
	})

	indexed := make([]*IndexedMessage, len(messages))
	for i, msg := range messages {
		channelName := channelNameMap[msg.Channel]
		if channelName == "" {
			channelName = msg.Channel
		}
		indexed[i] = &IndexedMessage{
			Index:       i + 1, // 1-based indexing
			SlackID:     msg.ID,
			ChannelID:   msg.Channel,
			ChannelName: channelName,
			Text:        msg.Text,
			PostAt:      time.Unix(int64(msg.PostAt), 0).In(localTZ),
		}
	}
	return indexed
}

// groupMessages groups messages by their text content
func groupMessages(messages []*IndexedMessage) []*MessageGroup {
	// Group by text
	textGroups := make(map[string][]*IndexedMessage)
	textOrder := []string{} // Maintain order of first occurrence
	
	for _, msg := range messages {
		if _, exists := textGroups[msg.Text]; !exists {
			textOrder = append(textOrder, msg.Text)
		}
		textGroups[msg.Text] = append(textGroups[msg.Text], msg)
	}

	// Create groups with labels
	groups := make([]*MessageGroup, len(textOrder))
	for i, text := range textOrder {
		label := generateGroupLabel(i)
		msgs := textGroups[text]
		
		// Assign group label to each message
		for _, msg := range msgs {
			msg.GroupLabel = label
		}
		
		groups[i] = &MessageGroup{
			Label:    label,
			Text:     text,
			Messages: msgs,
		}
	}

	return groups
}

func runList(cmd *cobra.Command, args []string) error {
	// Load credentials
	creds, err := LoadCredentials()
	if err != nil {
		return err
	}

	// Create Slack client and validate
	client := NewSlackClient(creds.Token)
	if err := client.ValidateCredentials(); err != nil {
		return err
	}
	fmt.Println("✓ Credentials validated")

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
		fmt.Println("\nNo scheduled messages found.")
		return nil
	}

	// Get channel name map
	channelNameMap, err := client.GetChannelNameMap()
	if err != nil {
		// Non-fatal, we'll just use IDs
		channelNameMap = make(map[string]string)
	}

	// Build indexed messages and groups
	indexed := buildIndexedMessages(messages, channelNameMap)
	groups := groupMessages(indexed)

	fmt.Printf("\nFound %d scheduled message(s) in %d group(s):\n", len(messages), len(groups))

	for _, group := range groups {
		// Truncate message text for display
		displayText := group.Text
		if len(displayText) > 60 {
			displayText = displayText[:60] + "..."
		}
		
		fmt.Printf("\n━━━ Group %s ━━━\n", group.Label)
		fmt.Printf("    Message: %s\n", displayText)
		
		for _, msg := range group.Messages {
			fmt.Printf("\n    ID: %d\n", msg.Index)
			fmt.Printf("    Channel: #%s\n", msg.ChannelName)
			fmt.Printf("    Scheduled: %s\n", msg.PostAt.Format("Mon Jan 02, 2006 at 03:04 PM MST"))
		}
	}
	fmt.Println()

	return nil
}

// isGroupLabel checks if a string is a valid group label (letter or letter+number)
func isGroupLabel(s string) bool {
	s = strings.ToUpper(strings.TrimSpace(s))
	if len(s) == 0 {
		return false
	}
	
	// First character must be a letter
	if s[0] < 'A' || s[0] > 'Z' {
		return false
	}
	
	// Single letter is valid
	if len(s) == 1 {
		return true
	}
	
	// Rest must be digits >= 2
	for _, c := range s[1:] {
		if !unicode.IsDigit(c) {
			return false
		}
	}
	
	// Check the number is >= 2
	if num, err := strconv.Atoi(s[1:]); err != nil || num < 2 {
		return false
	}
	
	return true
}

func runDelete(cmd *cobra.Command, args []string) error {
	// Validate arguments
	if len(args) == 0 && !deleteAll {
		return fmt.Errorf("must specify message IDs, group labels, or --all\nUsage: slack-scheduler delete [IDs or Groups...] or slack-scheduler delete --all")
	}
	if len(args) > 0 && deleteAll {
		return fmt.Errorf("cannot specify both IDs/groups and --all")
	}

	// Load credentials
	creds, err := LoadCredentials()
	if err != nil {
		return err
	}

	// Create Slack client and validate
	client := NewSlackClient(creds.Token)
	if err := client.ValidateCredentials(); err != nil {
		return err
	}
	fmt.Println("✓ Credentials validated")

	// Get all scheduled messages
	messages, err := client.ListScheduledMessages("")
	if err != nil {
		return err
	}

	if len(messages) == 0 {
		fmt.Println("\nNo scheduled messages found.")
		return nil
	}

	// Get channel name map for display
	channelNameMap, err := client.GetChannelNameMap()
	if err != nil {
		channelNameMap = make(map[string]string)
	}

	// Build indexed messages and groups
	indexed := buildIndexedMessages(messages, channelNameMap)
	groups := groupMessages(indexed)

	// Create lookup maps
	indexToMsg := make(map[int]*IndexedMessage)
	for _, msg := range indexed {
		indexToMsg[msg.Index] = msg
	}

	groupLabelToGroup := make(map[string]*MessageGroup)
	for _, g := range groups {
		groupLabelToGroup[strings.ToUpper(g.Label)] = g
	}

	// Collect messages to delete
	toDelete := make(map[int]*IndexedMessage) // Use map to avoid duplicates

	if deleteAll {
		for _, msg := range indexed {
			toDelete[msg.Index] = msg
		}
	} else {
		for _, arg := range args {
			arg = strings.TrimSpace(arg)
			
			// Try to parse as integer ID first
			if id, err := strconv.Atoi(arg); err == nil {
				msg, exists := indexToMsg[id]
				if !exists {
					return fmt.Errorf("message ID %d not found (valid IDs: 1-%d)", id, len(indexed))
				}
				toDelete[msg.Index] = msg
				continue
			}
			
			// Try to parse as group label
			if isGroupLabel(arg) {
				group, exists := groupLabelToGroup[strings.ToUpper(arg)]
				if !exists {
					validLabels := make([]string, len(groups))
					for i, g := range groups {
						validLabels[i] = g.Label
					}
					return fmt.Errorf("group %s not found (valid groups: %s)", strings.ToUpper(arg), strings.Join(validLabels, ", "))
				}
				for _, msg := range group.Messages {
					toDelete[msg.Index] = msg
				}
				continue
			}
			
			return fmt.Errorf("invalid argument: %s (expected integer ID or group label like A, B, A2, etc.)", arg)
		}
	}

	if len(toDelete) == 0 {
		fmt.Println("\nNo messages to delete.")
		return nil
	}

	// Sort by ID for consistent output
	var sortedMsgs []*IndexedMessage
	for _, msg := range toDelete {
		sortedMsgs = append(sortedMsgs, msg)
	}
	sort.Slice(sortedMsgs, func(i, j int) bool {
		return sortedMsgs[i].Index < sortedMsgs[j].Index
	})

	fmt.Printf("\nDeleting %d scheduled message(s)...\n", len(sortedMsgs))
	deleted := 0
	for _, msg := range sortedMsgs {
		// Check for empty Slack ID
		if msg.SlackID == "" {
			fmt.Printf("  ✗ Failed to delete ID %d (#%s): no Slack message ID available\n", msg.Index, msg.ChannelName)
			continue
		}
		if err := client.DeleteScheduledMessage(msg.ChannelID, msg.SlackID); err != nil {
			fmt.Printf("  ✗ Failed to delete ID %d (#%s): %v\n", msg.Index, msg.ChannelName, err)
			// Check if the scheduled time has passed
			if msg.PostAt.Before(time.Now()) {
				fmt.Printf("      Note: This message was scheduled for %s which has already passed.\n", msg.PostAt.Format("03:04 PM"))
				fmt.Printf("      The message may have already been sent and is no longer a scheduled message.\n")
			}
		} else {
			displayText := msg.Text
			if len(displayText) > 40 {
				displayText = displayText[:40] + "..."
			}
			fmt.Printf("  ✓ Deleted ID %d (#%s): %s\n", msg.Index, msg.ChannelName, displayText)
			deleted++
		}
	}
	fmt.Printf("\n✓ Deleted %d message(s)\n", deleted)

	return nil
}
