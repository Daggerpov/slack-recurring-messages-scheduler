package main

import (
	"fmt"
	"time"
)

// localTZ is the user's local timezone
var localTZ *time.Location

func init() {
	localTZ = time.Local
}

// Scheduler handles message scheduling logic
type Scheduler struct {
	client *SlackClient
	config *ScheduleConfig
}

// NewScheduler creates a new scheduler
func NewScheduler(client *SlackClient, config *ScheduleConfig) *Scheduler {
	return &Scheduler{
		client: client,
		config: config,
	}
}

// CalculateScheduleTimes returns all the times when messages should be sent
func (s *Scheduler) CalculateScheduleTimes() ([]time.Time, error) {
	// Parse start date and time
	startDateTime, err := s.parseDateTime(s.config.StartDate, s.config.SendTime)
	if err != nil {
		return nil, err
	}

	// Parse end date if provided (set to end of day)
	var endDateTime *time.Time
	if s.config.EndDate != "" {
		end, err := time.ParseInLocation("2006-01-02", s.config.EndDate, localTZ)
		if err != nil {
			return nil, fmt.Errorf("failed to parse end date: %w", err)
		}
		// Set to end of day (23:59:59)
		endOfDay := time.Date(end.Year(), end.Month(), end.Day(), 23, 59, 59, 0, localTZ)
		endDateTime = &endOfDay
	}

	var times []time.Time

	switch s.config.Interval {
	case IntervalNone:
		// Single message
		times = append(times, startDateTime)

	case IntervalDaily:
		times = s.calculateDailyTimes(startDateTime, endDateTime)

	case IntervalWeekly:
		times = s.calculateWeeklyTimes(startDateTime, endDateTime)

	case IntervalMonthly:
		times = s.calculateMonthlyTimes(startDateTime, endDateTime)

	default:
		return nil, fmt.Errorf("invalid interval: %s", s.config.Interval)
	}

	return times, nil
}

func (s *Scheduler) parseDateTime(date, timeStr string) (time.Time, error) {
	dateTimeStr := fmt.Sprintf("%s %s", date, timeStr)
	t, err := time.ParseInLocation("2006-01-02 15:04", dateTimeStr, localTZ)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse date/time: %w", err)
	}
	return t, nil
}

func (s *Scheduler) calculateDailyTimes(start time.Time, endDate *time.Time) []time.Time {
	var times []time.Time
	current := start
	count := s.config.RepeatCount

	// If no end date and count <= 0, default to 1
	if endDate == nil && count <= 0 {
		count = 1
	}

	for {
		// Check if we've exceeded end date
		if endDate != nil && current.After(*endDate) {
			break
		}

		times = append(times, current)

		// Check count limit (if count is set and positive)
		if count > 0 && len(times) >= count {
			break
		}

		// Move to next day
		current = current.AddDate(0, 0, 1)

		// Safety limit to prevent infinite loops (only if no end date)
		if endDate == nil && current.After(start.AddDate(10, 0, 0)) {
			break
		}
	}

	return times
}

func (s *Scheduler) calculateWeeklyTimes(start time.Time, endDate *time.Time) []time.Time {
	var times []time.Time

	// If specific days are specified, use them
	if len(s.config.Days) > 0 {
		times = s.calculateSpecificDaysTimes(start, endDate)
	} else {
		// Otherwise, repeat on the same day of week
		current := start
		count := s.config.RepeatCount

		// If no end date and count <= 0, default to 1
		if endDate == nil && count <= 0 {
			count = 1
		}

		for {
			// Check if we've exceeded end date
			if endDate != nil && current.After(*endDate) {
				break
			}

			times = append(times, current)

			// Check count limit (if count is set and positive)
			if count > 0 && len(times) >= count {
				break
			}

			// Move to next week
			current = current.AddDate(0, 0, 7)

			// Safety limit to prevent infinite loops (only if no end date)
			if endDate == nil && current.After(start.AddDate(5, 0, 0)) {
				break
			}
		}
	}

	return times
}

