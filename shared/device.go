package shared

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/byuoitav/event-forwarding-microservice/events"
	sd "github.com/byuoitav/event-forwarding-microservice/state/statedefinition"
)

// EditDeviceFromEvent .
func EditDeviceFromEvent(e sd.State, device sd.StaticDevice) (sd.StaticDevice, bool, error) {
	var changes bool
	var err error

	if HasTag(events.CoreState, e.Tags) {
		if s, ok := e.Value.(string); ok {
			if len(s) < 1 {
				// blank value: we don't do anything with this
				return device, false, nil
			}
		}
	}

	if e.Key == "responsive" {
		strVal, ok := e.Value.(string)
		if ok && strings.EqualFold(strVal, "ok") {
			changes, device, err = SetDeviceField(
				"last-health-success",
				e.Time,
				e.Time,
				device,
			)
			changes, device, err = SetDeviceField(
				"last-heartbeat",
				e.Time,
				e.Time,
				device,
			)
		}
	} else if HasTag(events.Heartbeat, e.Tags) {
		changes, device, err = SetDeviceField(
			"last-heartbeat",
			e.Time,
			e.Time,
			device,
		)
	} else {
		changes, device, err = SetDeviceField(
			e.Key,
			e.Value,
			e.Time,
			device,
		)
	}
	if err != nil {
		return device, false, err
	}

	// if it has a user-generated tag
	if HasTag(events.UserGenerated, e.Tags) {
		device.LastUserInput = e.Time
	}

	// i'm just going to assume yeah, ask joe later
	if HasTag(events.CoreState, e.Tags) || HasTag(events.DetailState, e.Tags) {
		device.LastStateReceived = e.Time
	}

	return device, changes, nil
}

// GetNewDevice .
func GetNewDevice(id string) (sd.StaticDevice, error) {

	rm := strings.Split(id, "-")
	if len(rm) != 3 {
		slog.Error("Invalid Device", "id", id)
		return sd.StaticDevice{}, errors.New(fmt.Sprintf("Can't build device manager: invalid ID %v", id))
	}

	device := sd.StaticDevice{
		DeviceID:              id,
		Room:                  rm[0] + "-" + rm[1],
		Building:              rm[0],
		UpdateTimes:           make(map[string]time.Time),
		Control:               id,
		EnableNotifications:   id,
		SuppressNotifications: id,
		ViewDashboard:         id,
		DeviceType:            GetDeviceTypeByID(id),
	}
	return device, nil
}
