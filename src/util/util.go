package utils

import "time"

func Time() int64 {
	return time.Now().UTC().UnixMilli()
}