func (s *Scheduler) calculateSpecificDaysTimes(start time.Time, endDate *time.Time) []time.Time {
	var times []time.Time
	current := start
	count := s.config.RepeatCount

	// Map DayOfWeek to time.Weekday
	dayMap := map[DayOfWeek]time.Weekday{
		Monday:    time.Monday,
		Tuesday:   time.Tuesday,
		Wednesday: time.Wednesday,
		Thursday:  time.Thursday,
		Friday:    time.Friday,
		Saturday:  time.Saturday,
		Sunday:    time.Sunday,
	}

	// Create a set of target weekdays
	targetDays := make(map[time.Weekday]bool)
	for _, d := range s.config.Days {
		targetDays[dayMap[d]] = true
	}

	// Find all matching days starting from start date
	for {
		// Check if we've exceeded end date
		if endDate != nil && current.After(*endDate) {
			break
		}

		// If this day matches one of our target days, add it
		if targetDays[current.Weekday()] {
			times = append(times, current)

			// Check count limit (if count is set and positive)
			if count > 0 && len(times) >= count {
				break
			}
		}

		// Move to next day
		current = current.AddDate(0, 0, 1)

		// Safety limit to prevent infinite loops (only if no end date)
		if endDate == nil && current.After(start.AddDate(5, 0, 0)) {
			break
		}
	}

	return times
}

func (s *Scheduler) calculateMonthlyTimes(start time.Time, endDate *time.Time) []time.Time {
	var times []time.Time
	current := start
	count := s.config.RepeatCount

	// If no end date and count <= 0, default to 1
	if endDate == nil && count <= 0 {
		count = 1
	}

	for {
		// Check if we've exceeded end date
		if endDate != nil && current.After(*endDate) {
			break
		}

		times = append(times, current)

		// Check count limit (if count is set and positive)
		if count > 0 && len(times) >= count {
			break
		}

		// Move to next month
		current = current.AddDate(0, 1, 0)

		// Safety limit to prevent infinite loops (only if no end date)
		if endDate == nil && current.After(start.AddDate(10, 0, 0)) {
			break
		}
	}

	return times
}

// Schedule schedules all messages and returns the scheduled message IDs
func (s *Scheduler) Schedule() ([]string, error) {
	times, err := s.CalculateScheduleTimes()
	if err != nil {
		return nil, err
	}

	// Resolve channel ID
	channelID, err := s.client.GetChannelID(s.config.Channel)
	if err != nil {
		return nil, err
	}

	var scheduledIDs []string
	now := time.Now().In(localTZ)

	for _, t := range times {
		// Skip times in the past
		if t.Before(now) {
			fmt.Printf("Skipping past time: %s\n", t.Format("2006-01-02 15:04 MST"))
			continue
		}

		// Slack only allows scheduling up to 120 days in advance
		maxFuture := now.AddDate(0, 0, 120)
		if t.After(maxFuture) {
			fmt.Printf("Skipping time too far in future (>120 days): %s\n", t.Format("2006-01-02 15:04 MST"))
			continue
		}

		fmt.Printf("Scheduling message for: %s\n", t.Format("2006-01-02 15:04 MST"))
		id, err := s.client.ScheduleMessage(channelID, s.config.Message, t)
		if err != nil {
			return scheduledIDs, err
		}
		scheduledIDs = append(scheduledIDs, id)
	}

	// Verify messages were actually scheduled by listing them
	fmt.Printf("\nVerifying scheduled messages...\n")
	scheduledMessages, err := s.client.ListScheduledMessages(channelID)
	if err != nil {
		fmt.Printf("Warning: Could not verify scheduled messages: %v\n", err)
	} else {
		fmt.Printf("Found %d scheduled message(s) in channel %s:\n", len(scheduledMessages), channelID)
		for _, msg := range scheduledMessages {
			postAt := time.Unix(int64(msg.PostAt), 0)
			fmt.Printf("  - ID: %s, Scheduled for: %s, Text: %.50s...\n", 
				msg.ID, postAt.Format("2006-01-02 15:04 MST"), msg.Text)
		}
		if len(scheduledMessages) == 0 {
			fmt.Printf("  ⚠️  No scheduled messages found! The message may not have been scheduled.\n")
			fmt.Printf("  Check that:\n")
			fmt.Printf("    1. Your app has 'chat:write' scope (and 'chat:write.public' if posting to public channels)\n")
			fmt.Printf("    2. Your app/bot is a member of the channel\n")
			fmt.Printf("    3. The scheduled time is in the future\n")
		}
	}

	return scheduledIDs, nil
}
