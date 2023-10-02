package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
	"sync"

	"github.com/byuoitav/common/log"
)

var once sync.Once

var c Config

const (
	//Couch .
	Couch = "couch"
	//Elk  .
	Elk = "elk"
	//TODO: Humio = "humio"
)

const (
	//Room .
	Room = "room"

	//Device .
	Device = "device"
)

// Config .
type Config struct {
	Forwarders []Forwarder `json:"forwarders"`
	Caches     []Cache     `json:"caches"`
}

var config Config

// GetConfig .
func GetConfig() Config {
	once.Do(func() {
		// parse configuration
		path := os.Getenv("SERVICE_CONFIG_LOCATION")
		if len(path) < 1 {
			path = "./service-config.json"
		}
		log.L.Infof("Parsing job configuration from: %s", path)

		// read job configuration
		b, err := ioutil.ReadFile(path)
		if err != nil {
			log.L.Fatalf("failed to read job configuration: %s", err)
		}

		// unmarshal job config
		err = json.Unmarshal(b, &config)
		if err != nil {
			log.L.Fatalf("failed to parse job configuration: %s", err)
		}
	})

	return config
}

// Contains .
func Contains(a []string, b string) bool {
	for i := range a {
		if a[i] == b {
			return true
		}
	}

	return false
}

// ReplaceEnv .
func ReplaceEnv(s string) string {

	if strings.HasPrefix(s, "ENV") {
		return os.Getenv(strings.TrimSpace(strings.TrimPrefix(s, "ENV")))
	}
	return s
}
