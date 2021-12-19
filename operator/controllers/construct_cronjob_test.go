package controllers

import (
	"fmt"
	"testing"
)

func Test_intervalToCrontabSchedule(t *testing.T) {
	panicingTests := []struct {
		name       string
		interval   string
		checkError func(panicString string, t *testing.T)
	}{
		{
			name:     "unparsable duration",
			interval: "unparsable",
			checkError: func(value string, t *testing.T) {
				expected := "time: invalid duration \"unparsable\""
				if value != expected {
					t.Errorf("Paniced with [%s], expected to panic with [%s]", value, expected)
				}
			},
		},
		{
			name:     "duration less than a minute",
			interval: "59s",
			checkError: func(value string, t *testing.T) {
				expected := fmt.Sprintf("Bad duration, must be at least a minute, but got %d minute", 0)
				if value != expected {
					t.Errorf("Paniced with [%s], expected to panic with [%s]", value, expected)
				}
			},
		},
		{
			name:     "duration bigger or equal 24 hours",
			interval: "24h",
			checkError: func(value string, t *testing.T) {
				expected := fmt.Sprintf("Bad duration, must be less than 24 hours, but got %d hours", 24)
				if value != expected {
					t.Errorf("Paniced with [%s], expected to panic with [%s]", value, expected)
				}
			},
		},
	}

	for _, tt := range panicingTests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if r == nil {
					t.Errorf("Expected to panic, but did not panic")
					return
				}
				value, ok := r.(string)
				if !ok {
					t.Errorf("Got panic value type %T, want string", r)
				}
				tt.checkError(value, t)
			}()

			_ = intervalToCrontabSchedule(tt.interval)
		})
	}

	tests := []struct {
		name     string
		interval string
		want     string
	}{
		{
			name:     "1 minute",
			interval: "70s",
			want:     "*/1 * * * *",
		},
		{
			name:     "59 minute",
			interval: "3599s",
			want:     "*/59 * * * *",
		},
		{
			name:     "1 hour",
			interval: "90m",
			want:     "* */1 * * *",
		},
		{
			name:     "23 hour",
			interval: "23h",
			want:     "* */23 * * *",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := intervalToCrontabSchedule(tt.interval)
			if got != tt.want {
				t.Errorf("Got [%s], want [%s]", got, tt.want)
			}
		})
	}
}
