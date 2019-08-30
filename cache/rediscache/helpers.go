package rediscache

import (
	"encoding/json"
	"sync"

	"github.com/byuoitav/common/log"
	"github.com/byuoitav/common/nerr"
	sd "github.com/byuoitav/common/state/statedefinition"
	"github.com/byuoitav/event-forwarding-microservice/cache/shared"
	"github.com/go-redis/redis"
)

func (rc *RedisCache) getAllDeviceKeys() ([]string, *nerr.E) {
	return GetAllKeys(rc.devclient)
}
func (rc *RedisCache) getAllRoomKeys() ([]string, *nerr.E) {
	return GetAllKeys(rc.roomclient)
}

//GetAllKeys .
func GetAllKeys(client *redis.Client) ([]string, *nerr.E) {

	var newkeys []string
	var keys []string
	var cursor uint64
	var err error

	for {
		newkeys, cursor, err = client.Scan(cursor, "*", 50).Result()
		if err != nil {
			log.L.Errorf("Couldn't get all device keys: %v", err.Error())
			return keys, nerr.Translate(err).Addf("Couldn't get all device keys")
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

//assumes that we've already locked the device
func (rc *RedisCache) getDevice(id string) (sd.StaticDevice, *nerr.E) {

	var curDevice sd.StaticDevice
	by, err := rc.devclient.Get(id).Bytes()
	if err == redis.Nil {
		//device doesn't exist - we can create a new oneA
		var er *nerr.E
		curDevice, er = shared.GetNewDevice(id)
		if er != nil {
			log.L.Errorf("Error accessing redis cache: %v", er.Error())
			return sd.StaticDevice{}, er.Addf("Couldn't access redis cache")
		}
	} else if err != nil {
		log.L.Errorf("Error accessing redis cache: %v", err.Error())
		return sd.StaticDevice{}, nerr.Translate(err).Addf("Couldn't access redis cache")
	} else {
		err := json.Unmarshal(by, &curDevice)
		if err != nil {
			log.L.Errorf("Error decoding device from cache record: %v", err.Error())
			return sd.StaticDevice{}, nerr.Translate(err).Addf("Bad data in redis cluster: %s: %s", err.Error(), by)
		}
	}
	log.L.Debugf("%+v", curDevice.WebsocketCount)

	return curDevice, nil
}

//assumes that we've already locked the room
func (rc *RedisCache) getRoom(id string) (sd.StaticRoom, *nerr.E) {

	var curRoom sd.StaticRoom
	by, err := rc.roomclient.Get(id).Bytes()
	if err == redis.Nil {
		var er *nerr.E
		//device doesn't exist - we can create a new one
		log.L.Debugf("Getting new room: %v", id)
		curRoom, er = shared.GetNewRoom(id)
		if er != nil {
			log.L.Errorf("Couldn't generate new room from ID: %v", id)
			return curRoom, er.Addf("Couldn't get room %v", id)
		}
	} else if err != nil {
		log.L.Errorf("Error accessing redis cache: %v", err.Error())
		return sd.StaticRoom{}, nerr.Translate(err).Addf("Couldn't access redis cache")
	} else {
		err := json.Unmarshal(by, &curRoom)
		if err != nil {
			return curRoom, nerr.Translate(err).Addf("Couldn't unmarshal from rediscache")
		}
	}

	return curRoom, nil
}

//assumes that we've already locked the device
func (rc *RedisCache) putDevice(dev sd.StaticDevice) *nerr.E {
	log.L.Debugf("Putting device %v to redis cache", dev.DeviceID)

	b, err := json.Marshal(dev)
	if err != nil {
		log.L.Errorf("%v", err.Error())
		return nerr.Translate(err).Addf("Couldn't encode device: %v", err.Error())
	}

	err = rc.devclient.Set(dev.DeviceID, b, 0).Err()
	if err != nil {
		return nerr.Translate(err).Addf("Couldn't access redis cache")
	}

	return nil
}

//assumes that we've already locked the room
func (rc *RedisCache) putRoom(rm sd.StaticRoom) *nerr.E {

	b, err := json.Marshal(rm)
	if err != nil {
		log.L.Errorf("%v", err.Error())
		return nerr.Translate(err).Addf("Couldn't encode room: %v", err.Error())
	}

	err = rc.roomclient.Set(rm.RoomID, b, 0).Err()
	if err != nil {
		return nerr.Translate(err).Addf("Couldn't access redis cache")
	}
	return nil
}
