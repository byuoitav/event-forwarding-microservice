package managers

import (
	"time"

	"github.com/byuoitav/common/log"
	"github.com/byuoitav/common/nerr"
	"github.com/byuoitav/common/v2/events"
	"github.com/byuoitav/event-forwarding-microservice/elk"
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
		return nerr.Create("Invalid type to send via an Elk Event Forwarder, must be an event from the byuoitav/common/events package.", "invalid-type")
	}

	e.incomingChannel <- event

	return nil
}

// starts the manager and buffer.
func (e *ElkTimeseriesForwarder) start() {

	log.L.Infof("Starting event forwarder for %v", e.index())
	ticker := time.NewTicker(e.interval)

	for {
		select {
		case <-ticker.C:
			//send it off
			log.L.Debugf("Sending bulk ELK update for %v", e.index())

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
