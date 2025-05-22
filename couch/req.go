package couch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log/slog"
	"net/http"
	"os"
	"sync"
)

var (
	username string
	password string
	once     sync.Once
)

func initialize() {
	slog.Info("Initializing couch requests")
	username = os.Getenv("DB_USERNAME")
	password = os.Getenv("DB_PASSWORD")
}

// MakeRequest makes a generic Couch Request
func MakeRequest(addr, method string, body interface{}) ([]byte, error) {
	once.Do(initialize)

	slog.Debug("Making couch request", "addr", addr)

	var reqBody []byte
	var err error

	// marshal request if not already an array of bytes
	switch v := body.(type) {
	case []byte:
		reqBody = v
	default:
		// marshal the request
		reqBody, err = json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	// create the request
	req, err := http.NewRequest(method, addr, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// add auth
	req.SetBasicAuth(username, password)

	// add headers
	if method == http.MethodPost || method == http.MethodPut {
		req.Header.Add("content-type", "application/json")
	}

	client := http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// read the resp
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// check resp code
	if resp.StatusCode/100 != 2 {
		msg := fmt.Sprintf("non 200 response code received. code: %v, body: %s", resp.StatusCode, respBody)
		slog.Error(msg, "statusCode", resp.StatusCode, "body", string(respBody))
		return respBody, fmt.Errorf("HTTP error: %s", msg)
	}

	return respBody, nil
}
