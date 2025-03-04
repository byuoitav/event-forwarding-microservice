package rediscache

import (
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	//"github.com/byuoitav/common/nerr"
	//sd "github.com/byuoitav/common/state/statedefinition"
	sd "github.com/byuoitav/event-forwarding-microservice/statedefinition"

	//"github.com/byuoitav/common/v2/events"
	"github.com/byuoitav/event-forwarding-microservice/cache/shared"
	"github.com/byuoitav/event-forwarding-microservice/config"
	customerror "github.com/byuoitav/event-forwarding-microservice/error"
	"github.com/byuoitav/event-forwarding-microservice/events"
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
func MakeRedisCache(pushCron string, configuration config.Cache) (shared.Cache, error) {

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
		errLog := &customerror.StandardError{
			Message: fmt.Sprintf("Couldn't communicate with redis server at %v", addr),
		}
		return toReturn, errLog
	}

	toReturn.roomclient = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: pass,
		DB:       configuration.RedisInfo.RoomDatabase,
	})

	_, err = toReturn.roomclient.Ping().Result()
	if err != nil {
		errLog := &customerror.StandardError{
			Message: fmt.Sprintf("Couldn't communicate with redis server at %v", addr),
		}
		return toReturn, errLog
	}

	return toReturn, nil
}

// CheckAndStoreDevice .
func (rc *RedisCache) CheckAndStoreDevice(device sd.StaticDevice) (bool, sd.StaticDevice, error) {
	v := rc.getDeviceMu(device.DeviceID)
	debugLog := fmt.Sprintf("Waiting for lock on device %v", device.DeviceID)
	slog.Debug(debugLog)
	v.Lock()
	defer v.Unlock()
	slog.Debug(fmt.Sprintf("Working on device %v", device.DeviceID))

	dev, err := rc.getDevice(device.DeviceID)
	if err != nil {
		errLog := &customerror.StandardError{
			Message: "Couldn't check and store device",
		}
		return false, dev, errLog
	}
	slog.Debug(fmt.Sprintf("Device: %+v", device))

	_, merged, changes, err := sd.CompareDevices(device, dev)
	if err != nil {
		errLog := &customerror.StandardError{
			Message: "Couldn't check and store device",
		}
		return false, dev, errLog
	}
	slog.Debug(fmt.Sprintf("Merged: %+v", merged))

	err = rc.putDevice(merged)
	if err != nil {
		errLog := &customerror.StandardError{
			Message: "Couldn't check and store device",
		}
		return false, device, errLog
	}

	err = shared.ForwardDevice(merged, changes, rc)

	return changes, merged, err
}

// CheckAndStoreRoom .
func (rc *RedisCache) CheckAndStoreRoom(room sd.StaticRoom) (bool, sd.StaticRoom, error) {
	v := rc.getRoomMu(room.RoomID)
	v.Lock()
	defer v.Unlock()

	slog.Debug(fmt.Sprintf("checking and storing room %v", room))
	rm, err := rc.getRoom(room.RoomID)
	if err != nil {
		errLog := &customerror.StandardError{
			Message: "Couldn't check and store device",
		}
		return false, rm, errLog
	}
	slog.Debug(fmt.Sprintf("got room: %v", rm))

	_, merged, changes, err := sd.CompareRooms(rm, room)
	if err != nil {
		errLog := &customerror.StandardError{
			Message: "Couldn't check and store device",
		}
		return false, rm, errLog
	}

	slog.Debug(fmt.Sprintf("changes: %v, Room: %+v", changes, merged))

	if changes {
		err := rc.putRoom(merged)
		if err != nil {
			errLog := &customerror.StandardError{
				Message: "Couldn't check and store device",
			}
			return false, room, errLog
		}
	}

	err = shared.ForwardRoom(merged, changes, rc)

	return changes, merged, err

}

// GetDeviceRecord .
func (rc *RedisCache) GetDeviceRecord(deviceID string) (sd.StaticDevice, error) {
	//get and lock the device
	v := rc.getDeviceMu(deviceID)
	v.Lock()
	defer v.Unlock()

	//get the device
	return rc.getDevice(deviceID)
}

// GetRoomRecord .
func (rc *RedisCache) GetRoomRecord(roomID string) (sd.StaticRoom, error) {
	//get and lock the room
	v := rc.getRoomMu(roomID)
	v.Lock()
	defer v.Unlock()

	//get the device
	return rc.getRoom(roomID)
}

// GetAllDeviceRecords .
func (rc *RedisCache) GetAllDeviceRecords() ([]sd.StaticDevice, error) {

	keys, er := rc.getAllDeviceKeys()
	if er != nil {
		errLog := &customerror.StandardError{
			Message: "Couldn't get all device records",
		}
		return []sd.StaticDevice{}, errLog
	}

	result, err := rc.devclient.MGet(keys...).Result()
	if err != nil {
		errLog := &customerror.StandardError{
			Message: "Couldn't get all device records",
		}
		return []sd.StaticDevice{}, errLog
	}

	var toReturn []sd.StaticDevice

	for i := range result {
		var tmp sd.StaticDevice

		err := json.Unmarshal([]byte(result[i].(string)), &tmp)
		if err != nil {
			errLog := &customerror.StandardError{
				Message: "Couldn't get all device records",
			}
			return []sd.StaticDevice{}, errLog
		}
		toReturn = append(toReturn, tmp)
	}

	return toReturn, nil
}

