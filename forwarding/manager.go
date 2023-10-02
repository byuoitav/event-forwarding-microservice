package forwarding

import (
	"fmt"
	"sync"
	"time"

	"github.com/byuoitav/common/log"
	"github.com/byuoitav/event-forwarding-microservice/config"
	"github.com/byuoitav/event-forwarding-microservice/forwarding/managers"
)

// BufferManager is meant to handle buffering events/updates to the eventual forever home of the information
type BufferManager interface {
	Send(toSend interface{}) error
}

// Key is made up of the CacheName-DataType-EventType
// e.g. default-device-all or legacy-event-all
var managerMap map[string][]BufferManager
var managerInit sync.Once

func initManagers() {
	log.L.Infof("Initializing buffer managers")

	c := config.GetConfig()

	managerMap = make(map[string][]BufferManager)
	for _, i := range c.Forwarders {
		curName := fmt.Sprintf(fmt.Sprintf("%v-%v-%v", i.CacheName, i.DataType, i.EventType))
		switch i.Type {
		case config.ELKSTATIC:
			switch i.DataType {
			case config.ROOM:
				log.L.Infof("Initializing manager %v", curName)
				managerMap[curName] = append(managerMap[curName], managers.GetDefaultElkStaticRoomForwarder(
					i.Elk.URL,
					GetIndexFunction(i.Elk.IndexPattern, i.Elk.IndexRotationInterval),
					time.Duration(i.Interval)*time.Second,
					i.Elk.Upsert,
				))
			case config.DEVICE:
				log.L.Infof("Initializing manager %v", curName)
				managerMap[curName] = append(managerMap[curName], managers.GetDefaultElkStaticDeviceForwarder(
					i.Elk.URL,
					GetIndexFunction(i.Elk.IndexPattern, i.Elk.IndexRotationInterval),
					time.Duration(i.Interval)*time.Second,
					i.Elk.Upsert,
				))
			}
		case config.ELKTIMESERIES:
			log.L.Infof("Initializing manager %v", curName)
			managerMap[curName] = append(managerMap[curName], managers.GetDefaultElkTimeSeries(
				i.Elk.URL,
				GetIndexFunction(i.Elk.IndexPattern, i.Elk.IndexRotationInterval),
				time.Duration(i.Interval)*time.Second,
			))
		case config.COUCH:
			log.L.Infof("Initializing manager %v", curName)
			managerMap[curName] = append(managerMap[curName], managers.GetDefaultCouchDeviceBuffer(
				i.Couch.URL,
				i.Couch.DatabaseName,
				time.Duration(i.Interval)*time.Second,
			))
		case config.WEBSOCKET:
			log.L.Infof("Initializing Websocket manager %v", curName)
			managerMap[curName] = append(managerMap[curName], managers.GetDefaultWebsocketForwarder())

			//TODO: Add a case for Humio here
			/*
				case config.HUMIO:
					log.L.Infof("Initializing Humio manager %v", curName)
					managerMap[curName] = append(managerMap[curName], managers.GetDefaultHumioForwarder())
			*/
		}
	}

	log.L.Infof("Buffer managers initialized")
}

// GetManagersForType a
func GetManagersForType(cacheName, dataType, eventType string) []BufferManager {
	managerInit.Do(initManagers)

	log.L.Debugf("Getting %s managers for %v-%v", cacheName, dataType, eventType)
	v, ok := managerMap[fmt.Sprintf("%s-%s-%s", cacheName, dataType, eventType)]
	if !ok {
		log.L.Debugf("Unknown manager type: %v", fmt.Sprintf("%s-%s-%s", cacheName, dataType, eventType))
		return []BufferManager{}
	}
	return v
}

// GetIndexFunction .
func GetIndexFunction(indexPattern, rotationInterval string) func() string {
	switch rotationInterval {

	case config.DAILY:
		return func() string {
			return fmt.Sprintf("%v-%v", indexPattern, time.Now().Format("20060102"))
		}
	case config.WEEKLY:
		return func() string {
			yr, wk := time.Now().ISOWeek()
			return fmt.Sprintf("%v-%v%v", indexPattern, yr, wk)
		}
	case config.MONTHLY:
		return func() string {
			return fmt.Sprintf("%v-%v", indexPattern, time.Now().Format("200601"))
		}
	case config.YEARLY:
		return func() string {
			return fmt.Sprintf("%v-%v", indexPattern, time.Now().Format("2006"))
		}
	case config.NOROTATE:
		return func() string {
			return indexPattern

		}
	default:
		log.L.Fatalf("Unknown interval %v for index %v", rotationInterval, indexPattern)
	}
	return func() string {
		return indexPattern
	}
}
