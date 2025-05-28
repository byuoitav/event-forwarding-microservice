// forwarding sets up the individual forwarders for each specified forwarder in the config
package forwarding

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/byuoitav/event-forwarding-microservice/config"
	"github.com/byuoitav/event-forwarding-microservice/forwarding/managers"
)

// BufferManager is meant to handle buffering events/updates to the eventual forever home of the information
type BufferManager interface {
	Send(toSend interface{}) error
}

// Key is made up of the DataType-EventType
// e.g. default-device-all or legacy-event-all
var managerMap map[string][]BufferManager
var managerInit sync.Once

func initManagers() {
	slog.Info("Initializing buffer managers")

	c := config.GetConfig()

	managerMap = make(map[string][]BufferManager)
	for _, i := range c.Forwarders {
		curName := fmt.Sprintf(fmt.Sprintf("%v-%v", i.DataType, i.EventType))
		switch i.Type {
		case config.ELKSTATIC:
			switch i.DataType {
			case config.ROOM:
				slog.Info("Initializing manager", "name", curName)
				managerMap[curName] = append(managerMap[curName], managers.GetDefaultElkStaticRoomForwarder(
					i.Elk.URL,
					GetIndexFunction(i.Elk.IndexPattern, i.Elk.IndexRotationInterval),
					time.Duration(i.Interval)*time.Second,
					i.Elk.Upsert,
				))
			case config.DEVICE:
				slog.Info("Initializing manager", "name", curName)
				managerMap[curName] = append(managerMap[curName], managers.GetDefaultElkStaticDeviceForwarder(
					i.Elk.URL,
					GetIndexFunction(i.Elk.IndexPattern, i.Elk.IndexRotationInterval),
					time.Duration(i.Interval)*time.Second,
					i.Elk.Upsert,
				))
			}
		case config.ELKTIMESERIES:
			slog.Info("Initializing manager", "name", curName)
			managerMap[curName] = append(managerMap[curName], managers.GetDefaultElkTimeSeries(
				i.Elk.URL,
				GetIndexFunction(i.Elk.IndexPattern, i.Elk.IndexRotationInterval),
				time.Duration(i.Interval)*time.Second,
			))
		case config.COUCH:
			slog.Info("Initializing manager", "name", curName)
			managerMap[curName] = append(managerMap[curName], managers.GetDefaultCouchDeviceBuffer(
				i.Couch.URL,
				i.Couch.DatabaseName,
				time.Duration(i.Interval)*time.Second,
			))
		case config.WEBSOCKET:
			slog.Info("Initializing Websocket manager", "name", curName)
			managerMap[curName] = append(managerMap[curName], managers.GetDefaultWebsocketForwarder())

		case config.HUMIO:
			slog.Info("Initializing Humio manager", "name", curName)
			managerMap[curName] = append(managerMap[curName], managers.GetDefaultHumioForwarder(
				time.Duration(i.Humio.Interval)*time.Second,
				i.Humio.BufferSize,
				i.Humio.IngestToken,
			))
		}
	}
	slog.Info("Buffer managers initialized")
}

// GetManagersForType a
func GetManagersForType(cacheName, dataType, eventType string) []BufferManager {
	managerInit.Do(initManagers)

	slog.Debug("Getting managers", "dataType", dataType, "eventType", eventType)
	v, ok := managerMap[fmt.Sprintf("%s-%s-%s", cacheName, dataType, eventType)]
	if !ok {
		slog.Debug("Unknown manager type", "type", fmt.Sprintf("%s-%s-%s", cacheName, dataType, eventType))
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
		slog.Error("Unknown interval for index", "interval", rotationInterval, "index", indexPattern)
	}
	return func() string {
		return indexPattern
	}
}
