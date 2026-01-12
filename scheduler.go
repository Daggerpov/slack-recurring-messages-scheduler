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

	var times []time.Time

	switch s.config.Interval {
	case IntervalNone:
		// Single message
		times = append(times, startDateTime)

	case IntervalDaily:
		times = s.calculateDailyTimes(startDateTime)

	case IntervalWeekly:
		times = s.calculateWeeklyTimes(startDateTime)

	case IntervalMonthly:
		times = s.calculateMonthlyTimes(startDateTime)

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

func (s *Scheduler) calculateDailyTimes(start time.Time) []time.Time {
	count := s.config.RepeatCount
	if count <= 0 {
		count = 1
	}

	times := make([]time.Time, count)
	for i := 0; i < count; i++ {
		times[i] = start.AddDate(0, 0, i)
	}
	return times
}

func (s *Scheduler) calculateWeeklyTimes(start time.Time) []time.Time {
	count := s.config.RepeatCount
	if count <= 0 {
		count = 1
	}

	var times []time.Time

	// If specific days are specified, use them
	if len(s.config.Days) > 0 {
		times = s.calculateSpecificDaysTimes(start, count)
	} else {
		// Otherwise, repeat on the same day of week
		for i := 0; i < count; i++ {
			times = append(times, start.AddDate(0, 0, i*7))
		}
	}

	return times
}

func (s *Scheduler) calculateSpecificDaysTimes(start time.Time, totalOccurrences int) []time.Time {
	var times []time.Time
	current := start

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
	for len(times) < totalOccurrences {
		if targetDays[current.Weekday()] {
			times = append(times, current)
		}
		current = current.AddDate(0, 0, 1)

		// Safety limit to prevent infinite loops
		if current.After(start.AddDate(5, 0, 0)) {
			break
		}
	}

	return times
}

func (s *Scheduler) calculateMonthlyTimes(start time.Time) []time.Time {
	count := s.config.RepeatCount
	if count <= 0 {
		count = 1
	}

	times := make([]time.Time, count)
	for i := 0; i < count; i++ {
		times[i] = start.AddDate(0, i, 0)
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

	return scheduledIDs, nil
}
