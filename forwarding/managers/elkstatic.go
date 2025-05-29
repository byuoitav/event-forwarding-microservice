package managers

import (
	"errors"
	"log/slog"
	"time"

	"github.com/byuoitav/event-forwarding-microservice/elk"
	sd "github.com/byuoitav/event-forwarding-microservice/state/statedefinition"
)

// ElkStaticDeviceForwarder is for a device
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

// ElkStaticForwarder is the general stuff
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
		return errors.New("Invalid type to send via an Elk device Forwarder, must be a static device as defined in state/statedefinition")
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
		return errors.New("Invalid type to send via an Elk room Forwarder, must be a static room as defined in state/statedefinition")
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
	slog.Info("Starting device forwarder", "index", e.index())
	ticker := time.NewTicker(e.interval)

	for {
		select {
		case <-ticker.C:
			//send it off
			slog.Debug("Sending bulk ELK update", "index", e.index())

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
	slog.Info("Starting room forwarder", "index", e.index())
	ticker := time.NewTicker(e.interval)

	for {
		select {
		case <-ticker.C:
			//send it off
			slog.Debug("Sending bulk ELK update", "index", e.index())

			go prepAndForward(e.index(), e.url, e.buffer)
			e.buffer = make(map[string]elk.ElkBulkUpdateItem)

		case event := <-e.incomingChannel:
			e.bufferevent(event)
		case id := <-e.deleteChannel:
			e.deleteRecord(id)
		}
	}
}

func (e *ElkStaticDeviceForwarder) bufferevent(event sd.StaticDevice) {
	if len(event.DeviceID) < 1 {
		return
	}

	//check to see if we already have one for this device
	v, ok := e.buffer[event.DeviceID]
	if !ok {
		Header := elk.HeaderIndex{
			Index: e.index(),
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
