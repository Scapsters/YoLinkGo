package utils

import (
	"encoding/json"
	"fmt"
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

func ToMap[T any](data any) (map[string]T, error) {
	marshalled, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("error while marshalling %v: %w", data, err)
	}
	var mapped map[string]T
	if err := json.Unmarshal(marshalled, &mapped); err != nil {
		return nil, fmt.Errorf("error while unmarshalling %v with intermediate value %v: %w", data, marshalled, err)
	}
	return mapped, nil
}
