package utils

import (
	"time"
)

// Current time in seconds.
func TimeSeconds() int64 {
	return time.Now().UTC().Unix()
}


