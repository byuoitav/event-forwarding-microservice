package rediscache

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/byuoitav/common/log"
	"github.com/byuoitav/common/nerr"
	sd "github.com/byuoitav/common/state/statedefinition"
	"github.com/byuoitav/common/v2/events"
	"github.com/byuoitav/event-forwarding-microservice/cache/shared"
	"github.com/byuoitav/event-forwarding-microservice/config"
	"github.com/go-redis/redis"
)

// RedisCache .
type RedisCache struct {
	configuration config.Cache

	devclient  *redis.Client
	roomclient *redis.Client

	devLock *sync.RWMutex
	devmu   map[string]*sync.Mutex

	roomLock *sync.RWMutex
	roommu   map[string]*sync.Mutex
}

func init() {
	gob.Register(sd.StaticDevice{})
	gob.Register(time.Now())
	gob.Register(map[string]time.Time{})
	gob.Register(sd.StaticRoom{})
}

// MakeRedisCache .
func MakeRedisCache(pushCron string, configuration config.Cache) (shared.Cache, *nerr.E) {

	//substitute the password if needed
	pass := config.ReplaceEnv(configuration.RedisInfo.Password)
	addr := config.ReplaceEnv(configuration.RedisInfo.URL)
	if addr == "" {
		addr = "localhost:6379"
	}
	if configuration.RedisInfo.RoomDatabase == 0 {
		configuration.RedisInfo.RoomDatabase = 1
	}

	toReturn := &RedisCache{
		configuration: configuration,
		devmu:         map[string]*sync.Mutex{},
		roommu:        map[string]*sync.Mutex{},
		devLock:       &sync.RWMutex{},
		roomLock:      &sync.RWMutex{},
	}

	toReturn.devclient = redis.NewClient(&redis.Options{
		PoolSize:    500,
		PoolTimeout: 10 * time.Second,
		Addr:        addr,
		Password:    pass,
		DB:          configuration.RedisInfo.DevDatabase,
	})

	_, err := toReturn.devclient.Ping().Result()
	if err != nil {
		return toReturn, nerr.Translate(err).Addf("Couldn't communicate with redis server at %v", addr)
	}

	toReturn.roomclient = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: pass,
		DB:       configuration.RedisInfo.RoomDatabase,
	})

	_, err = toReturn.roomclient.Ping().Result()
	if err != nil {
		return toReturn, nerr.Translate(err).Addf("Couldn't communicate with redis server at %v", addr)
	}

	return toReturn, nil
}

// CheckAndStoreDevice .
func (rc *RedisCache) CheckAndStoreDevice(device sd.StaticDevice) (bool, sd.StaticDevice, *nerr.E) {
	v := rc.getDeviceMu(device.DeviceID)
	log.L.Debugf("Waiting for lock on device %v", device.DeviceID)
	v.Lock()
	defer v.Unlock()
	log.L.Debugf("Working on device %v", device.DeviceID)

	dev, err := rc.getDevice(device.DeviceID)
	if err != nil {
		return false, dev, err.Addf("Couldn't check and store device")
	}
	log.L.Debugf("Device: %+v", device)

	_, merged, changes, err := sd.CompareDevices(device, dev)
	if err != nil {
		return false, dev, err.Addf("Couldn't check and store device")
	}
	log.L.Debugf("Merged: %+v", merged)

	err = rc.putDevice(merged)
	if err != nil {
		return false, device, err.Addf("Couldn't check and store device")
	}

	err = shared.ForwardDevice(merged, changes, rc)

	return changes, merged, err
}

// CheckAndStoreRoom .
func (rc *RedisCache) CheckAndStoreRoom(room sd.StaticRoom) (bool, sd.StaticRoom, *nerr.E) {
	v := rc.getRoomMu(room.RoomID)
	v.Lock()
	defer v.Unlock()

	log.L.Debugf("checking and storing room %v", room)
	rm, err := rc.getRoom(room.RoomID)
	if err != nil {
		return false, rm, err.Addf("Couldn't check and store room")
	}
	log.L.Debugf("got room: %v", rm)

	_, merged, changes, err := sd.CompareRooms(rm, room)
	if err != nil {
		return false, rm, err.Addf("Couldn't check and store room")
	}

	log.L.Debugf("changes: %v, Room: %+v", changes, merged)

	if changes {
		err := rc.putRoom(merged)
		if err != nil {
			return false, room, err.Addf("Couldn't check and store room")
		}
	}

	err = shared.ForwardRoom(merged, changes, rc)

	return changes, merged, err

}

// GetDeviceRecord .
func (rc *RedisCache) GetDeviceRecord(deviceID string) (sd.StaticDevice, *nerr.E) {
	//get and lock the device
	v := rc.getDeviceMu(deviceID)
	v.Lock()
	defer v.Unlock()

	//get the device
	return rc.getDevice(deviceID)
}

