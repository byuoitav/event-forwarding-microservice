package managers

import (
	"fmt"
	"log/slog"
	"time"

	//sd "github.com/byuoitav/common/state/statedefinition"
	"github.com/byuoitav/event-forwarding-microservice/elk"
	customerror "github.com/byuoitav/event-forwarding-microservice/error"
	sd "github.com/byuoitav/event-forwarding-microservice/statedefinition"
)

// ElkStaticDeviceForwarder is for a device
// USED - GetDefaultElkStaticDeviceForwarder
type ElkStaticDeviceForwarder struct {
	ElkStaticForwarder
	update          bool
	incomingChannel chan sd.StaticDevice
	deleteChannel   chan string
	buffer          map[string]elk.ElkBulkUpdateItem
}

// ElkStaticRoomForwarder is for rooms
type ElkStaticRoomForwarder struct {
	ElkStaticForwarder
	update          bool
	incomingChannel chan sd.StaticRoom
	deleteChannel   chan string
	buffer          map[string]elk.ElkBulkUpdateItem
}

// ElkStaticForwarder is the common stuff
// USED - GetDefaultElkStaticDeviceForwarder
type ElkStaticForwarder struct {
	interval time.Duration //how often to send an update
	url      string
	index    func() string //function to get the indexA
}

// GetDefaultElkStaticDeviceForwarder returns a regular static device forwarder with a buffer size of 10000
func GetDefaultElkStaticDeviceForwarder(URL string, index func() string, interval time.Duration, update bool) *ElkStaticDeviceForwarder {
	toReturn := &ElkStaticDeviceForwarder{
		ElkStaticForwarder: ElkStaticForwarder{
			interval: interval,
			url:      URL,
			index:    index,
		},
		update:          update,
		incomingChannel: make(chan sd.StaticDevice, 10000),
		buffer:          make(map[string]elk.ElkBulkUpdateItem),
	}

	go toReturn.start()

	return toReturn
}

// Send takes a device and adds it to the buffer
func (e *ElkStaticDeviceForwarder) Send(toSend interface{}) error {

	var event sd.StaticDevice

	switch ev := toSend.(type) {
	case *sd.StaticDevice:
		event = *ev
	case sd.StaticDevice:
		event = ev
	default:
		typeError := &customerror.StandardError{
			Message: fmt.Sprintf("Invalid type to send via an Elk device Forwarder, must be a static device as defined in byuoitav/state-parser/state/statedefinition"),
		}
		return typeError
	}

	e.incomingChannel <- event

	return nil
}

// Send takes a room and adds it to the buffer
func (e *ElkStaticRoomForwarder) Send(toSend interface{}) error {

	var event sd.StaticRoom

	switch e := toSend.(type) {
	case *sd.StaticRoom:
		event = *e
	case sd.StaticRoom:
		event = e
	default:
		typeError := &customerror.StandardError{
			Message: fmt.Sprintf("Invalid type to send via an Elk device Forwarder, must be a static device as defined in byuoitav/state-parser/state/statedefinition"),
		}
		return typeError
	}

	e.incomingChannel <- event

	return nil
}

// Delete .
func (e *ElkStaticRoomForwarder) Delete(id string) error {
	e.deleteChannel <- id
	return nil
}

// Delete .
func (e *ElkStaticDeviceForwarder) Delete(id string) error {
	e.deleteChannel <- id
	return nil
}

// GetDefaultElkStaticRoomForwarder returns a regular static room forwarder with a buffer size of 10000
// USED in Forwarding/manager.go - GetDefaultElkStaticRoomForwarder
func GetDefaultElkStaticRoomForwarder(URL string, index func() string, interval time.Duration, update bool) *ElkStaticRoomForwarder {
	toReturn := &ElkStaticRoomForwarder{
		ElkStaticForwarder: ElkStaticForwarder{
			interval: interval,
			url:      URL,
			index:    index,
		},
		incomingChannel: make(chan sd.StaticRoom, 10000),
		buffer:          make(map[string]elk.ElkBulkUpdateItem),
		update:          update,
	}

	go toReturn.start()

	return toReturn
}

