// Package newyear calculates the number of calendar days left until the
// next New Year. It has no dependencies on HTTP or any other transport.
package newyear

import "time"

const hoursPerDay = 24

// Next returns midnight of January 1 of the year following the date of t,
// in the location of t.
func Next(t time.Time) time.Time {
	return time.Date(t.Year()+1, time.January, 1, 0, 0, 0, 0, t.Location())
}

// DaysUntil returns the number of calendar days from the date of t to
// January 1 of the next year. The time of day is ignored and the calendar
// date is taken in the location of t, so December 31 gives 1 and January 1
// gives 365 (366 in a leap year).
func DaysUntil(t time.Time) int {
	year, month, day := t.Date()

	// Calculations are done in UTC, where every day lasts exactly 24 hours,
	// so daylight saving time transitions cannot shift the result.
	from := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	to := time.Date(year+1, time.January, 1, 0, 0, 0, 0, time.UTC)

	return int(to.Sub(from).Hours() / hoursPerDay)
}
