package managers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/byuoitav/event-forwarding-microservice/events"
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
	slog.Info("Getting default Humio forwarder", "bufferSize", bufferSize, "interval", interval.String())
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
		return errors.New("invalid type to send via a Humio Forwarder, must be an event from the byuoitav/common/events package")
	}
	e.incomingChannel <- event
	return nil
}

// Starts the event forwarder specific to Humio
func (e *HumioForwarder) Start() {
	slog.Info("Starting event forwarder for Humio")
	ticker := time.NewTicker(e.interval)

	for {
		select {
		case <-ticker.C:
			if len(e.eventsBuffer) != 0 {
				slog.Debug("Sending events to Humio")
				e.sendBuffer()
				e.flushBuffer()
			}

		case event := <-e.incomingChannel:
			slog.Debug("Receiving Event for Humio")
			e.bufferevent(event)
		}
	}
}

// Adds events to the buffer
func (e *HumioForwarder) bufferevent(event events.Event) {
	slog.Debug("Buffering event for Humio", "GeneratingSystem", event.GeneratingSystem)
	//convert events.Event to HumioEvent
	alias := convertEvent(event)
	e.eventsBuffer = append(e.eventsBuffer, alias)
	//ensure the buffer doesn't get too big
	if len(e.eventsBuffer) > e.bufferSize-1 {
		slog.Debug("Humio Buffer surpassing limit", "bufferSize", e.bufferSize)
		e.sendBuffer()
		e.flushBuffer()
	}
}

// clears the buffer of all events
func (e *HumioForwarder) flushBuffer() {
	slog.Debug("Flushing buffer for Humio", "eventCount", len(e.eventsBuffer))
	e.eventsBuffer = []HumioEvent{}
}

// marshals the buffer to json
func (e *HumioForwarder) marshalBuffer() []byte {
	slog.Debug("Marshaling buffer for Humio")
	logs, err := json.Marshal(e.eventsBuffer)
	if err != nil {
		slog.Debug("Failed to marshal buffer for Humio", "error", err.Error())
	}
	slog.Debug("Marshaled buffer", "logs", string(logs))
	return logs
}

// send the buffer to humio
func (e *HumioForwarder) sendBuffer() error {
	slog.Info("Sending buffer for Humio", "eventCount", len(e.eventsBuffer))
	if len(e.eventsBuffer) > 0 {
		_, err := humio.MakeHumioRequest(http.MethodPost, "/api/v1/ingest/json", e.marshalBuffer(), e.ingestToken)
		if err != nil {
			slog.Debug("Failed to send buffer for Humio", "error", err.Error())
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
	GeneratingSystem string          `json:"generating-system"`
	EventTags        []string        `json:"event-tags"`
	TargetDevice     BasicDeviceInfo `json:"target-device"`
	AffectedRoom     BasicRoomInfo   `json:"affected-room"`
	Key              string          `json:"key"`
	Value            string          `json:"value"`
	User             string          `json:"user"`
	Data             interface{}     `json:"data,omitempty"`
	Timestamp        int64           `json:"@timestamp"`
	Timezone         string          `json:"@timezone"`
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