func (e *ElkStaticDeviceForwarder) start() {
	infoMsg := fmt.Sprintf("Starting device forwarder for %v", e.index())
	slog.Info(infoMsg)
	ticker := time.NewTicker(e.interval)

	for {
		select {
		case <-ticker.C:
			//send it off
			debugMsg := fmt.Sprintf("Sending bulk ELK update for %v", e.index())
			slog.Debug(debugMsg)

			go prepAndForward(e.index(), e.url, e.buffer)
			e.buffer = make(map[string]elk.ElkBulkUpdateItem)

		case event := <-e.incomingChannel:
			e.bufferevent(event)
		case id := <-e.deleteChannel:
			e.deleteRecord(id)
		}
	}
}

func (e *ElkStaticRoomForwarder) start() {
	infoMsg := fmt.Sprintf("Starting room forwarder for %v", e.index())
	slog.Info(infoMsg)
	ticker := time.NewTicker(e.interval)

	for {
		select {
		case <-ticker.C:
			//send it off
			debugMsg := fmt.Sprintf("Sending bulk ELK update for %v", e.index())
			slog.Debug(debugMsg)

			go prepAndForward(e.index(), e.url, e.buffer)
			e.buffer = make(map[string]elk.ElkBulkUpdateItem)

		case event := <-e.incomingChannel:
			e.bufferevent(event)
		case id := <-e.deleteChannel:
			e.deleteRecord(id)
		}
	}
}

// USED in start() -> ElkStaticDeviceForwarder
func (e *ElkStaticDeviceForwarder) bufferevent(event sd.StaticDevice) {
	if len(event.DeviceID) < 1 {
		return
	}

	//check to see if we already have one for this device
	v, ok := e.buffer[event.DeviceID]
	if !ok {
		Header := elk.HeaderIndex{
			Index: e.index(),
			Type:  "av-device",
		}
		if e.update {
			Header.ID = event.DeviceID
		}
		e.buffer[event.DeviceID] = elk.ElkBulkUpdateItem{
			Index: elk.ElkUpdateHeader{Header: Header},
			Doc:   event,
		}
	} else {
		//we replace
		v.Doc = event
		e.buffer[event.DeviceID] = v
	}

}

func (e *ElkStaticDeviceForwarder) deleteRecord(id string) {
	if len(id) < 1 {
		return
	}

	Header := elk.HeaderIndex{
		Index: e.index(),
		Type:  "av-device",
		ID:    id,
	}
	e.buffer[id] = elk.ElkBulkUpdateItem{
		Delete: elk.ElkDeleteHeader{Header: Header},
	}
}

func (e *ElkStaticRoomForwarder) deleteRecord(id string) {
	if len(id) < 1 {
		return
	}

	Header := elk.HeaderIndex{
		Index: e.index(),
		Type:  "av-room",
		ID:    id,
	}
	e.buffer[id] = elk.ElkBulkUpdateItem{
		Delete: elk.ElkDeleteHeader{Header: Header},
	}
}

func (e *ElkStaticRoomForwarder) bufferevent(event sd.StaticRoom) {

	if len(event.RoomID) < 1 {
		return
	}

	v, ok := e.buffer[event.RoomID]
	if !ok {
		Header := elk.HeaderIndex{
			Index: e.index(),
			Type:  "av-room",
		}
		if e.update {
			Header.ID = event.RoomID
		}
		e.buffer[event.RoomID] = elk.ElkBulkUpdateItem{
			Index: elk.ElkUpdateHeader{Header: Header},
			Doc:   event,
		}
	} else {
		v.Doc = event
		e.buffer[event.RoomID] = v
	}
}

func prepAndForward(caller, url string, vals map[string]elk.ElkBulkUpdateItem) {
	var toUpdate []elk.ElkBulkUpdateItem
	for _, v := range vals {
		toUpdate = append(toUpdate, v)
	}

	elk.BulkForward(caller, url, "", "", toUpdate)
}
