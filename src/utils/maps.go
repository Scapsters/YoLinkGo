package utils

import (
	"encoding/json"
	"fmt"
)

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

type KVPair struct {
	K string
	V string
}

// Traverse a map m where all keys and nested keys are strings, and values are strings or maps of strings, adding all key value pairs to array a
// keyPrefix will prefix all keys with the given string. External callers can provide ""
func TraverseMap(m map[string]any, a []KVPair, keyPrefix string) []KVPair {
	if keyPrefix != "" {
		keyPrefix += "."
	}
	for k, v := range m {
		switch v := v.(type) {
		case map[string]any:
			a = append(a, TraverseMap(v, []KVPair{}, keyPrefix+k)...)
		default:
			a = append(a, KVPair{K: keyPrefix + k, V: fmt.Sprint(v)})
		}
	}
	return a
}