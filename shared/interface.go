package shared

import (

	//sd "github.com/byuoitav/common/state/statedefinition"
	sd "github.com/byuoitav/event-forwarding-microservice/statedefinition"
	//"github.com/byuoitav/common/v2/events"
	"github.com/byuoitav/event-forwarding-microservice/events"
)

// Cache is our state cache - it's meant to be a representation of the static indexes
type Cache interface {
	CheckAndStoreDevice(device sd.StaticDevice) (bool, sd.StaticDevice, error)
	CheckAndStoreRoom(room sd.StaticRoom) (bool, sd.StaticRoom, error)

	GetDeviceRecord(deviceID string) (sd.StaticDevice, error)
	GetRoomRecord(roomID string) (sd.StaticRoom, error)
	GetAllDeviceRecords() ([]sd.StaticDevice, error)
	GetAllRoomRecords() ([]sd.StaticRoom, error)

	StoreDeviceEvent(toSave sd.State) (bool, sd.StaticDevice, error)
	StoreAndForwardEvent(event events.Event) (bool, error)

	RemoveDevice(deviceID string) error       //Removes a specific device record
	RemoveRoom(roomID string) error           //Removes a specific room record
	NukeRoom(roomID string) ([]string, error) //Removes a room and all of it's devices

	GetCacheType() string
	GetCacheName() string
}