// GetAllRoomRecords .
func (rc *RedisCache) GetAllRoomRecords() ([]sd.StaticRoom, error) {
	keys, er := rc.getAllRoomKeys()
	if er != nil {
		errLog := &customerror.StandardError{
			Message: "Couldn't get all device records",
		}
		return []sd.StaticRoom{}, errLog
	}

	result, err := rc.roomclient.MGet(keys...).Result()
	if err != nil {
		errLog := &customerror.StandardError{
			Message: "Couldn't get all device records",
		}
		return []sd.StaticRoom{}, errLog
	}

	var toReturn []sd.StaticRoom

	for i := range result {
		var tmp sd.StaticRoom

		err := json.Unmarshal([]byte(result[i].(string)), &tmp)
		if err != nil {
			errLog := &customerror.StandardError{
				Message: "Couldn't get all device records",
			}
			return []sd.StaticRoom{}, errLog
		}
		toReturn = append(toReturn, tmp)
	}

	return toReturn, nil
}

// RemoveDevice .
func (rc *RedisCache) RemoveDevice(id string) error {
	v := rc.getDeviceMu(id)
	v.Lock()
	defer v.Unlock()

	// This is really muddy code: wrapping this makes my brain hurt.
	// I am leaving this here as a tombstone to really weird code
	// return nerr.Translate(rc.devclient.Del(id).Err()).Addf("Couldn't remove device %v", id)

	err := rc.devclient.Del(id).Err()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			slog.Debug("Device does not exist in the Redis Cache")
			// If the key doesn't exist, it's not a failure
			return nil
		}
		errLog := &customerror.StandardError{
			Message: fmt.Sprintf("Redis delete error for device %q: %w", id, err),
		}
		return errLog
	}

	return nil
}

// RemoveRoom .
func (rc *RedisCache) RemoveRoom(id string) error {

	v := rc.getRoomMu(id)
	v.Lock()
	defer v.Unlock()

	// This is combining the error return while running the actual function
	// I am leaving this here as a tombstone to really weird code - Don't do this
	// return nerr.Translate(rc.roomclient.Del(id).Err()).Addf("Couldn't remove room %v", id)

	err := rc.roomclient.Del(id).Err()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// If the key doesn't exist, it's not a failure
			slog.Debug("Room does not exist in the Redis Cache")
			return nil
		}
		errLog := &customerror.StandardError{
			Message: fmt.Sprintf("Redis delete error for device %q: %w", id, err),
		}
		return errLog
	}

	slog.Debug(fmt.Sprintf("RemoveRoom removed the room %v", id))
	return nil
}

// NukeRoom .
func (rc *RedisCache) NukeRoom(id string) ([]string, error) {
	keys, err := rc.devclient.Keys(fmt.Sprintf("%s*", id)).Result()
	if err != nil {
		retErr := &customerror.StandardError{
			Message: "Error Retrieving device info from Redis",
		}
		return []string{}, retErr
	}

	er := rc.RemoveRoom(id)
	if er != nil {
		remErr := &customerror.StandardError{
			Message: fmt.Sprintf("Error removing room %v from Redis in NukeRoom", id),
		}
		return []string{}, remErr
	}

	for i := range keys {
		er = rc.RemoveDevice(keys[i])
		if er != nil {
			devErr := &customerror.StandardError{
				Message: fmt.Sprintf("Error removing device %v from Redis in NukeRoom", i),
			}
			return []string{}, devErr
		}
	}

	return keys, nil
}

// StoreDeviceEvent .
func (rc *RedisCache) StoreDeviceEvent(toSave sd.State) (bool, sd.StaticDevice, error) {
	if len(toSave.ID) < 1 {
		devErr := &customerror.StandardError{
			Message: "State must include device ID",
		}
		return false, sd.StaticDevice{}, devErr
	}

	//get and lock the device
	v := rc.getDeviceMu(toSave.ID)
	v.Lock()
	defer v.Unlock()

	dev, err := rc.getDevice(toSave.ID)
	if err != nil {
		//return false, sd.StaticDevice{}, err.Addf("Couldn't store device event")
		return false, sd.StaticDevice{}, err
	}

	//make our edits
	merged, changes, err := shared.EditDeviceFromEvent(toSave, dev)
	if err != nil {
		//return false, sd.StaticDevice{}, err.Addf("Couldn't store device event")
		return false, sd.StaticDevice{}, err
	}

	err = rc.putDevice(merged)

	return changes, merged, err
}

// StoreAndForwardEvent .
func (rc *RedisCache) StoreAndForwardEvent(event events.Event) (bool, error) {
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
