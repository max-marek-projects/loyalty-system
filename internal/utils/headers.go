package utils

import (
	"strconv"
	"time"
)

// ParseRetryAfter converts a Retry-After header value (seconds or RFC1123 date) into a time.Duration.
// Returns 0 if parsing fails or if the duration is in the past.
func ParseRetryAfter(headerVal string) time.Duration {
	if headerVal == "" {
		return 0
	}
	// Format 1: as delta-seconds (integer)
	if seconds, err := strconv.Atoi(headerVal); err == nil {
		if seconds < 0 {
			return 0
		}
		return time.Duration(seconds) * time.Second
	}
	// Format 2: as an HTTP-date (RFC1123)
	if targetTime, err := time.Parse(time.RFC1123, headerVal); err == nil {
		waitDuration := time.Until(targetTime)
		if waitDuration < 0 {
			return 0
		}
		return waitDuration
	}

	return 0
}
