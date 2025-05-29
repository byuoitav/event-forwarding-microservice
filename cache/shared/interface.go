package shared

import (
	"github.com/byuoitav/event-forwarding-microservice/events"
	"github.com/byuoitav/event-forwarding-microservice/state/statedefinition"
)

// Cache is our state cache - it's meant to be a representation of the static indexes
type Cache interface {
	CheckAndStoreDevice(device statedefinition.StaticDevice) (bool, statedefinition.StaticDevice, error)
	CheckAndStoreRoom(room statedefinition.StaticRoom) (bool, statedefinition.StaticRoom, error)

	GetDeviceRecord(deviceID string) (statedefinition.StaticDevice, error)
	GetRoomRecord(roomID string) (statedefinition.StaticRoom, error)
	GetAllDeviceRecords() ([]statedefinition.StaticDevice, error)
	GetAllRoomRecords() ([]statedefinition.StaticRoom, error)

	StoreDeviceEvent(toSave statedefinition.State) (bool, statedefinition.StaticDevice, error)
	StoreAndForwardEvent(event events.Event) (bool, error)

	RemoveDevice(deviceID string) error       //Removes a specific device record
	RemoveRoom(roomID string) error           //Removes a specific room record
	NukeRoom(roomID string) ([]string, error) //Removes a room and all of it's devices

	GetCacheType() string
	GetCacheName() string
}
