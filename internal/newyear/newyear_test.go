package newyear

import (
	"testing"
	"time"
)

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func TestDaysUntil(t *testing.T) {
	tests := []struct {
		name string
		in   time.Time
		want int
	}{
		{"start of common year", date(2023, time.January, 1), 365},
		{"start of leap year", date(2024, time.January, 1), 366},
		{"end of common year", date(2023, time.December, 31), 1},
		{"end of leap year", date(2024, time.December, 31), 1},
		{"leap year before Feb 29", date(2024, time.February, 28), 308},
		{"leap year Feb 29", date(2024, time.February, 29), 307},
		{"leap year after Feb 29", date(2024, time.March, 1), 306},
		{"common year Feb 28", date(2023, time.February, 28), 307},
		{"common year Mar 1", date(2023, time.March, 1), 306},
		{"century not leap year", date(2100, time.March, 1), 306},
		{"400-year leap year", date(2000, time.March, 1), 306},
		{"middle of year", date(2025, time.July, 15), 170},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DaysUntil(tt.in); got != tt.want {
				t.Errorf("DaysUntil(%s) = %d, want %d", tt.in.Format(time.DateOnly), got, tt.want)
			}
		})
	}
}

func TestDaysUntilIgnoresTimeOfDay(t *testing.T) {
	morning := time.Date(2024, time.December, 31, 0, 0, 1, 0, time.UTC)
	evening := time.Date(2024, time.December, 31, 23, 59, 59, 0, time.UTC)

	if got := DaysUntil(morning); got != 1 {
		t.Errorf("DaysUntil(morning) = %d, want 1", got)
	}
	if got := DaysUntil(evening); got != 1 {
		t.Errorf("DaysUntil(evening) = %d, want 1", got)
	}
}

func TestDaysUntilUsesDateInOwnLocation(t *testing.T) {
	moscow := time.FixedZone("UTC+3", 3*60*60)
	// 2025-01-01 01:00 in UTC+3 is still 2024-12-31 in UTC.
	in := time.Date(2025, time.January, 1, 1, 0, 0, 0, moscow)

	if got := DaysUntil(in); got != 365 {
		t.Errorf("DaysUntil(%s) = %d, want 365", in, got)
	}
}

func TestNext(t *testing.T) {
	loc := time.FixedZone("UTC+3", 3*60*60)
	in := time.Date(2024, time.June, 10, 15, 30, 0, 0, loc)
	want := time.Date(2025, time.January, 1, 0, 0, 0, 0, loc)

	if got := Next(in); !got.Equal(want) || got.Location() != loc {
		t.Errorf("Next(%s) = %s, want %s", in, got, want)
	}
}

func BenchmarkDaysUntil(b *testing.B) {
	in := date(2024, time.February, 29)
	for b.Loop() {
		DaysUntil(in)
	}
}
