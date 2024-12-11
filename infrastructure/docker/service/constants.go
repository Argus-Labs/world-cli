package service

import "time"

const (
	// Service Health Check
	DefaultRetries  = 20
	DefaultTimeout  = time.Second
	DefaultInterval = 3 * time.Second

	// Platform Check
	PlatformPartCount = 2

	// Database Settings
	DBRetries  = 5
	DBTimeout  = 3 * time.Second
	DBInterval = 3 * time.Second
)
