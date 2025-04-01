package elk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

// CONST
const (
	ALERTING_TRUE  = 1
	ALERTING_FALSE = 0
	POWER_STANDBY  = "standby"
	POWER_ON       = "on"
)

// VAR
var (
	APIAddr  = os.Getenv("ELK_DIRECT_ADDRESS") // or should this be ELK_ADDR?
	username = os.Getenv("ELK_SA_USERNAME")
	password = os.Getenv("ELK_SA_PASSWORD")
)

// ElkBulkUpdateItem .
type ElkBulkUpdateItem struct {
	Index  ElkUpdateHeader
	Delete ElkDeleteHeader
	Doc    interface{}
}

// ElkDeleteHeader .
type ElkDeleteHeader struct {
	Header HeaderIndex `json:"delete"`
}

// ElkUpdateHeader .
type ElkUpdateHeader struct {
	Header HeaderIndex `json:"index"`
}

// HeaderIndex .
type HeaderIndex struct {
	Index string `json:"_index"`
	ID    string `json:"_id,omitempty"`
}

// BulkUpdateResponse there are other types, but we don't worry about them,
// since we don't really do any smart parsing at this time.
type BulkUpdateResponse struct {
	Errors bool `json:"errors"`
}

// MakeGenericELKRequest .
func MakeGenericELKRequest(addr, method string, body interface{}, user, pass string) ([]byte, error) {
	slog.Debug("Making ELK request", "addr", addr)

	if len(user) == 0 || len(pass) == 0 {
		if len(username) == 0 || len(password) == 0 {
			slog.Error("ELK_SA_USERNAME or ELK_SA_PASSWORD is not set")
		}
	}

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
			return []byte{}, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	// create the request
	req, err := http.NewRequest(method, addr, bytes.NewReader(reqBody))
	if err != nil {
		return []byte{}, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	if len(user) == 0 || len(pass) == 0 {
		// add auth
		req.SetBasicAuth(username, password)
	} else {
		req.SetBasicAuth(user, pass)
	}

	// add headers
	if method == http.MethodPost || method == http.MethodPut {
		req.Header.Add("content-type", "application/x-ndjson")
	}

	client := http.Client{
		Timeout: 3 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// read the resp
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to read response body: %w", err)
	}

	// check resp code
	if resp.StatusCode/100 != 2 {
		msg := fmt.Sprintf("non 200 response code received. code: %v, body: %s", resp.StatusCode, respBody)
		return respBody, fmt.Errorf(msg)
	}

	return respBody, nil
}

// MakeELKRequest .
func MakeELKRequest(method, endpoint string, body interface{}) ([]byte, error) {
	if len(APIAddr) == 0 {
		slog.Error("ELK_DIRECT_ADDRESS is not set")
	}

	// format whole address
	addr := fmt.Sprintf("%s%s", APIAddr, endpoint)
	return MakeGenericELKRequest(addr, method, body, "", "")
}

// BulkForward preps a bulk request and forwards it.
// Leave user and pass blank to use the env variables defined above.
func BulkForward(caller, url, user, pass string, toSend []ElkBulkUpdateItem) {
	if len(toSend) == 0 {
		return
	}
	slog.Info("Sending bulk upsert", "caller", caller, "items", len(toSend))

	slog.Debug("Building payload", "caller", caller)
	// build our payload
	payload := []byte{}
	for i := range toSend {
		var headerbytes []byte
		var err error

		if len(toSend[i].Delete.Header.Index) > 0 { // it's a delete
			headerbytes, err = json.Marshal(toSend[i].Delete)
			if err != nil {
				slog.Error("Couldn't marshal header for elk event bulk update", "caller", caller, "item", toSend[i])
				continue
			}
			payload = append(payload, headerbytes...)
			payload = append(payload, '\n')

		} else { // do our base case
			headerbytes, err = json.Marshal(toSend[i].Index)
			if err != nil {
				slog.Error("Couldn't marshal header for elk event bulk update", "caller", caller, "item", toSend[i])
				continue
			}

			bodybytes, err := json.Marshal(toSend[i].Doc)
			if err != nil {
				slog.Error("Couldn't marshal body for elk event bulk update", "caller", caller, "item", toSend[i])
				continue
			}
			payload = append(payload, headerbytes...)
			payload = append(payload, '\n')
			payload = append(payload, bodybytes...)
			payload = append(payload, '\n')
		}
	}

	payload = append(payload, '\n') // Ensure the final newline

	slog.Debug("Payload built", "caller", caller, "payload", string(payload))

	// once our payload is built
	slog.Debug("Payload built, sending...", "caller", caller)

	url = strings.Trim(url, "/")         // remove any trailing slash so we can append it again
	addr := fmt.Sprintf("%v/_bulk", url) // make the addr

	resp, err := MakeGenericELKRequest(addr, "POST", payload, user, pass)
	if err != nil {
		slog.Error("Couldn't send bulk update", "caller", caller, "error", err.Error())
		return
	}

	elkresp := BulkUpdateResponse{}

	err = json.Unmarshal(resp, &elkresp)
	if err != nil {
		slog.Error("Unknown response received from ELK in response to bulk update", "caller", caller, "response", string(resp))
		return
	}
	if elkresp.Errors {
		slog.Error("Errors received from ELK during bulk update", "caller", caller, "response", string(resp))
		return
	}
	slog.Debug("Successfully sent bulk ELK updates", "caller", caller)
}
