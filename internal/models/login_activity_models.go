package models

import "time"

type LoginDeviceInfo struct {
	UserAgent        string  `json:"userAgent"`
	Platform         string  `json:"platform"`
	Language         string  `json:"language"`
	ScreenResolution string  `json:"screenResolution"`
	Timezone         string  `json:"timezone"`
	Timestamp        string  `json:"timestamp"`
	Latitude         float64 `json:"latitude"`
	Longitude        float64 `json:"longitude"`
	Accuracy         float64 `json:"accuracy"`
}

type LoginActivity struct {
	LoginID        int64     `json:"login_id"`
	UserID         string    `json:"user_id"`
	UserAgent      string    `json:"user_agent"`
	Platform       string    `json:"platform"`
	Latitude       float64   `json:"latitude"`
	Longitude      float64   `json:"longitude"`
	Accuracy       float64   `json:"accuracy"`
	LoginTimestamp string    `json:"login_timestamp"`
	CreatedAt      time.Time `json:"created_at"`
}
