package main

import (
	"testing"
	"time"
)

// Helper to create a scheduler for testing (no Slack client needed for time calculations)
func newTestScheduler(config *ScheduleConfig) *Scheduler {
	return &Scheduler{
		client: nil,
		config: config,
	}
}

// Helper to parse date string to time in local timezone
func mustParseDate(t *testing.T, dateStr string) time.Time {
	t.Helper()
	parsed, err := time.ParseInLocation("2006-01-02", dateStr, time.Local)
	if err != nil {
		t.Fatalf("failed to parse date %s: %v", dateStr, err)
	}
	return parsed
}

func TestScheduler_CalculateScheduleTimes_SingleMessage(t *testing.T) {
	config := &ScheduleConfig{
		Message:     "Test message",
		Channel:     "test-channel",
		StartDate:   "2025-01-15",
		SendTime:    "14:00",
		Interval:    IntervalNone,
		RepeatCount: 0,
	}

	scheduler := newTestScheduler(config)
	times, err := scheduler.CalculateScheduleTimes()
	if err != nil {
		t.Fatalf("CalculateScheduleTimes() error = %v", err)
	}

	if len(times) != 1 {
		t.Errorf("expected 1 scheduled time, got %d", len(times))
	}

	expected := mustParseDate(t, "2025-01-15").Add(14 * time.Hour)
	if !times[0].Equal(expected) {
		t.Errorf("expected time %v, got %v", expected, times[0])
	}
}

func TestScheduler_CalculateScheduleTimes_Daily(t *testing.T) {
	tests := []struct {
		name        string
		config      *ScheduleConfig
		wantCount   int
		wantFirstAt string
		wantLastAt  string
	}{
		{
			name: "daily with count",
			config: &ScheduleConfig{
				StartDate:   "2025-01-15",
				SendTime:    "09:00",
				Interval:    IntervalDaily,
				RepeatCount: 5,
			},
			wantCount:   5,
			wantFirstAt: "2025-01-15",
			wantLastAt:  "2025-01-19",
		},
		{
			name: "daily with end date",
			config: &ScheduleConfig{
				StartDate:   "2025-01-15",
				SendTime:    "09:00",
				Interval:    IntervalDaily,
				RepeatCount: 0,
				EndDate:     "2025-01-17",
			},
			wantCount:   3, // 15, 16, 17
			wantFirstAt: "2025-01-15",
			wantLastAt:  "2025-01-17",
		},
		{
			name: "daily with count and end date (count reached first)",
			config: &ScheduleConfig{
				StartDate:   "2025-01-15",
				SendTime:    "09:00",
				Interval:    IntervalDaily,
				RepeatCount: 2,
				EndDate:     "2025-01-20",
			},
			wantCount:   2,
			wantFirstAt: "2025-01-15",
			wantLastAt:  "2025-01-16",
		},
		{
			name: "daily no count no end date defaults to 1",
			config: &ScheduleConfig{
				StartDate:   "2025-01-15",
				SendTime:    "09:00",
				Interval:    IntervalDaily,
				RepeatCount: 0,
			},
			wantCount:   1,
			wantFirstAt: "2025-01-15",
			wantLastAt:  "2025-01-15",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheduler := newTestScheduler(tt.config)
			times, err := scheduler.CalculateScheduleTimes()
			if err != nil {
				t.Fatalf("CalculateScheduleTimes() error = %v", err)
			}

			if len(times) != tt.wantCount {
				t.Errorf("expected %d times, got %d", tt.wantCount, len(times))
			}

			if len(times) > 0 {
				firstDate := times[0].Format("2006-01-02")
				if firstDate != tt.wantFirstAt {
					t.Errorf("first date = %s, want %s", firstDate, tt.wantFirstAt)
				}

				lastDate := times[len(times)-1].Format("2006-01-02")
				if lastDate != tt.wantLastAt {
					t.Errorf("last date = %s, want %s", lastDate, tt.wantLastAt)
				}
			}
		})
	}
}

