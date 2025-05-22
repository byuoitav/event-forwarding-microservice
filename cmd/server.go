package main

import (
	"context"
	"net/http"
	"os"

	"log/slog"

	"github.com/byuoitav/central-event-system/hub/base"
	"github.com/byuoitav/central-event-system/messenger"

	"github.com/byuoitav/event-forwarding-microservice/events"
	"github.com/byuoitav/event-forwarding-microservice/helpers"

	"github.com/gin-gonic/gin"
	"github.com/spf13/pflag"
)

var logger *slog.Logger

func main() {
	var port, logLev string
	pflag.StringVarP(&port, "port", "p", "8333", "port for microservice to av-api communication")
	pflag.StringVarP(&logLev, "log", "l", "Info", "Initial log level")
	pflag.Parse()

	port = ":" + port
	logLevel := new(slog.LevelVar)

	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)

	setLogLevel(logLev, logLevel)

	go helpers.GetForwardManager().Start(context.TODO())
	// connect to the hub
	messenger, err := messenger.BuildMessenger(os.Getenv("HUB_ADDRESS"), base.Messenger, 5000)
	if err != nil {
		logger.Error("failed to build messenger", "error", err)
		os.Exit(1)
	}

	// get events from the hub
	go func() {
		messenger.SubscribeToRooms("*")

		for {
			// the messenger comes from the central-event-system, which is dependent on /common/v2/events
			processEvent(events.ConvertV2ToCommon(messenger.ReceiveEvent()))
		}
	}()

	router := gin.Default()
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	router.GET("/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "good",
		})
	})

	router.GET("/logLevel/:level", func(context *gin.Context) {
		err := setLogLevel(context.Param("level"), logLevel)
		if err != nil {
			logger.Error("can not set log level", "error", err)
			context.JSON(http.StatusInternalServerError, err.Error())
			return
		}
		context.JSON(http.StatusOK, gin.H{
			"current logLevel": logLevel.Level(),
		})
	})

	router.GET("/logLevel", func(context *gin.Context) {
		context.JSON(http.StatusOK, gin.H{
			"current logLevel": logLevel.Level(),
		})
	})

	router.Run(port)
}

func processEvent(event events.Event) {
	helpers.GetForwardManager().EventStream <- event
}

func setLogLevel(level string, logLevel *slog.LevelVar) error {
	lvl, err := stringToLogLevel(level)
	if err != nil {
		return err
	}
	logLevel.Set(lvl)
	return nil
}
