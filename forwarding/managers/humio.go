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

// HumioForwarder is a forwarder that sends events to Humio
type HumioForwarder struct {
	incomingChannel chan events.Event
	interval        time.Duration
	bufferSize      int
	eventsBuffer    []HumioEvent
	ingestToken     string
}

// returns a forwarder that sends events to Humio
func GetDefaultHumioForwarder(interval time.Duration, bufferSize int, ingestToken string) *HumioForwarder {
	log.L.Infof("Getting default Humio forwarder with bufferSize %d and interval %s", bufferSize, interval.String())
	forwarder := &HumioForwarder{
		interval:        interval,
		bufferSize:      bufferSize,
		incomingChannel: make(chan events.Event, 10000),
		eventsBuffer:    []HumioEvent{},
		ingestToken:     ingestToken,
	}
	go forwarder.Start()
	return forwarder
}

// Sends events to the Humio Forwarder's incoming channel
func (e *HumioForwarder) Send(toSend interface{}) error {
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

// Starts the event forwarder specific to Humio
func (e *HumioForwarder) Start() {
	log.L.Infof("Starting event forwarder for Humio")
	ticker := time.NewTicker(e.interval)

	for {
		select {
		case <-ticker.C:
			if (len(e.eventsBuffer)) != 0 {
				log.L.Debugf("Sending events to Humio")
				e.sendBuffer()
				e.flushBuffer()
			}

		case event := <-e.incomingChannel:
			log.L.Debugf("Receiving Event for Humio")
			e.bufferevent(event)
		}
	}
}

// Adds events to the buffer
func (e *HumioForwarder) bufferevent(event events.Event) {
	log.L.Debugf("Buffering event from %s for Humio\n", event.GeneratingSystem)
	//convert events.Event to HumioEvent
	alias := convertEvent(event)
	e.eventsBuffer = append(e.eventsBuffer, alias)
	//insure the buffer doesn't get too big
	if len(e.eventsBuffer) > e.bufferSize-1 {
		log.L.Debugf("Humio Buffer surpassing %d events", e.bufferSize)
		e.sendBuffer()
		e.flushBuffer()
	}
}

// clears the buffer of all events
func (e *HumioForwarder) flushBuffer() {
	log.L.Debugf("Flushing buffer of %d events for Humio", len(e.eventsBuffer))
	e.eventsBuffer = []HumioEvent{}
}

// marshals the buffer to json
func (e *HumioForwarder) marshalBuffer() []byte {
	log.L.Debugf("Marshaling buffer for Humio")
	logs, err := json.Marshal(e.eventsBuffer)
	if err != nil {
		log.L.Debugf("Failed to marshal buffer for Humio: %s", err.Error())
	}
	log.L.Debugf(string(logs))
	return logs
}

// send the buffer to humio
func (e *HumioForwarder) sendBuffer() error {
	log.L.Infof("Sending buffer for Humio of %d events", len(e.eventsBuffer))
	if len(e.eventsBuffer) > 0 {
		_, err := humio.MakeHumioRequest(http.MethodPost, "/api/v1/ingest/json", e.marshalBuffer(), e.ingestToken)
		if err != nil {
			log.L.Debugf("Failed to send buffer for Humio: %s", err.Error())
			return err
		}
	}
	return nil
}

// convert events.Event to HumioEvent
func convertEvent(event events.Event) HumioEvent {
	//convert timestamp to unix time
	unixTime := event.Timestamp.Unix() * 1000
	//create and return humio event using new time
	alias := HumioEvent{
		GeneratingSystem: event.GeneratingSystem,
		EventTags:        event.EventTags,
		TargetDevice: BasicDeviceInfo{
			BasicRoomInfo: BasicRoomInfo{
				BuildingID: event.TargetDevice.BuildingID,
				RoomID:     event.TargetDevice.RoomID,
			},
			DeviceID: event.TargetDevice.DeviceID,
		},
		AffectedRoom: BasicRoomInfo{
			BuildingID: event.AffectedRoom.BuildingID,
			RoomID:     event.AffectedRoom.RoomID,
		},
		Key:       event.Key,
		Value:     event.Value,
		User:      event.User,
		Data:      event.Data,
		Timestamp: unixTime,
		Timezone:  event.Timestamp.Format("MDT"),
	}
	return alias
}

// Edits the field names to match the Humio schema
type HumioEvent struct {
	// GeneratingSystem is the system actually generating the event. i.e. For an API call against a raspberry pi this would be the hostname of the raspberry pi running the AV-API. If the call is against AWS, this would be 'AWS'
	GeneratingSystem string `json:"generating-system"`

	// EventTags is a collection of strings to give more information about what kind of event this is, used in routing and processing events. See the EventTags const delcaration for some common tags.
	EventTags []string `json:"event-tags"`

	// TargetDevice is the device being affected by the event. e.g. a power on event, this would be the device powering on
	TargetDevice BasicDeviceInfo `json:"target-device"`

	// AffectedRoom is the room being affected by the event. e.g. in events arising from an API call this is the room called in the API
	AffectedRoom BasicRoomInfo `json:"affected-room"`

	// Key of the event
	Key string `json:"key"`

	// Value of the event
	Value string `json:"value"`

	// User is the user associated with generating the event
	User string `json:"user"`

	// Data is an optional field to dump data that you wont necessarily want to aggregate on, but you may want to search on
	Data interface{} `json:"data,omitempty"`

	// Timestamp is the time the event took place
	Timestamp int64 `json:"@timestamp"`

	// Timezone
	Timezone string `json:"@timezone"`
}

// BasicRoomInfo contains device information that is easy to aggregate on.
type BasicRoomInfo struct {
	BuildingID string `json:"buildingID,omitempty"`
	RoomID     string `json:"roomID,omitempty"`
}

// BasicDeviceInfo contains device information that is easy to aggregate on.
type BasicDeviceInfo struct {
	BasicRoomInfo
	DeviceID string `json:"deviceID,omitempty"`
}
