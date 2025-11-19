package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const requestTimeout = 5

// Post form to url. Ensures JSON encoding, distinct from forms.
func PostJson[T any](urlString string, headers map[string]string, body map[string]any) (*T, error) {
	// Buld request
	bodyJson, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("error while marshalling %v: %w", body, err)
	}
	bodyBytes := bytes.NewBuffer(bodyJson)
	context, cancel := TimeoutContext(requestTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(
		context,
		http.MethodPost,
		urlString,
		bodyBytes,
	)
	if err != nil {
		return nil, fmt.Errorf("error while building request with url %v and body %v: %w", urlString, bodyBytes, err)
	}
	for k, v := range headers {
		request.Header.Set(k, v)
	}

	// Do request
	client := &http.Client{}
	return interpretResponse[T](client.Do(request))
}

// Post form to url. Ensures form encoding, distinct from JSON.
func PostForm[T any](urlString string, body map[string]string) (*T, error) {
	// Build request
	formValues := url.Values{}
	for k, val := range body {
		formValues.Set(k, val)
	}
	context, cancel := TimeoutContext(requestTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(
		context,
		http.MethodPost,
		urlString, 
		strings.NewReader(formValues.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf("error while building request with url %v and form values %v: %w", urlString, formValues, err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Do request
	client := &http.Client{}
	return interpretResponse[T](client.Do(request))
}

func interpretResponse[T any](response *http.Response, err error) (*T, error) {
	// Check statuses
	if err != nil {
		return nil, fmt.Errorf("error during request %v: %w", response, err)
	}
	OKStatus := 200
	if response.StatusCode != OKStatus {
		return nil, fmt.Errorf("non-200 code received %v: %v", response, response.Status)
	}
	defer LogErrors(response.Body.Close, fmt.Sprintf("Closing body %v", response.Body))
	// Cast to type
	var out *T
	err = json.NewDecoder(response.Body).Decode(&out)
	if err != nil {
		return nil, fmt.Errorf("error during decoding of response %v, %w", response.Body, err)
	}

	return out, nil
}
