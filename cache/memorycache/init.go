package memorycache

import (
	"log/slog"

	"github.com/byuoitav/event-forwarding-microservice/config"
	"github.com/byuoitav/event-forwarding-microservice/state/statedefinition"
	"github.com/robfig/cron"
)

// MakeMemoryCache .
func MakeMemoryCache(devices []statedefinition.StaticDevice, rooms []statedefinition.StaticRoom, pushCron string, c config.Cache) (*Memorycache, error) {
	toReturn := Memorycache{
		cacheType: "memory",
		pushCron:  cron.New(),
		name:      c.Name,
	}

	slog.Info("adding the cron push")
	//build our push cron
	er := toReturn.pushCron.AddFunc(pushCron, toReturn.PushAllDevices)
	if er != nil {
		slog.Error("Couldn't add the push all devices cron job to the cache")
	}

	//starting the cron job
	toReturn.pushCron.Start()

	//go through and create our maps
	toReturn.deviceCache = make(map[string]DeviceItemManager)

	for i := range devices {
		//check for duplicate
		v, ok := toReturn.deviceCache[devices[i].DeviceID]
		if ok {
			continue
		}

		if len(devices[i].DeviceID) < 1 {
			slog.Error("DeviceID cannot be blank.", "device", devices[i])
			continue
		}

		v, err := GetNewDeviceManagerWithDevice(devices[i])
		if err != nil {
			slog.Error("Cannot create device manager", "deviceID", devices[i].DeviceID, "error", err.Error())
			continue
		}

		respChan := make(chan DeviceTransactionResponse, 1)
		v.WriteRequests <- DeviceTransactionRequest{
			MergeDeviceEdit: true,
			MergeDevice:     devices[i],
			ResponseChan:    respChan,
		}
		val := <-respChan

		if val.Error != nil {
			slog.Error("Error initializing cache", "deviceID", devices[i].DeviceID, "error", val.Error.Error())
		}
		toReturn.deviceCache[devices[i].DeviceID] = v
	}

	toReturn.roomCache = make(map[string]RoomItemManager)
	for i := range rooms {
		//check for duplicate
		v, ok := toReturn.roomCache[devices[i].DeviceID]
		if ok {
			continue
		}
		v = GetNewRoomManager(rooms[i].RoomID)

		respChan := make(chan RoomTransactionResponse, 1)
		v.WriteRequests <- RoomTransactionRequest{
			MergeRoom:    rooms[i],
			ResponseChan: respChan,
		}
		val := <-respChan

		if val.Error != nil {
			slog.Error("Error initializing cache", "roomID", rooms[i].RoomID, "error", val.Error.Error())
		}
		toReturn.roomCache[rooms[i].RoomID] = v
	}

	return &toReturn, nil
}
