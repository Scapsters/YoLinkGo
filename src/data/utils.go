package data

import "time"

// Epoch seconds into Excel-readable date string.
func EpochSecondsToExcelDate(seconds int64) string {
	return time.Unix(seconds, 0).UTC().Format("2006-01-02 15:04:05")
}
