package types

import (
	"fmt"
	"strings"
)

// Interval represents the repeat interval type
type Interval string

const (
	IntervalNone    Interval = "none"
	IntervalDaily   Interval = "daily"
	IntervalWeekly  Interval = "weekly"
	IntervalMonthly Interval = "monthly"
)

// ValidIntervals for validation
var ValidIntervals = []Interval{IntervalNone, IntervalDaily, IntervalWeekly, IntervalMonthly}

func (i Interval) IsValid() bool {
	for _, v := range ValidIntervals {
		if i == v {
			return true
		}
	}
	return false
}

// DayOfWeek represents days of the week
type DayOfWeek string

const (
	Monday    DayOfWeek = "monday"
	Tuesday   DayOfWeek = "tuesday"
	Wednesday DayOfWeek = "wednesday"
	Thursday  DayOfWeek = "thursday"
	Friday    DayOfWeek = "friday"
	Saturday  DayOfWeek = "saturday"
	Sunday    DayOfWeek = "sunday"
)

// Short day names for CLI convenience
var DayShortNames = map[string]DayOfWeek{
	"mon": Monday,
	"tue": Tuesday,
	"wed": Wednesday,
	"thu": Thursday,
	"fri": Friday,
	"sat": Saturday,
	"sun": Sunday,
}

var DayFullNames = map[string]DayOfWeek{
	"monday":    Monday,
	"tuesday":   Tuesday,
	"wednesday": Wednesday,
	"thursday":  Thursday,
	"friday":    Friday,
	"saturday":  Saturday,
	"sunday":    Sunday,
}

func ParseDayOfWeek(s string) (DayOfWeek, error) {
	lower := strings.ToLower(s)
	if d, ok := DayShortNames[lower]; ok {
		return d, nil
	}
	if d, ok := DayFullNames[lower]; ok {
		return d, nil
	}
	return "", fmt.Errorf("invalid day of week: %s (use: mon,tue,wed,thu,fri,sat,sun)", s)
}

func ParseDaysOfWeek(s string) ([]DayOfWeek, error) {
	if s == "" {
		return nil, nil
	}
	parts := strings.Split(s, ",")
	days := make([]DayOfWeek, 0, len(parts))
	for _, p := range parts {
		d, err := ParseDayOfWeek(strings.TrimSpace(p))
		if err != nil {
			return nil, err
		}
		days = append(days, d)
	}
	return days, nil
}

// ScheduleConfig holds all scheduling configuration
type ScheduleConfig struct {
	// Message content (supports Slack formatting, @mentions, etc.)
	Message string `json:"message"`

	// Channel ID or name to send to
	Channel string `json:"channel"`

	// Start date in YYYY-MM-DD format
	StartDate string `json:"start_date"`

	// Time to send in HH:MM format (24-hour, local time)
	SendTime string `json:"send_time"`

	// Repeat interval
	Interval Interval `json:"interval"`

	// Number of times to repeat (0 = once/no repeat, -1 = infinite)
	// If EndDate is also set, will stop at whichever comes first
	RepeatCount int `json:"repeat_count"`

	// End date in YYYY-MM-DD format (optional)
	// If set, recurrence will stop on or before this date
	EndDate string `json:"end_date,omitempty"`

	// Specific days of week (for weekly interval)
	Days []DayOfWeek `json:"days,omitempty"`
}

// Credentials holds Slack API credentials
type Credentials struct {
	// Slack Bot Token (starts with xoxb-) or User Token (starts with xoxp-)
	Token string `json:"token"`
}
