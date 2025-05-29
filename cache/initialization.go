package cache

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/byuoitav/event-forwarding-microservice/cache/memorycache"
	"github.com/byuoitav/event-forwarding-microservice/cache/shared"
	"github.com/byuoitav/event-forwarding-microservice/config"
	"github.com/byuoitav/event-forwarding-microservice/elk"
	"github.com/byuoitav/event-forwarding-microservice/state/statedefinition"
)

const maxSize = 10000
const pushCron = "0 0 0 * * *"

// InitializeCaches initializes the caches with data from ELK
func InitializeCaches() {
	slog.Info("Initializing Caches")
	Caches = make(map[string]shared.Cache)

	c := config.GetConfig()
	for _, i := range c.Caches {
		slog.Info("Initializing cache", "name", i.Name)
		var devs []statedefinition.StaticDevice
		var rooms []statedefinition.StaticRoom
		var er error
		//depending on storage, data, and cache type depends on what function we call.
		switch i.StorageType {
		case config.Elk:
			//within the elk type
			devs, er = GetElkStaticDevices(i.ELKinfo.DeviceIndex, i.ELKinfo.URL)
			if er != nil {
				slog.Error("Couldn't get information for device cache", "name", i.Name, "error", er.Error())
			}

			if i.ELKinfo.RoomIndex != "" {
				rooms, er = GetElkStaticRooms(i.ELKinfo.RoomIndex, i.ELKinfo.URL)
				if er != nil {
					slog.Error("Couldn't get information for room cache", "name", i.Name, "error", er.Error())
				}
			}
		default:
			slog.Info("No storage type")
		}
		cache, err := makeCache(devs, rooms, i)
		if err != nil {
			slog.Error("Couldn't make cache", "error", err.Error())
			continue
		}

		Caches[i.Name] = cache
		slog.Info("Cache initialized", "name", i.Name, "type", i.CacheType, "devices", len(devs), "rooms", len(rooms))
	}

	slog.Info("Caches Initialized.")
}

// GetElkStaticDevices queries the provided index in ELK and unmarshals the records into a list of static devices
func GetElkStaticDevices(index, url string) ([]statedefinition.StaticDevice, error) {
	slog.Debug("Getting device information from", "index", index)
	query := elk.GenericQuery{
		Size: maxSize,
	}

	b, er := json.Marshal(query)
	if er != nil {
		return []statedefinition.StaticDevice{}, fmt.Errorf("Couldn't marshal generic query %v: %w", query, er)
	}

	resp, err := elk.MakeGenericELKRequest(fmt.Sprintf("%v/%v/_search", url, index), "GET", b, "", "")
	if err != nil {
		return []statedefinition.StaticDevice{}, fmt.Errorf("Couldn't retrieve static index %v for cache: %w", index, err)
	}

	var queryResp elk.StaticDeviceQueryResponse
	er = json.Unmarshal(resp, &queryResp)
	if er != nil {
		return []statedefinition.StaticDevice{}, fmt.Errorf("Couldn't unmarshal response from static index %v: %w", index, er)
	}

	var toReturn []statedefinition.StaticDevice
	for i := range queryResp.Hits.Wrappers {
		toReturn = append(toReturn, queryResp.Hits.Wrappers[i].Device)
	}

	return toReturn, nil
}

// GetElkStaticRooms retrieves the list of static rooms from the privided elk index - assumes the ELK_DIRECT_ADDRESS env variable.
func GetElkStaticRooms(index, url string) ([]statedefinition.StaticRoom, error) {
	query := elk.GenericQuery{
		Size: maxSize,
	}

	b, er := json.Marshal(query)
	if er != nil {
		return []statedefinition.StaticRoom{}, fmt.Errorf("Couldn't marshal generic query %v: %w", query, er)
	}

	resp, err := elk.MakeGenericELKRequest(fmt.Sprintf("%v/%v/_search", url, index), "GET", b, "", "")
	if err != nil {
		return []statedefinition.StaticRoom{}, fmt.Errorf("Couldn't retrieve static index %v for cache: %w", index, err)
	}
	slog.Info("Getting the info for", "index", index)

	var queryResp elk.StaticRoomQueryResponse
	er = json.Unmarshal(resp, &queryResp)
	if er != nil {
		return []statedefinition.StaticRoom{}, fmt.Errorf("Couldn't unmarshal response from static index %v: %w", index, er)
	}

	var toReturn []statedefinition.StaticRoom
	for i := range queryResp.Hits.Wrappers {
		toReturn = append(toReturn, queryResp.Hits.Wrappers[i].Room)
	}

	return toReturn, nil
}

func makeCache(devices []statedefinition.StaticDevice, rooms []statedefinition.StaticRoom, config config.Cache) (shared.Cache, error) {
	switch config.CacheType {
	case "memory":
		return memorycache.MakeMemoryCache(devices, rooms, pushCron, config)
	}
	return nil, fmt.Errorf("Unknown cache type %v", config.CacheType)
}
