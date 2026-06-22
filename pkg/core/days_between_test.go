package core

import (
	"testing"
	"time"
)

// TestDaysBetween pins the contract of the daysBetween helper. The
// helper takes time.Time values (rather than strings) so DST and
// fixed-zone behaviour can be tested without touching the
// process-wide time.Local.
func TestDaysBetween(t *testing.T) {
	utc := time.UTC

	tests := []struct {
		name  string
		start time.Time
		end   time.Time
		want  int
	}{
		{
			name:  "same instant",
			start: time.Date(2025, 6, 20, 0, 0, 0, 0, utc),
			end:   time.Date(2025, 6, 20, 0, 0, 0, 0, utc),
			want:  0,
		},
		{
			name:  "one day apart in UTC",
			start: time.Date(2025, 6, 20, 0, 0, 0, 0, utc),
			end:   time.Date(2025, 6, 21, 0, 0, 0, 0, utc),
			want:  1,
		},
		{
			name:  "seven days apart in UTC",
			start: time.Date(2025, 6, 13, 0, 0, 0, 0, utc),
			end:   time.Date(2025, 6, 20, 0, 0, 0, 0, utc),
			want:  7,
		},
		{
			// US spring-forward: 2024-03-10 02:00 EST -> 03:00 EDT.
			// A 24-hour wall-clock span from Mar 10 00:00 EST to
			// Mar 11 00:00 EDT is only 23 actual hours. The
			// calendar-date diff is still 1 day.
			name:  "one calendar day across spring-forward DST",
			start: time.Date(2024, 3, 10, 0, 0, 0, 0, mustLoadNY(t)),
			end:   time.Date(2024, 3, 11, 0, 0, 0, 0, mustLoadNY(t)),
			want:  1,
		},
		{
			// US fall-back: 2024-11-03 02:00 EDT -> 01:00 EST.
			// A 48-hour wall-clock span from Nov 2 00:00 EDT to
			// Nov 4 00:00 EST is 49 actual hours. The
			// calendar-date diff is still 2 days.
			name:  "two calendar days across fall-back DST",
			start: time.Date(2024, 11, 2, 0, 0, 0, 0, mustLoadNY(t)),
			end:   time.Date(2024, 11, 4, 0, 0, 0, 0, mustLoadNY(t)),
			want:  2,
		},
		{
			// A multi-day span entirely inside DST, both endpoints
			// at midnight. The actual elapsed time is 48h; the
			// calendar-date diff is 2 days.
			name:  "two calendar days in DST",
			start: time.Date(2024, 7, 15, 0, 0, 0, 0, mustLoadNY(t)),
			end:   time.Date(2024, 7, 17, 0, 0, 0, 0, mustLoadNY(t)),
			want:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := daysBetween(tt.start, tt.end); got != tt.want {
				t.Errorf("daysBetween(%s, %s) = %d, want %d", tt.start, tt.end, got, tt.want)
			}
		})
	}
}

// mustLoadNY returns the America/New_York location and fails the
// test if it cannot be loaded (e.g. on a minimal Go install that
// does not include the tzdata package). America/New_York is
// present in the standard tzdata that ships with Go 1.15+, and
// the project's go.mod requires Go 1.25.
func mustLoadNY(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("LoadLocation(America/New_York) failed: %v", err)
	}
	return loc
}
