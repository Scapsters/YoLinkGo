package requests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// Post form to url. Ensures JSON encoding, distinct from forms
func PostJson[T any](urlString string, headers map[string]string, body map[string]any) (*T, error) {
	// Buld request
	bodyJson, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("error while marshalling %v: %w", body, err)
	}
	req, err := http.NewRequest("POST", urlString, bytes.NewBuffer(bodyJson))
	if err != nil {
		return nil, fmt.Errorf("error while building request: %w", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	// Do request
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error during request to url %v with body %v: %w", urlString, bodyJson, err)
	}
	if response.StatusCode != 200 {
		return nil, fmt.Errorf("error during request to url %v with body %v: %v", urlString, body, response.Status)
	}
	defer response.Body.Close()
	// Cast to type
	var out *T
	err = json.NewDecoder(response.Body).Decode(&out)
	if err != nil {
		return nil, fmt.Errorf("error during decoding of response to request to url %v with body %v and response %v, %w", urlString, bodyJson, response, err)
	}
	return out, nil
}

// Post form to url. Ensures form encoding, distinct from JSON
func PostForm[T any](urlString string, body map[string]string) (*T, error) {
	formValues := url.Values{}
	for k, val := range body {
		formValues.Set(k, val)
	}
	response, err := http.Post(
		urlString, "application/x-www-form-urlencoded",
		strings.NewReader(formValues.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf("error during request to url %v with body %v: %w", urlString, body, err)
	}
	if response.StatusCode != 200 {
		return nil, fmt.Errorf("error during request to url %v with body %v: %v", urlString, body, response.Status)
	}
	defer response.Body.Close()

	var out *T
	err = json.NewDecoder(response.Body).Decode(&out)
	if err != nil {
		return nil, fmt.Errorf("error during decoding of response to request to url %v with body %v and response %v, %w", urlString, body, response, err)
	}

	return out, nil
}
