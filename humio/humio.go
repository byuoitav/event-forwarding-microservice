// The Humio package is for making final http requests to Humio
package humio

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	//"github.com/byuoitav/common/log"
	//"github.com/byuoitav/common/nerr"
	"github.com/byuoitav/event-forwarding-microservice/customerror"
)

var (
	APIAddr = os.Getenv("HUMIO_DIRECT_ADDRESS")
)

// sends a http request to humio using the given method, body, and authToken
func MakeGenericHumioRequest(addr, method string, body interface{}, authToken string) ([]byte, error) {
	var reqBody []byte
	var err error

	//marshal request if not already an array of bytes
	switch v := body.(type) {
	case []byte:
		reqBody = v
	default:
		//marshal the request
		reqBody, err = json.Marshal(v)
		if err != nil {
			return []byte{}, err
		}
	}

	//create the request
	req, err := http.NewRequest(method, addr, bytes.NewReader(reqBody))
	if err != nil {
		slog.Info("Error making new request", err.Error(), "Error")
		return []byte{}, err
	}

	// add headers
	if method == http.MethodPost {
		req.Header.Add("content-type", "application/json")
	}
	// humio ingest token
	if method == http.MethodGet || method == http.MethodPost {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", authToken))
	}

	client := http.Client{
		Timeout: 3 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		slog.Info("Error with response", err.Error(), "Error")
		return []byte{}, err
	}
	defer resp.Body.Close()

	//read the resp
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Info("Error reading response body", err.Error(), "Error")
		return []byte{}, err
	}

	if resp.StatusCode/100 != 2 {
		err = &customerror.WebError{
			StatusCode: resp.StatusCode,
			Message:    http.StatusText(resp.StatusCode),
		}
		msg := fmt.Sprintf("non 200 reponse code received. code: %v, body: %s", resp.StatusCode, respBody)
		slog.Error("Non 200 Response code. Code:", msg, "ERROR")
		return respBody, err
	}

	return respBody, nil
}

// MakeHumioRequest sends an http request to humio using a direct address stored in the environment
func MakeHumioRequest(method, endpoint string, body interface{}, authToken string) ([]byte, error) {
	if len(APIAddr) == 0 {
		slog.Error("HUMIO_DIRECT_ADDRESS is not set.")
		os.Exit(1)
	}

	//format whole address
	addr := fmt.Sprintf("%s%s", APIAddr, endpoint)
	return MakeGenericHumioRequest(addr, method, body, authToken)
}