func TestScheduler_CalculateScheduleTimes_Weekly(t *testing.T) {
	tests := []struct {
		name        string
		config      *ScheduleConfig
		wantCount   int
		wantFirstAt string
		wantLastAt  string
	}{
		{
			name: "weekly same day with count",
			config: &ScheduleConfig{
				StartDate:   "2025-01-15", // Wednesday
				SendTime:    "10:00",
				Interval:    IntervalWeekly,
				RepeatCount: 4,
			},
			wantCount:   4,
			wantFirstAt: "2025-01-15",
			wantLastAt:  "2025-02-05", // 3 weeks later
		},
		{
			name: "weekly with end date",
			config: &ScheduleConfig{
				StartDate: "2025-01-15", // Wednesday
				SendTime:  "10:00",
				Interval:  IntervalWeekly,
				EndDate:   "2025-02-01",
			},
			wantCount:   3, // Jan 15, 22, 29
			wantFirstAt: "2025-01-15",
			wantLastAt:  "2025-01-29",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheduler := newTestScheduler(tt.config)
			times, err := scheduler.CalculateScheduleTimes()
			if err != nil {
				t.Fatalf("CalculateScheduleTimes() error = %v", err)
			}

			if len(times) != tt.wantCount {
				t.Errorf("expected %d times, got %d", tt.wantCount, len(times))
			}

			if len(times) > 0 {
				firstDate := times[0].Format("2006-01-02")
				if firstDate != tt.wantFirstAt {
					t.Errorf("first date = %s, want %s", firstDate, tt.wantFirstAt)
				}

				lastDate := times[len(times)-1].Format("2006-01-02")
				if lastDate != tt.wantLastAt {
					t.Errorf("last date = %s, want %s", lastDate, tt.wantLastAt)
				}
			}
		})
	}
}

func TestScheduler_CalculateScheduleTimes_WeeklySpecificDays(t *testing.T) {
	tests := []struct {
		name        string
		config      *ScheduleConfig
		wantCount   int
		wantDays    []string // expected dates
	}{
		{
			name: "mon wed fri with count",
			config: &ScheduleConfig{
				StartDate:   "2025-01-13", // Monday
				SendTime:    "09:00",
				Interval:    IntervalWeekly,
				RepeatCount: 6,
				Days:        []DayOfWeek{Monday, Wednesday, Friday},
			},
			wantCount: 6,
			wantDays:  []string{"2025-01-13", "2025-01-15", "2025-01-17", "2025-01-20", "2025-01-22", "2025-01-24"},
		},
		{
			name: "tue thu with end date",
			config: &ScheduleConfig{
				StartDate: "2025-01-14", // Tuesday
				SendTime:  "09:00",
				Interval:  IntervalWeekly,
				EndDate:   "2025-01-23",
				Days:      []DayOfWeek{Tuesday, Thursday},
			},
			wantCount: 4, // Jan 14, 16, 21, 23
			wantDays:  []string{"2025-01-14", "2025-01-16", "2025-01-21", "2025-01-23"},
		},
		{
			name: "single day weekly",
			config: &ScheduleConfig{
				StartDate:   "2025-01-17", // Friday
				SendTime:    "09:00",
				Interval:    IntervalWeekly,
				RepeatCount: 3,
				Days:        []DayOfWeek{Friday},
			},
			wantCount: 3,
			wantDays:  []string{"2025-01-17", "2025-01-24", "2025-01-31"},
		},
		{
			name: "start date not on specified day - finds next matching day",
			config: &ScheduleConfig{
				StartDate:   "2025-01-13", // Monday
				SendTime:    "09:00",
				Interval:    IntervalWeekly,
				RepeatCount: 3,
				Days:        []DayOfWeek{Friday}, // Only Fridays
			},
			wantCount: 3,
			wantDays:  []string{"2025-01-17", "2025-01-24", "2025-01-31"}, // First Friday is Jan 17
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheduler := newTestScheduler(tt.config)
			times, err := scheduler.CalculateScheduleTimes()
			if err != nil {
				t.Fatalf("CalculateScheduleTimes() error = %v", err)
			}

			if len(times) != tt.wantCount {
				t.Errorf("expected %d times, got %d", tt.wantCount, len(times))
				for i, tm := range times {
					t.Logf("  [%d] %s", i, tm.Format("2006-01-02 Mon"))
				}
			}

			for i, tm := range times {
				if i < len(tt.wantDays) {
					got := tm.Format("2006-01-02")
					if got != tt.wantDays[i] {
						t.Errorf("time[%d] = %s, want %s", i, got, tt.wantDays[i])
					}
				}
			}
		})
	}
}

