package config

import (
	"encoding/json"
	"io/ioutil"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

var once sync.Once

var c Config

const (
	//Couch .
	Couch = "couch"
	//Elk  .
	Elk = "elk"
	//Humio
	Humio = "humio"
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
}

var config Config

// GetConfig .
func GetConfig() Config {
	once.Do(func() {
		getConfigFile()
	})
	return config
}

// Retrieves the config file from AWS S3
func getConfigFile() {
	awsAccessKey := os.Getenv("AWS_ACCESS_KEY")
	awsSecretKey := os.Getenv("AWS_SECRET_KEY")
	if len(awsAccessKey) == 0 {
		slog.Info("ERROR: AWS_ACCESS_KEY not set")
	}
	if len(awsSecretKey) == 0 {
		slog.Info("ERROR: AWS_SECRET_KEY not set")
	}
	awsRegion := "us-west-2"

	creds := credentials.NewStaticCredentials(awsAccessKey, awsSecretKey, "")
	awsConfig := &aws.Config{
		Region:      aws.String(awsRegion),
		Credentials: creds,
	}

	sess, err := session.NewSession(awsConfig)
	if err != nil {
		slog.Info("Error creating AWS session:", "error", err)
		return
	}

	// Create S3 service client
	svc := s3.New(sess)

	bucketName := os.Getenv("AWS_BUCKET_NAME") // "av-microservices-configs" in the dev environment
	if len(bucketName) == 0 {
		slog.Info("ERROR: AWS_BUCKET_NAME not set")
	}
	objectPath := "service-config.json"

	params := &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectPath),
	}

	resp, err := svc.GetObject(params)
	if err != nil {
		slog.Info("Error getting object from bucket:", "objectPath", objectPath, "bucketName", bucketName, "error", err)
		return
	}
	defer resp.Body.Close()

	// Read config file
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		slog.Info("Error reading object from bucket:", "objectPath", objectPath, "bucketName", bucketName, "error", err)
		return
	}

	// Unmarshal config file
	err = json.Unmarshal(b, &config)
	if err != nil {
		slog.Info("Error unmarshalling object from bucket:", "objectPath", objectPath, "bucketName", bucketName, "error", err)
		return
	}

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
