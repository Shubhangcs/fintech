package utils

import (
	"net/http"
	"time"
)

const DateLayout = "2006-01-02"

func ParseDateParam(r *http.Request, key string) *time.Time {
	val := r.URL.Query().Get(key)
	if val == "" {
		return nil
	}
	t, err := time.Parse(DateLayout, val)
	if err != nil {
		return nil
	}
	return &t
}
