package main

import (
	"net/http"
	"os"

	"github.com/byuoitav/central-event-system/hub/base"
	"github.com/byuoitav/central-event-system/messenger"
	"github.com/byuoitav/common"
	"github.com/byuoitav/common/log"
	"github.com/byuoitav/common/v2/events"
)

func main() {
	log.SetLevel("info")
	port := ":8333"
	router := common.NewRouter()

	// connect to the hub
	messenger, err := messenger.BuildMessenger(os.Getenv("HUB_ADDRESS"), base.Messenger, 5000)
	if err != nil {
		log.L.Fatalf("failed to build messenger: %s", err)
	}

	// get events from the hub
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

	router.StartServer(&server)
}

func processEvent(event events.Event) {
	forwarder.EventStream <- event
}
