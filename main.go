package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

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
)

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