// GetRoomRecord .
func (rc *RedisCache) GetRoomRecord(roomID string) (sd.StaticRoom, *nerr.E) {
	//get and lock the room
	v := rc.getRoomMu(roomID)
	v.Lock()
	defer v.Unlock()

	//get the device
	return rc.getRoom(roomID)
}

// GetAllDeviceRecords .
func (rc *RedisCache) GetAllDeviceRecords() ([]sd.StaticDevice, *nerr.E) {

	keys, er := rc.getAllDeviceKeys()
	if er != nil {
		return []sd.StaticDevice{}, er.Addf("Couldn't get all device records")
	}

	result, err := rc.devclient.MGet(keys...).Result()
	if err != nil {
		return []sd.StaticDevice{}, nerr.Translate(err).Addf("Couldn't get all device records")
	}

	var toReturn []sd.StaticDevice

	for i := range result {
		var tmp sd.StaticDevice

		err := json.Unmarshal([]byte(result[i].(string)), &tmp)
		if err != nil {
			return []sd.StaticDevice{}, nerr.Translate(err).Addf("Couldn't get all device records")
		}
		toReturn = append(toReturn, tmp)
	}

	return toReturn, nil
}

// GetAllRoomRecords .
func (rc *RedisCache) GetAllRoomRecords() ([]sd.StaticRoom, *nerr.E) {
	keys, er := rc.getAllRoomKeys()
	if er != nil {
		return []sd.StaticRoom{}, er.Addf("Couldn't get all device records")
	}

	result, err := rc.roomclient.MGet(keys...).Result()
	if err != nil {
		return []sd.StaticRoom{}, nerr.Translate(err).Addf("Couldn't get all device records")
	}

	var toReturn []sd.StaticRoom

	for i := range result {
		var tmp sd.StaticRoom

		err := json.Unmarshal([]byte(result[i].(string)), &tmp)
		if err != nil {
			return []sd.StaticRoom{}, nerr.Translate(err).Addf("Couldn't get all device records")
		}
		toReturn = append(toReturn, tmp)
	}

	return toReturn, nil
}

// RemoveDevice .
func (rc *RedisCache) RemoveDevice(id string) *nerr.E {
	v := rc.getDeviceMu(id)
	v.Lock()
	defer v.Unlock()

	return nerr.Translate(rc.devclient.Del(id).Err()).Addf("Couldn't remove device %v", id)
}

// RemoveRoom .
func (rc *RedisCache) RemoveRoom(id string) *nerr.E {

	v := rc.getRoomMu(id)
	v.Lock()
	defer v.Unlock()

	return nerr.Translate(rc.roomclient.Del(id).Err()).Addf("Couldn't remove room %v", id)
}

// NukeRoom .
func (rc *RedisCache) NukeRoom(id string) ([]string, *nerr.E) {
	keys, err := rc.devclient.Keys(fmt.Sprintf("%s*", id)).Result()
	if err != nil {
		return []string{}, nerr.Translate(err).Addf("Couldn't nuke room")
	}

	er := rc.RemoveRoom(id)
	if er != nil {
		return []string{}, er.Addf("Couldn't nuke room")
	}

	for i := range keys {
		er = rc.RemoveDevice(keys[i])
		if er != nil {
			return []string{}, er.Addf("Couldn't nuke room")
		}
	}

	return keys, nil
}

// StoreDeviceEvent .
func (rc *RedisCache) StoreDeviceEvent(toSave sd.State) (bool, sd.StaticDevice, *nerr.E) {
	if len(toSave.ID) < 1 {
		return false, sd.StaticDevice{}, nerr.Create("State must include device ID", "invaid-parameter")
	}

	//get and lock the device
	v := rc.getDeviceMu(toSave.ID)
	v.Lock()
	defer v.Unlock()

	dev, err := rc.getDevice(toSave.ID)
	if err != nil {
		return false, sd.StaticDevice{}, err.Addf("Couldn't store device event")
	}

	//make our edits
	merged, changes, err := shared.EditDeviceFromEvent(toSave, dev)
	if err != nil {
		return false, sd.StaticDevice{}, err.Addf("Couldn't store device event")
	}

	err = rc.putDevice(merged)

	return changes, merged, err
}

// StoreAndForwardEvent .
func (rc *RedisCache) StoreAndForwardEvent(event events.Event) (bool, *nerr.E) {
	return shared.ForwardAndStoreEvent(event, rc)
}

// GetCacheType .
func (rc *RedisCache) GetCacheType() string {
	return "redis"
}

// GetCacheName .
func (rc *RedisCache) GetCacheName() string {
	return rc.configuration.Name
}
