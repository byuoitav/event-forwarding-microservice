package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"runtime"

	"github.com/byuoitav/central-event-system/hub/base"
	"github.com/byuoitav/central-event-system/messenger"

	//"github.com/byuoitav/common"
	//"github.com/byuoitav/common/log"
	//"github.com/byuoitav/common/v2/events"
	"github.com/byuoitav/event-forwarding-microservice/events"
	"github.com/byuoitav/event-forwarding-microservice/helpers"
	"github.com/gin-gonic/gin"
)

var (
	port   string
	logger *slog.Logger
)

func main() {

	flag.StringVar(&port, "port", "8333", "port for microservice to av-api communication")
	flag.Parse()

	port = ":" + port

	//setup logger
	var logLevel = new(slog.LevelVar)
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)

	// set log levels
	logLevel.Set(slog.LevelInfo)
	if runtime.GOOS == "windows" {
		logLevel.Set(slog.LevelDebug)
		logger.Info("running from Windows, logging set to debug")
	}

	logger.Info("Event Forwardering Service -- Started --")

	// Build Gin Server
	router := gin.Default()

	// Start Gin server with endpoints
	router.GET("/healthz", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, "OK")
	})

	router.GET("/status", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, "Service is Active")
	})

	router.GET("/log-level", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, logLevel.String())
	})

	// Allow setting the log level
	router.PUT("/log-level/:level", func(ctx *gin.Context) {
		lvl := ctx.Param("level")

		// Get the log level and convert it to slog
		level, err := stringToLogLevel(lvl)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, "invalid log level")
			return
		}

		// Set the Log Level
		logLevel.Set(level)
		newlogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
		slog.SetDefault(newlogger)
		ctx.String(http.StatusOK, lvl)
	})

	// Starting the Forwarder Manager which will send to the specified systems
	go helpers.GetForwardManager().Start(context.TODO())

	// Connect to the Event Hub
	messenger, err := messenger.BuildMessenger(os.Getenv("HUB_ADDRESS"), base.Messenger, 5000)
	if err != nil {
		logger.Error("failed to build messenger: %s", err)
		os.Exit(1) // Exiting due to messenger not building properly.
	}

	// Start the pump to get events from the hub
	go func() {
		messenger.SubscribeToRooms("*")

		for {
			processEvent(messenger.ReceiveEvent())
		}
	}()

	server := http.Server{
		Addr:           port,
		MaxHeaderBytes: 1024 * 10,
	}

	router.Run(server.Addr)
}

func processEvent(event events.Event) {
	helpers.GetForwardManager().EventStream <- event
}
