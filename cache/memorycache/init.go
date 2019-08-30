package memorycache

import (
	"github.com/byuoitav/common/log"
	"github.com/byuoitav/common/nerr"
	"github.com/byuoitav/common/state/statedefinition"
	"github.com/byuoitav/event-forwarding-microservice/config"
	"github.com/robfig/cron"
)

// MakeMemoryCache .
func MakeMemoryCache(devices []statedefinition.StaticDevice, rooms []statedefinition.StaticRoom, pushCron string, c config.Cache) (*Memorycache, *nerr.E) {
	toReturn := Memorycache{
		cacheType: "memory",
		pushCron:  cron.New(),
		name:      c.Name,
	}

	log.L.Infof("adding the cron push")
	//build our push cron
	er := toReturn.pushCron.AddFunc(pushCron, toReturn.PushAllDevices)
	if er != nil {
		log.L.Errorf("Couldn't add the push all devices cron job to the cache")

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
			log.L.Errorf("DeviceID cannot be blank. Device: %+v", devices[i])
			continue
		}

		v, err := GetNewDeviceManagerWithDevice(devices[i])
		if err != nil {
			log.L.Errorf("Cannot create device manager for %v: %v", devices[i].DeviceID, err.Error())
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
			log.L.Errorf("Error initializing cache for %v: %v.", devices[i].DeviceID, val.Error.Error())
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
			log.L.Errorf("Error initializing cache for %v: %v.", rooms[i].RoomID, val.Error.Error())
		}
		toReturn.roomCache[rooms[i].RoomID] = v
	}

	return &toReturn, nil
}
