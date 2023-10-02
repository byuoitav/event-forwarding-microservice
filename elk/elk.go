package elk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/byuoitav/common/log"
	"github.com/byuoitav/common/nerr"
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
	Type  string `json:"_type"`
	ID    string `json:"_id,omitempty"`
}

// BulkUpdateResponse there are other types, but we don't worry about them,
// since we don't really do any smart parsing at this time.
type BulkUpdateResponse struct {
	Errors bool `json:"errors"`
}

// MakeGenericELKRequest .
func MakeGenericELKRequest(addr, method string, body interface{}, user, pass string) ([]byte, *nerr.E) {
	log.L.Debugf("Making ELK request against: %s", addr)

	if len(user) == 0 || len(pass) == 0 {
		if len(username) == 0 || len(password) == 0 {
			log.L.Fatalf("ELK_SA_USERNAME, or ELK_SA_PASSWORD is not set.")
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
			return []byte{}, nerr.Translate(err)
		}
	}
	//	log.L.Debugf("Body: %s", reqBody)

	// create the request
	req, err := http.NewRequest(method, addr, bytes.NewReader(reqBody))
	if err != nil {
		return []byte{}, nerr.Translate(err)
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
		return []byte{}, nerr.Translate(err)
	}
	defer resp.Body.Close()

	// read the resp
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, nerr.Translate(err)
	}

	// check resp code
	if resp.StatusCode/100 != 2 {
		msg := fmt.Sprintf("non 200 reponse code received. code: %v, body: %s", resp.StatusCode, respBody)
		return respBody, nerr.Create(msg, http.StatusText(resp.StatusCode))
	}

	return respBody, nil

}

// MakeELKRequest .
func MakeELKRequest(method, endpoint string, body interface{}) ([]byte, *nerr.E) {
	if len(APIAddr) == 0 {
		log.L.Fatalf("ELK_DIRECT_ADDRESS is not set.")
	}

	// format whole address
	addr := fmt.Sprintf("%s%s", APIAddr, endpoint)
	return MakeGenericELKRequest(addr, method, body, "", "")
}

// BulkForward preps a bulk request and forwards it.
// Leave user and pass blank to use the env variables defined above.
func BulkForward(caller, url, user, pass string, toSend []ElkBulkUpdateItem) {
	log.L.Infof("%v Sending bulk upsert for %v items.", caller, len(toSend))

	if len(toSend) == 0 {
		return
	}

	//DEBUG
	/*
		for i := range toSend {
			log.L.Debugf("%+v", toSend[i])
		}
	*/

	log.L.Debugf("%v Building payload", caller)
	//build our payload
	payload := []byte{}
	for i := range toSend {
		var headerbytes []byte
		var err error

		if len(toSend[i].Delete.Header.Index) > 0 { //it's a delete
			headerbytes, err = json.Marshal(toSend[i].Delete)
			if err != nil {
				log.L.Errorf("%v Couldn't marshal header for elk event bulk update: %v", caller, toSend[i])
				continue
			}
			payload = append(payload, headerbytes...)
			payload = append(payload, '\n')

		} else { // do our base case
			headerbytes, err = json.Marshal(toSend[i].Index)
			if err != nil {
				log.L.Errorf("%v Couldn't marshal header for elk event bulk update: %v", caller, toSend[i])
				continue
			}

			bodybytes, err := json.Marshal(toSend[i].Doc)
			if err != nil {
				log.L.Errorf("%v Couldn't marshal header for elk event bulk update: %v", caller, toSend[i])
				continue
			}
			payload = append(payload, headerbytes...)
			payload = append(payload, '\n')
			payload = append(payload, bodybytes...)
			payload = append(payload, '\n')

		}
	}

	//once our payload is built
	log.L.Debugf("%v Payload built, sending...", caller)
	//log.L.Debugf("%s", payload)

	url = strings.Trim(url, "/")         //remove any trailing slash so we can append it again
	addr := fmt.Sprintf("%v/_bulk", url) //make the addr

	resp, er := MakeGenericELKRequest(addr, "POST", payload, user, pass)
	if er != nil {
		log.L.Errorf("%v Couldn't send bulk update. error %v", caller, er.Error())
		return
	}

	elkresp := BulkUpdateResponse{}

	err := json.Unmarshal(resp, &elkresp)
	if err != nil {
		log.L.Errorf("%v Unknown response received from ELK in response to bulk update: %s", caller, resp)
		return
	}
	if elkresp.Errors {
		log.L.Errorf("%v Errors received from ELK during bulk update %s", caller, resp)
		return
	}
	log.L.Debugf("%v Successfully sent bulk ELK updates", caller)
}
