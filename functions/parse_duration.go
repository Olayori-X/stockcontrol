package functions

import (
	"time"
)

// ParseDurationOrDefault tries to parse a duration string (e.g. "30m", "2h", "1d").
// If parsing fails, it returns the provided default duration.
func ParseDurationOrDefault(input string, defaultVal time.Duration) time.Duration {
	// Special handling for days (Go's ParseDuration doesn't support "d")
	if len(input) > 0 && input[len(input)-1] == 'd' {
		days, err := time.ParseDuration(input[:len(input)-1] + "h")
		if err == nil {
			return days * 24 // convert days to hours
		}
	}

	// Try parsing normally
	d, err := time.ParseDuration(input)
	if err != nil {
		return defaultVal
	}
	return d
}
