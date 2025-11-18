package utils

import (
	"time"
)

// Current time in milliseconds
func Time() int64 {
	return time.Now().UTC().UnixMilli()
}

// Current time in seconds
func TimeSeconds() int64 {
	return time.Now().UTC().UnixMilli() / 1000
}

// Epoch seconds into Excel-readable date string
func EpochMillisecondsToExcelDate(seconds int64) string {
	return time.Unix(seconds/1000, seconds%1000).UTC().Format("2006-01-02 15:04:05")
}