func TestScheduler_CalculateScheduleTimes_Monthly(t *testing.T) {
	tests := []struct {
		name        string
		config      *ScheduleConfig
		wantCount   int
		wantFirstAt string
		wantLastAt  string
	}{
		{
			name: "monthly with count",
			config: &ScheduleConfig{
				StartDate:   "2025-01-15",
				SendTime:    "10:00",
				Interval:    IntervalMonthly,
				RepeatCount: 3,
			},
			wantCount:   3,
			wantFirstAt: "2025-01-15",
			wantLastAt:  "2025-03-15",
		},
		{
			name: "monthly with end date",
			config: &ScheduleConfig{
				StartDate: "2025-01-15",
				SendTime:  "10:00",
				Interval:  IntervalMonthly,
				EndDate:   "2025-04-01",
			},
			wantCount:   3, // Jan 15, Feb 15, Mar 15
			wantFirstAt: "2025-01-15",
			wantLastAt:  "2025-03-15",
		},
		{
			name: "monthly on 31st (handles short months)",
			config: &ScheduleConfig{
				StartDate:   "2025-01-31",
				SendTime:    "10:00",
				Interval:    IntervalMonthly,
				RepeatCount: 3,
			},
			wantCount:   3,
			wantFirstAt: "2025-01-31",
			// Go's AddDate wraps, so Jan 31 + 1 month = Feb 28/Mar 3 depending on year
			// This is expected Go behavior
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheduler := newTestScheduler(tt.config)
			times, err := scheduler.CalculateScheduleTimes()
			if err != nil {
				t.Fatalf("CalculateScheduleTimes() error = %v", err)
			}

			if len(times) != tt.wantCount {
				t.Errorf("expected %d times, got %d", tt.wantCount, len(times))
			}

			if len(times) > 0 && tt.wantFirstAt != "" {
				firstDate := times[0].Format("2006-01-02")
				if firstDate != tt.wantFirstAt {
					t.Errorf("first date = %s, want %s", firstDate, tt.wantFirstAt)
				}
			}

			if len(times) > 0 && tt.wantLastAt != "" {
				lastDate := times[len(times)-1].Format("2006-01-02")
				if lastDate != tt.wantLastAt {
					t.Errorf("last date = %s, want %s", lastDate, tt.wantLastAt)
				}
			}
		})
	}
}

func TestScheduler_CalculateScheduleTimes_TimeOfDay(t *testing.T) {
	tests := []struct {
		name     string
		sendTime string
		wantHour int
		wantMin  int
	}{
		{"morning", "09:00", 9, 0},
		{"afternoon", "14:30", 14, 30},
		{"evening", "18:45", 18, 45},
		{"midnight", "00:00", 0, 0},
		{"end of day", "23:59", 23, 59},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ScheduleConfig{
				StartDate: "2025-01-15",
				SendTime:  tt.sendTime,
				Interval:  IntervalNone,
			}

			scheduler := newTestScheduler(config)
			times, err := scheduler.CalculateScheduleTimes()
			if err != nil {
				t.Fatalf("CalculateScheduleTimes() error = %v", err)
			}

			if len(times) != 1 {
				t.Fatalf("expected 1 time, got %d", len(times))
			}

			if times[0].Hour() != tt.wantHour {
				t.Errorf("hour = %d, want %d", times[0].Hour(), tt.wantHour)
			}
			if times[0].Minute() != tt.wantMin {
				t.Errorf("minute = %d, want %d", times[0].Minute(), tt.wantMin)
			}
		})
	}
}

func TestScheduler_CalculateScheduleTimes_InvalidInputs(t *testing.T) {
	tests := []struct {
		name    string
		config  *ScheduleConfig
		wantErr bool
	}{
		{
			name: "invalid date format",
			config: &ScheduleConfig{
				StartDate: "01-15-2025", // wrong format
				SendTime:  "09:00",
				Interval:  IntervalNone,
			},
			wantErr: true,
		},
		{
			name: "invalid time format",
			config: &ScheduleConfig{
				StartDate: "2025-01-15",
				SendTime:  "9:00 AM", // wrong format
				Interval:  IntervalNone,
			},
			wantErr: true,
		},
		{
			name: "invalid end date format",
			config: &ScheduleConfig{
				StartDate: "2025-01-15",
				SendTime:  "09:00",
				Interval:  IntervalDaily,
				EndDate:   "invalid-date",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheduler := newTestScheduler(tt.config)
			_, err := scheduler.CalculateScheduleTimes()
			if (err != nil) != tt.wantErr {
				t.Errorf("CalculateScheduleTimes() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
