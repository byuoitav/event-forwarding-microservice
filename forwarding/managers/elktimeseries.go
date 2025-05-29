package managers

import (
	"errors"
	"time"

	"log/slog"

	"github.com/byuoitav/event-forwarding-microservice/elk"
	"github.com/byuoitav/event-forwarding-microservice/events"
)

// ElkTimeseriesForwarder NOT THREAD SAFE
type ElkTimeseriesForwarder struct {
	incomingChannel chan events.Event
	buffer          []elk.ElkBulkUpdateItem
	ElkStaticForwarder
}

// GetDefaultElkTimeSeries returns a default elk event forwarder after setting it up.
func GetDefaultElkTimeSeries(URL string, index func() string, interval time.Duration) *ElkTimeseriesForwarder {
	toReturn := &ElkTimeseriesForwarder{
		incomingChannel: make(chan events.Event, 1000),
		ElkStaticForwarder: ElkStaticForwarder{
			interval: interval,
			url:      URL,
			index:    index,
		},
	}

	//start the manager
	go toReturn.start()

	return toReturn
}

// Send .
func (e *ElkTimeseriesForwarder) Send(toSend interface{}) error {
	var event events.Event

	switch e := toSend.(type) {
	case *events.Event:
		event = *e
	case events.Event:
		event = e
	default:
		return errors.New("Invalid type to send via an Elk Event Forwarder, must be an event from the events package.")
	}

	e.incomingChannel <- event

	return nil
}

// starts the manager and buffer.
func (e *ElkTimeseriesForwarder) start() {
	slog.Info("Starting event forwarder", "index", e.index())
	ticker := time.NewTicker(e.interval)

	for {
		select {
		case <-ticker.C:
			//send it off
			slog.Debug("Sending bulk ELK update", "index", e.index())

			go elk.BulkForward(e.index(), e.url, "", "", e.buffer)
			e.buffer = []elk.ElkBulkUpdateItem{}

		case event := <-e.incomingChannel:
			e.bufferevent(event)
		}
	}
}

// NOT THREAD SAFE
func (e *ElkTimeseriesForwarder) bufferevent(event events.Event) {
	e.buffer = append(e.buffer, elk.ElkBulkUpdateItem{
		Index: elk.ElkUpdateHeader{
			Header: elk.HeaderIndex{
				Index: e.index(),
			}},
		Doc: event,
	})
}
