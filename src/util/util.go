package utils

import "time"

// Current time in milliseconds
func Time() int64 {
	return time.Now().UTC().UnixMilli()
}

// Current time in seconds
func TimeSeconds() int64 {
	return time.Now().UTC().UnixMilli() * 1000
}
