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
func PostJson(url string, body map[string]any) (map[string]any, error) {
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

	var out map[string]any
	err = json.NewDecoder(resp.Body).Decode(&out)
	if err != nil {
		return nil, fmt.Errorf("error during decoding of response to request to url %v with body %v and response %v, %w", url, bodyJson, resp, err)
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
	defer response.Body.Close()

	var out *T
	err = json.NewDecoder(response.Body).Decode(&out)
	if err != nil {
		return nil, fmt.Errorf("error during decoding of response to request to url %v with body %v and response %v, %w", urlString, body, response, err)
	}

	return out, nil
}