package requests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

func Post(url string, body any) (any, error) {
	bodyJson, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("error while marshalling %v: %w", body, err)
	}
	bodyBytes := bytes.NewBuffer(bodyJson)
	resp, err := http.Post(url, "application/json", bodyBytes)
	if err != nil {
		return nil, fmt.Errorf("error during request to url %v with body %v: %w", url, bodyJson, err)
	}
	defer resp.Body.Close()

	var out any
	err = json.NewDecoder(resp.Body).Decode(&out)
	if err != nil {
		return nil, fmt.Errorf("error during decoding of response to request to url %v with body %v and response %v, %w", url, bodyJson, resp, err)
	}

	return out, nil
}
