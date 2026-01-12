package main

import (
	"testing"
)

func TestInterval_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		interval Interval
		want     bool
	}{
		{"none is valid", IntervalNone, true},
		{"daily is valid", IntervalDaily, true},
		{"weekly is valid", IntervalWeekly, true},
		{"monthly is valid", IntervalMonthly, true},
		{"empty string is invalid", Interval(""), false},
		{"random string is invalid", Interval("yearly"), false},
		{"DAILY uppercase is invalid", Interval("DAILY"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.interval.IsValid(); got != tt.want {
				t.Errorf("Interval.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseDayOfWeek(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    DayOfWeek
		wantErr bool
	}{
		// Short names
		{"mon short", "mon", Monday, false},
		{"tue short", "tue", Tuesday, false},
		{"wed short", "wed", Wednesday, false},
		{"thu short", "thu", Thursday, false},
		{"fri short", "fri", Friday, false},
		{"sat short", "sat", Saturday, false},
		{"sun short", "sun", Sunday, false},

		// Full names
		{"monday full", "monday", Monday, false},
		{"tuesday full", "tuesday", Tuesday, false},
		{"wednesday full", "wednesday", Wednesday, false},
		{"thursday full", "thursday", Thursday, false},
		{"friday full", "friday", Friday, false},
		{"saturday full", "saturday", Saturday, false},
		{"sunday full", "sunday", Sunday, false},

		// Case insensitive
		{"MON uppercase", "MON", Monday, false},
		{"Friday mixed case", "Friday", Friday, false},
		{"WEDNESDAY uppercase", "WEDNESDAY", Wednesday, false},

		// Invalid inputs
		{"empty string", "", "", true},
		{"invalid day", "notaday", "", true},
		{"m single char", "m", "", true},
		{"monday typo", "munday", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDayOfWeek(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDayOfWeek() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseDayOfWeek() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseDaysOfWeek(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []DayOfWeek
		wantErr bool
	}{
		{"empty string", "", nil, false},
		{"single day", "mon", []DayOfWeek{Monday}, false},
		{"two days", "mon,fri", []DayOfWeek{Monday, Friday}, false},
		{"three days", "mon,wed,fri", []DayOfWeek{Monday, Wednesday, Friday}, false},
		{"all weekdays", "mon,tue,wed,thu,fri", []DayOfWeek{Monday, Tuesday, Wednesday, Thursday, Friday}, false},
		{"with spaces", "mon, wed, fri", []DayOfWeek{Monday, Wednesday, Friday}, false},
		{"full names", "monday,wednesday,friday", []DayOfWeek{Monday, Wednesday, Friday}, false},
		{"mixed format", "mon,Wednesday,FRI", []DayOfWeek{Monday, Wednesday, Friday}, false},
		{"invalid day in list", "mon,invalid,fri", nil, true},
		{"single invalid", "invalid", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDaysOfWeek(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDaysOfWeek() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(got) != len(tt.want) {
					t.Errorf("ParseDaysOfWeek() returned %d days, want %d", len(got), len(tt.want))
					return
				}
				for i := range got {
					if got[i] != tt.want[i] {
						t.Errorf("ParseDaysOfWeek()[%d] = %v, want %v", i, got[i], tt.want[i])
					}
				}
			}
		})
	}
}
