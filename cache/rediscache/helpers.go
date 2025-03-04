package rediscache

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	//sd "github.com/byuoitav/common/state/statedefinition"
	"github.com/byuoitav/event-forwarding-microservice/cache/shared"
	sd "github.com/byuoitav/event-forwarding-microservice/statedefinition"
	"github.com/go-redis/redis"
)

func (rc *RedisCache) getAllDeviceKeys() ([]string, error) {
	return GetAllKeys(rc.devclient)
}
func (rc *RedisCache) getAllRoomKeys() ([]string, error) {
	return GetAllKeys(rc.roomclient)
}

// GetAllKeys .
func GetAllKeys(client *redis.Client) ([]string, error) {

	var newkeys []string
	var keys []string
	var cursor uint64
	var err error

	for {
		newkeys, cursor, err = client.Scan(cursor, "*", 50).Result()
		if err != nil {
			slog.Error(fmt.Sprintf("Couldn't get all device keys: %v", err.Error()))
			return keys, err
		}
		keys = append(keys, newkeys...)

		if cursor == 0 {
			break
		}
	}

	return keys, nil
}

func (rc *RedisCache) getDeviceMu(id string) *sync.Mutex {

	rc.devLock.RLock()

	//check to see if someone else is already editing the same device, if so, we wait for it
	v, ok := rc.devmu[id]
	rc.devLock.RUnlock()

	//First access to the device
	if !ok {
		//we need to add it
		rc.devLock.Lock()
		v = &sync.Mutex{}
		//make sure no one else added it while we were waiting
		v, ok = rc.devmu[id]
		if !ok {
			v = &sync.Mutex{}
			rc.devmu[id] = v
		}
		rc.devLock.Unlock()
	}
	return v
}
func (rc *RedisCache) getRoomMu(id string) *sync.Mutex {

	rc.roomLock.RLock()

	//check to see if someone else is already editing the same device, if so, we wait for it
	v, ok := rc.roommu[id]
	rc.roomLock.RUnlock()

	//First access to the device
	if !ok {
		//we need to add it
		rc.roomLock.Lock()
		v = &sync.Mutex{}
		//make sure no one else added it while we were waiting
		v, ok = rc.roommu[id]
		if !ok {
			v = &sync.Mutex{}

			rc.roommu[id] = v
		}
		rc.roomLock.Unlock()
	}
	return v
}

// assumes that we've already locked the device
func (rc *RedisCache) getDevice(id string) (sd.StaticDevice, error) {

	var curDevice sd.StaticDevice
	by, err := rc.devclient.Get(id).Bytes()
	if err == redis.Nil {
		//device doesn't exist - we can create a new oneA
		var er error
		curDevice, er = shared.GetNewDevice(id)
		if er != nil {
			slog.Error(fmt.Sprintf("Error accessing redis cache: %v", er.Error()))
			return sd.StaticDevice{}, er
		}
	} else if err != nil {
		slog.Error(fmt.Sprintf("Error accessing redis cache: %v", err.Error()))
		return sd.StaticDevice{}, err
	} else {
		err := json.Unmarshal(by, &curDevice)
		if err != nil {
			slog.Error(fmt.Sprintf("Error decoding device from cache record: %v", err.Error()))
			return sd.StaticDevice{}, err
		}
	}
	slog.Debug(fmt.Sprintf("%+v", curDevice.WebsocketCount))

	return curDevice, nil
}

// assumes that we've already locked the room
func (rc *RedisCache) getRoom(id string) (sd.StaticRoom, error) {

	var curRoom sd.StaticRoom
	by, err := rc.roomclient.Get(id).Bytes()
	if err == redis.Nil {
		var er error
		//device doesn't exist - we can create a new one
		slog.Debug(fmt.Sprintf("Getting new room: %v", id))
		curRoom, er = shared.GetNewRoom(id)
		if er != nil {
			slog.Error(fmt.Sprintf("Couldn't generate new room from ID: %v", id))
			return curRoom, er
		}
	} else if err != nil {
		slog.Error(fmt.Sprintf("Error accessing redis cache: %v", err.Error()))
		return sd.StaticRoom{}, err
	} else {
		err := json.Unmarshal(by, &curRoom)
		if err != nil {
			slog.Debug("Trying to Unmarshal", "getRoom", "error")
			return curRoom, err
		}
	}

	return curRoom, nil
}

// assumes that we've already locked the device
func (rc *RedisCache) putDevice(dev sd.StaticDevice) error {
	slog.Debug(fmt.Sprintf("Putting device %v to redis cache", dev.DeviceID))

	b, err := json.Marshal(dev)
	if err != nil {
		slog.Error(fmt.Sprintf("%v", err.Error()))
		return err
	}

	err = rc.devclient.Set(dev.DeviceID, b, 0).Err()
	if err != nil {
		return err
	}

	return nil
}

// assumes that we've already locked the room
func (rc *RedisCache) putRoom(rm sd.StaticRoom) error {

	b, err := json.Marshal(rm)
	if err != nil {
		slog.Error(fmt.Sprintf("%v", err.Error()))
		return err
	}

	err = rc.roomclient.Set(rm.RoomID, b, 0).Err()
	if err != nil {
		return err
	}
	return nil
}
