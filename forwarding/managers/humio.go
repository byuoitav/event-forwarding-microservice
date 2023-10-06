package managers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/byuoitav/common/log"
	"github.com/byuoitav/common/nerr"
	"github.com/byuoitav/common/v2/events"
	"github.com/byuoitav/event-forwarding-microservice/humio"
)

type HumioForwarder struct {
	incomingChannel chan events.Event
	interval        time.Duration
	eventsBuffer    []events.Event
}

func GetDefaultHumioForwarder() *HumioForwarder {
	log.L.Infof("Getting default Humio forwarder")
	forwarder := &HumioForwarder{
		interval:        5 * time.Second,
		incomingChannel: make(chan events.Event, 10000),
		eventsBuffer:    []events.Event{},
	}
	go forwarder.Start()
	return forwarder
}

// Send
func (e *HumioForwarder) Send(toSend interface{}) error {
	log.L.Debugf("Sending event to Humio")
	var event events.Event

	switch e := toSend.(type) {
	case *events.Event:
		event = *e
	case events.Event:
		event = e
	default:
		return nerr.Create("Invalid type to send via a Humio Forwarder, must be an event from the byuoitav/common/events package.", "invalid-type")
	}
	e.incomingChannel <- event
	return nil
}

// Start
func (e *HumioForwarder) Start() {
	log.L.Infof("Starting event forwarder for Humio")
	ticker := time.NewTicker(e.interval)

	for {
		select {
		case <-ticker.C:
			log.L.Debugf("Here We ARE")

			log.L.Debugf("Sending events to Humio")
			e.sendBuffer()
			e.flushBuffer()

		case event := <-e.incomingChannel:
			e.bufferevent(event)
		}
	}
}

// Buffer Events
func (e *HumioForwarder) bufferevent(event events.Event) {
	log.L.Infof("Buffering event from %s for Humio\n", event.GeneratingSystem)
	e.eventsBuffer = append(e.eventsBuffer, event)
	//insure the buffer doesn't get too big
	if len(e.eventsBuffer) > 4000 {
		log.L.Debugf("Humio Buffer surpassing 4000 events")
		e.sendBuffer()
		e.flushBuffer()
	}
}

func (e *HumioForwarder) flushBuffer() {
	log.L.Debugf("Flushing buffer for Humio")
	e.eventsBuffer = []events.Event{}
}

func (e *HumioForwarder) marshalBuffer() []byte {
	log.L.Debugf("Marshalling buffer for Humio")
	logs, err := json.Marshal(e.eventsBuffer)
	if err != nil {
		log.L.Debugf("Failed to marshal buffer for Humio: %s", err.Error())
	}
	log.L.Debugf(string(logs))
	return logs
}

func (e *HumioForwarder) sendBuffer() error {
	log.L.Debugf("Sending buffer for Humio")
	if len(e.eventsBuffer) > 0 {
		_, err := humio.MakeHumioRequest(http.MethodPost, "/api/v1/ingest/json", e.marshalBuffer())
		if err != nil {
			log.L.Debugf("Failed to send buffer for Humio: %s", err.Error())
			return err
		}
	}
	return nil
}
