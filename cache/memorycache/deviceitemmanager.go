package memorycache

import (
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/byuoitav/event-forwarding-microservice/cache/shared"
	sd "github.com/byuoitav/event-forwarding-microservice/state/statedefinition"
)

/*
DeviceItemManager handles managing access to a single device in a cache. Changes to the device are submitted via the IncomingWriteChan and reads are submitted via the IncomingReadChan.
*/
type DeviceItemManager struct {
	WriteRequests chan DeviceTransactionRequest //channel to buffer changes to the device.
	ReadRequests  chan chan sd.StaticDevice
	KillChannel   chan bool
}

// DeviceTransactionRequest is submitted to read/write a the device being managed by this manager
// If both a MergeDevice and an Event are submitted teh MergeDevice will be processed first
type DeviceTransactionRequest struct {
	ResponseChan chan DeviceTransactionResponse

	// If you want to update the managed device with the values in this device. Note that the lastest edit timestamp field controls which fields will be kept in a merge.
	MergeDeviceEdit bool
	MergeDevice     sd.StaticDevice

	// If you want to store an event and return changes (if any)
	EventEdit bool
	Event     sd.State
}

// DeviceTransactionResponse .
type DeviceTransactionResponse struct {
	Changes   bool            //if the Transaction Request resulted in changes
	NewDevice sd.StaticDevice //the updated device with the changes included in the Transaction request included
	Error     error           //if there were errors
}

// GetNewDeviceManager .
func GetNewDeviceManager(id string) (DeviceItemManager, error) {
	a := DeviceItemManager{
		WriteRequests: make(chan DeviceTransactionRequest, 100),
		ReadRequests:  make(chan chan sd.StaticDevice, 100),
	}

	dev, err := shared.GetNewDevice(id)
	if err != nil {
		return a, err
	}
	go StartDeviceManager(a, dev)
	return a, nil
}

// GetNewDeviceManagerWithDevice will assume overwriting of all the info, won't initialize to default values
func GetNewDeviceManagerWithDevice(dev sd.StaticDevice) (DeviceItemManager, error) {
	a := DeviceItemManager{
		WriteRequests: make(chan DeviceTransactionRequest, 100),
		ReadRequests:  make(chan chan sd.StaticDevice, 100),
	}

	rm := strings.Split(dev.DeviceID, "-")
	if len(rm) != 3 {
		return DeviceItemManager{}, errors.New("Invalid device, must have deviceID")
	}

	go StartDeviceManager(a, dev)
	return a, nil
}

// StartDeviceManager is a blocking call to start that device manager listening over the read and write channels.
func StartDeviceManager(m DeviceItemManager, device sd.StaticDevice) {
	var merged sd.StaticDevice
	var changes bool
	var err error

	for {
		select {

		case <-m.KillChannel:
			slog.Info("Killing device manager", "deviceID", device.DeviceID)
			return

		case write := <-m.WriteRequests:
			if write.ResponseChan == nil {
				continue
			}

			if write.MergeDeviceEdit {
				if write.MergeDevice.DeviceID != device.DeviceID {
					write.ResponseChan <- DeviceTransactionResponse{Error: errors.New("Can't change the ID of a device"), NewDevice: device, Changes: false}
					continue
				}
				_, merged, changes, err = sd.CompareDevices(device, write.MergeDevice)

				if err != nil {
					write.ResponseChan <- DeviceTransactionResponse{Error: err, Changes: false}
					continue
				}
				if changes {
					//only reassign if we have to
					device = merged
				}
			}

			if write.EventEdit {
				merged, changes, err = shared.EditDeviceFromEvent(write.Event, device)
				device = merged
			}

			write.ResponseChan <- DeviceTransactionResponse{Error: err, NewDevice: device, Changes: changes}
		case read := <-m.ReadRequests:
			//just send it back
			if read != nil {
				//we need to
				newdev := device
				newdev.UpdateTimes = make(map[string]time.Time)
				for k, v := range device.UpdateTimes {
					newdev.UpdateTimes[k] = v
				}
				read <- newdev
			}
		}
	}
}
