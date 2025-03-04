package elk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	//"github.com/byuoitav/common/log"
	//"github.com/byuoitav/common/nerr"
	customerror "github.com/byuoitav/event-forwarding-microservice/error"
)

// VAR
var (
	APIAddr  = os.Getenv("ELK_DIRECT_ADDRESS") // or should this be ELK_ADDR?
	username = os.Getenv("ELK_SA_USERNAME")
	password = os.Getenv("ELK_SA_PASSWORD")
)

// ElkBulkUpdateItem .
// USED
type ElkBulkUpdateItem struct {
	Index  ElkUpdateHeader
	Delete ElkDeleteHeader
	Doc    interface{}
}

// ElkDeleteHeader .
// USED
type ElkDeleteHeader struct {
	Header HeaderIndex `json:"delete"`
}

// ElkUpdateHeader .
// USED
type ElkUpdateHeader struct {
	Header HeaderIndex `json:"index"`
}

// HeaderIndex .
// USED
type HeaderIndex struct {
	Index string `json:"_index"`
	Type  string `json:"_type"`
	ID    string `json:"_id,omitempty"`
}

// BulkUpdateResponse there are other types, but we don't worry about them,
// since we don't really do any smart parsing at this time.
type BulkUpdateResponse struct {
	Errors bool `json:"errors"`
}

// MakeGenericELKRequest .
func MakeGenericELKRequest(addr, method string, body interface{}, user, pass string) ([]byte, error) {
	slog.Debug("Making ELK request against system", addr, "DEBUG")

	if len(user) == 0 || len(pass) == 0 {
		if len(username) == 0 || len(password) == 0 {
			slog.Error("ELK_SA_USERNAME, or ELK_SA_PASSWORD is not set.")
			os.Exit(1)
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
			return []byte{}, err
		}
	}
	debugLog := fmt.Sprintf("Body: %s", reqBody)
	slog.Debug(debugLog)

	// create the request
	req, err := http.NewRequest(method, addr, bytes.NewReader(reqBody))
	if err != nil {
		return []byte{}, err
	}

	if len(user) == 0 || len(pass) == 0 {
		// add auth
		req.SetBasicAuth(username, password)
	} else {
		req.SetBasicAuth(user, pass)
	}

	// add headers
	if method == http.MethodPost || method == http.MethodPut {
		req.Header.Add("content-type", "application/json")
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

	// read the resp
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Info("Error reading response body", err.Error(), "Error")
		return []byte{}, err
	}

	// check resp code
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

// BulkForward preps a bulk request and forwards it.
// Leave user and pass blank to use the env variables defined above.
// USED
func BulkForward(caller, url, user, pass string, toSend []ElkBulkUpdateItem) {
	if len(toSend) == 0 {
		return
	}
	slog.Info("%v Sending bulk upsert for %v items.", caller, len(toSend))
	//DEBUG
	/*
		for i := range toSend {
			log.L.Debugf("%+v", toSend[i])
		}
	*/

	slog.Debug("%v Building payload", caller, "DEBUG")
	//build our payload
	payload := []byte{}
	for i := range toSend {
		var headerbytes []byte
		var err error

		if len(toSend[i].Delete.Header.Index) > 0 { //it's a delete
			headerbytes, err = json.Marshal(toSend[i].Delete)
			if err != nil {
				slog.Error("%v Couldn't marshal header for elk event bulk update: %v", caller, toSend[i])
				continue
			}
			payload = append(payload, headerbytes...)
			payload = append(payload, '\n')

		} else { // do our base case
			headerbytes, err = json.Marshal(toSend[i].Index)
			if err != nil {
				slog.Error("%v Couldn't marshal header for elk event bulk update: %v", caller, toSend[i])
				continue
			}

			bodybytes, err := json.Marshal(toSend[i].Doc)
			if err != nil {
				slog.Error("%v Couldn't marshal header for elk event bulk update: %v", caller, toSend[i])
				continue
			}
			payload = append(payload, headerbytes...)
			payload = append(payload, '\n')
			payload = append(payload, bodybytes...)
			payload = append(payload, '\n')

		}
	}

	//once our payload is built
	slog.Debug("%v Payload built, sending...", caller, "DEBUG")
	//log.L.Debugf("%s", payload)

	url = strings.Trim(url, "/")         //remove any trailing slash so we can append it again
	addr := fmt.Sprintf("%v/_bulk", url) //make the addr

	// Make the request
	resp, er := MakeGenericELKRequest(addr, "POST", payload, user, pass)
	if er != nil {
		slog.Error("%v Couldn't send bulk update. error %v", caller, er.Error())
		return
	}

	elkresp := BulkUpdateResponse{}

	err := json.Unmarshal(resp, &elkresp)
	if err != nil {
		slog.Error("%v Unknown response received from ELK in response to bulk update: %s", caller, resp)
		return
	}
	if elkresp.Errors {
		slog.Error("%v Errors received from ELK during bulk update %s", caller, resp)
		return
	}
	slog.Debug("%v Successfully sent bulk ELK updates", caller, "DEBUG")
}
