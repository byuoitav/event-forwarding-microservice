package couch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log/slog"
	"net/http"
	"os"
	"strings"
)

// ConfigFile represents a generic config file for shipwright
type ConfigFile struct {
	ID   string          `json:"_id"`
	Path string          `json:"path"`
	File json.RawMessage `json:"contents"`
}

type CouchError struct {
	Error  string `json:"error"`
	Reason string `json:"reason"`
}

type configFileQueryResponse struct {
	Docs []struct {
		Rev string `json:"_rev,omitempty"`
		*ConfigFile
	} `json:"docs"`
	Bookmark string `json:"bookmark"`
	Warning  string `json:"warning"`
}

// UpdateConfigFiles updates the config files on disk from couchdb, using database db
func UpdateConfigFiles(ctx context.Context, db string) error {
	errMsg := "unable to update config files"

	if len(db) == 0 {
		return fmt.Errorf("%s: must pass a valid db name", errMsg)
	}

	addr := os.Getenv("DB_ADDRESS")
	if len(addr) == 0 {
		return fmt.Errorf("%s: DB_ADDRESS is not set", errMsg)
	}

	url := fmt.Sprintf("%s/%s/_find", strings.Trim(addr, "/"), db)
	slog.Info("Updating config files", "url", url)

	query := []byte(`{
	"selector": {
		"_id": {
			"$regex": ""
		}
	}
	}`)

	// build request
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(query))
	if err != nil {
		return fmt.Errorf("%s: %w", errMsg, err)
	}

	req = req.WithContext(ctx)
	req.Header.Add("content-type", "application/json")

	// add auth
	uname := os.Getenv("DB_USERNAME")
	pass := os.Getenv("DB_PASSWORD")
	if len(uname) > 0 && len(pass) > 0 {
		req.SetBasicAuth(uname, pass)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %w", errMsg, err)
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("%s: %w", errMsg, err)
	}

	if resp.StatusCode/100 != 2 {
		ce := CouchError{}
		err = json.Unmarshal(b, &ce)
		if err != nil {
			return fmt.Errorf("%s: received a %d response from %s. body: %s", errMsg, resp.StatusCode, url, b)
		}

		return fmt.Errorf("%s: %+v", errMsg, ce)
	}

	// read the docs from the response
	docs := configFileQueryResponse{}
	err = json.Unmarshal(b, &docs)
	if err != nil {
		return fmt.Errorf("%s: %w", errMsg, err)
	}

	configs := []ConfigFile{}
	for i := range docs.Docs {
		if docs.Docs[i].ConfigFile != nil {
			configs = append(configs, *docs.Docs[i].ConfigFile)
		}
	}

	return WriteFilesToDisk(configs)
}

// WriteFilesToDisk writes each of the config files to disk
func WriteFilesToDisk(configs []ConfigFile) error {
	for _, config := range configs {
		path := strings.TrimRight(config.Path, "/")
		path = path + "/" + config.ID

		if len(path) == 0 {
			continue
		}

		slog.Info("Writing new config file", "path", path)

		// create dirs in case they don't exist
		err := os.MkdirAll(config.Path, 0775)
		if err != nil {
			return fmt.Errorf("unable to write %s to disk: %w", config.Path, err)
		}

		f, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("unable to write %s to disk: %w", config.Path, err)
		}

		_, err = f.Write(config.File)
		if err != nil {
			return fmt.Errorf("unable to write %s to disk: %w", config.Path, err)
		}
	}

	return nil
}
