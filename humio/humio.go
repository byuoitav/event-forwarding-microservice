package humio

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/byuoitav/common/log"
	"github.com/byuoitav/common/nerr"
)

var (
	APIAddr   = os.Getenv("HUMIO_DIRECT_ADDRESS")
	authToken = os.Getenv("HUMIO_INGEST_TOKEN")
)

func MakeGenericHumioRequest(addr, method string, body interface{}, username, password string) ([]byte, *nerr.E) {
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
			return []byte{}, nerr.Translate(err)
		}
	}

	//create the request
	req, err := http.NewRequest(method, addr, bytes.NewReader(reqBody))
	if err != nil {
		return []byte{}, nerr.Translate(err)
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
		return []byte{}, nerr.Translate(err)
	}
	defer resp.Body.Close()

	//read the resp
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, nerr.Translate(err)
	}

	if resp.StatusCode/100 != 2 {
		msg := fmt.Sprintf("non 200 reponse code received. code: %v, body: %s", resp.StatusCode, respBody)
		return respBody, nerr.Create(msg, http.StatusText(resp.StatusCode))
	}

	return respBody, nil
}

func MakeHumioRequest(method, endpoint string, body interface{}) ([]byte, *nerr.E) {
	if len(APIAddr) == 0 {
		log.L.Fatalf("HUMIO_DIRECT_ADDRESS is not set.")
	}

	if len(authToken) == 0 {
		log.L.Fatalf("HUMIO_INGEST_TOKEN is not set.")
	}

	//format whole address
	addr := fmt.Sprintf("%s%s", APIAddr, endpoint)
	return MakeGenericHumioRequest(addr, method, body, "", "")
}
