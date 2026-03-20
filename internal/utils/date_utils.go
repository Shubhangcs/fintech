package utils

import (
	"net/http"
	"time"
)

const DateLayout = "2006-01-02"

// ParseDateParam parses a date query param (YYYY-MM-DD) and shifts it from
// IST (UTC+5:30) to UTC by subtracting 5h30m, so DB comparisons are correct.
func ParseDateParam(r *http.Request, key string) *time.Time {
	val := r.URL.Query().Get(key)
	if val == "" {
		return nil
	}
	t, err := time.Parse(DateLayout, val)
	if err != nil {
		return nil
	}
	utc := t.Add(-5*time.Hour - 30*time.Minute)
	return &utc
}
