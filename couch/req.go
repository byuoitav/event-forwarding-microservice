package couch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sync"

	customerror "github.com/byuoitav/event-forwarding-microservice/error"
	//"github.com/byuoitav/common/log"
	//"github.com/byuoitav/common/nerr"
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
// USED to make Couch Requests -> Forwarding/Managers/couch.go
func MakeRequest(addr, method string, body interface{}) ([]byte, error) {
	once.Do(initialize)

	slog.Debug("Making couch request against: %s", addr, "INFO")

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
			return []byte{}, err
		}
	}

	// create the request
	req, err := http.NewRequest(method, addr, bytes.NewReader(reqBody))
	if err != nil {
		slog.Info("Error making new request", err.Error(), "Error")
		return []byte{}, err
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
		slog.Info("Error with response", err.Error(), "Error")
		return []byte{}, err
	}
	defer resp.Body.Close()

	// read the resp
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Info("Error reading response body", err.Error(), "Error")
		return []byte{}, err
	}

	// check resp code
	if resp.StatusCode/100 != 2 {
		msg := fmt.Sprintf("non 200 reponse code received. code: %v, body: %s", resp.StatusCode, respBody)
		slog.Info("Response Code", msg, "INFO")
		wErr := &customerror.WebError{
			StatusCode: resp.StatusCode,
			Message:    "Received a non 200 response code",
		}

		return respBody, wErr
	}

	slog.Info("Response Received", "status_code", resp.StatusCode)
	return respBody, nil

}
