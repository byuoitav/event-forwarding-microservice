package cache

import (
	"encoding/json"
	"fmt"

	"github.com/byuoitav/common/log"
	"github.com/byuoitav/common/nerr"
	"github.com/byuoitav/common/state/statedefinition"
	"github.com/byuoitav/event-forwarding-microservice/cache/memorycache"
	"github.com/byuoitav/event-forwarding-microservice/cache/rediscache"
	"github.com/byuoitav/event-forwarding-microservice/cache/shared"
	"github.com/byuoitav/event-forwarding-microservice/config"
	"github.com/byuoitav/event-forwarding-microservice/elk"
)

const maxSize = 10000

const pushCron = "0 0 0 * * *"

// InitializeCaches initializes the caches with data from ELK
func InitializeCaches() {
	log.L.Infof("Initializing Caches")
	Caches = make(map[string]shared.Cache)

	c := config.GetConfig()
	for _, i := range c.Caches {
		log.L.Infof("Initializing cache %v", i.Name)
		var devs []statedefinition.StaticDevice
		var rooms []statedefinition.StaticRoom
		var er *nerr.E
		//depending on storage, data, and cache type depends on what function we call.
		switch i.StorageType {
		case config.Elk:
			//within the elk type
			devs, er = GetElkStaticDevices(i.ELKinfo.DeviceIndex, i.ELKinfo.URL)
			if er != nil {
				log.L.Errorf(er.Addf("Couldn't get information for device cache %v", i.Name).Error())
			}

			if i.ELKinfo.RoomIndex != "" {
				rooms, er = GetElkStaticRooms(i.ELKinfo.RoomIndex, i.ELKinfo.URL)
				if er != nil {
					log.L.Errorf(er.Addf("Couldn't get information for room cache %v", i.Name).Error())
				}
			}
		default:
			log.L.Infof("No storage type")
		}
		cache, err := makeCache(devs, rooms, i)
		if err != nil {
			log.L.Fatalf("Couldn't make cache: %v", err.Error())
		}

		Caches[i.Name] = cache
		log.L.Infof("Cache %v initialized with type %v. %v devices and %v rooms", i.Name, i.CacheType, len(devs), len(rooms))
	}

	log.L.Infof("Caches Initialized.")
}

// GetElkStaticDevices queries the provided index in ELK and unmarshals the records into a list of static devices
func GetElkStaticDevices(index, url string) ([]statedefinition.StaticDevice, *nerr.E) {
	log.L.Debugf("Getting device information from %v", index)
	query := elk.GenericQuery{
		Size: maxSize,
	}

	b, er := json.Marshal(query)
	if er != nil {
		return []statedefinition.StaticDevice{}, nerr.Translate(er).Addf("Couldn't marshal generic query %v", query)
	}

	resp, err := elk.MakeGenericELKRequest(fmt.Sprintf("%v/%v/_search", url, index), "GET", b, "", "")
	if err != nil {
		return []statedefinition.StaticDevice{}, err.Addf("Couldn't retrieve static index %v for cache", index)
	}

	var queryResp elk.StaticDeviceQueryResponse

	er = json.Unmarshal(resp, &queryResp)
	if er != nil {
		return []statedefinition.StaticDevice{}, nerr.Translate(er).Addf("Couldn't unmarshal response from static index %v.", index)
	}

	var toReturn []statedefinition.StaticDevice
	for i := range queryResp.Hits.Wrappers {
		toReturn = append(toReturn, queryResp.Hits.Wrappers[i].Device)
	}

	return toReturn, nil
}

// GetElkStaticRooms retrieves the list of static rooms from the privided elk index - assumes the ELK_DIRECT_ADDRESS env variable.
func GetElkStaticRooms(index, url string) ([]statedefinition.StaticRoom, *nerr.E) {
	query := elk.GenericQuery{
		Size: maxSize,
	}

	b, er := json.Marshal(query)
	if er != nil {
		return []statedefinition.StaticRoom{}, nerr.Translate(er).Addf("Couldn't marshal generic query %v", query)
	}

	resp, err := elk.MakeGenericELKRequest(fmt.Sprintf("%v/%v/_search", url, index), "GET", b, "", "")
	if err != nil {
		return []statedefinition.StaticRoom{}, err.Addf("Couldn't retrieve static index %v for cache", index)
	}
	log.L.Infof("Getting the info for %v", index)

	var queryResp elk.StaticRoomQueryResponse

	er = json.Unmarshal(resp, &queryResp)
	if er != nil {
		return []statedefinition.StaticRoom{}, nerr.Translate(er).Addf("Couldn't unmarshal response from static index %v.", index)
	}

	var toReturn []statedefinition.StaticRoom
	for i := range queryResp.Hits.Wrappers {
		toReturn = append(toReturn, queryResp.Hits.Wrappers[i].Room)
	}

	return toReturn, nil

}

func makeCache(devices []statedefinition.StaticDevice, rooms []statedefinition.StaticRoom, config config.Cache) (shared.Cache, *nerr.E) {
	switch config.CacheType {
	case "memory":
		return memorycache.MakeMemoryCache(devices, rooms, pushCron, config)
	case "redis":
		return rediscache.MakeRedisCache(devices, rooms, pushCron, config)
	}

	return nil, nerr.Create("Unkown cache type %v", config.CacheType)

}
