package shared

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/byuoitav/event-forwarding-microservice/config"
	"github.com/byuoitav/event-forwarding-microservice/events"
	"github.com/byuoitav/event-forwarding-microservice/forwarding"
	sd "github.com/byuoitav/event-forwarding-microservice/state/statedefinition"
)

var alertRegex *regexp.Regexp
var stringtype = reflect.TypeOf("")
var timetype = reflect.TypeOf(time.Now())
var booltype = reflect.TypeOf((*bool)(nil))
var inttype = reflect.TypeOf((*int)(nil))
var float64type = reflect.TypeOf((*float64)(nil))

func init() {
	alertRegex = regexp.MustCompile(`alerts\..+`)
}

// PushAllDevices .
func PushAllDevices(c Cache) {
	//get all the records
	slog.Debug("Pushing updates for all devices to DELTA and ALL indexes")

	devs, err := c.GetAllDeviceRecords()
	if err != nil {
		slog.Error("Couldn't push all devices", "error", err)
		return
	}
	list := forwarding.GetManagersForType(config.DEVICE, config.DELTA)
	for i := range list {
		for j := range devs {
			er := list[i].Send(devs[j])
			if er != nil {
				slog.Warn("Problem sending all update for devices", "deviceID", devs[j].DeviceID, "error", er)
			}
		}
	}

	list = forwarding.GetManagersForType(config.DEVICE, config.ALL)
	for i := range list {
		for j := range devs {
			er := list[i].Send(devs[j])
			if er != nil {
				slog.Warn("Problem sending all update for devices", "deviceID", devs[j].DeviceID, "error", er)
			}
		}
	}

	slog.Info("Done sending update for all devices")

}

func updateHeartbeat(v events.Event, c Cache) {
	split := strings.Split(v.GeneratingSystem, "-")
	if len(split) < 3 {
		slog.Debug("invalid generating system: invalid-arguments")
		return
	}
	heartbeatEvent := events.Event{
		GeneratingSystem: "",
		Timestamp:        v.Timestamp,
		EventTags:        []string{events.Heartbeat},
		TargetDevice:     events.GenerateBasicDeviceInfo(v.GeneratingSystem),
		AffectedRoom:     events.GenerateBasicRoomInfo(fmt.Sprintf("%v-%v", split[0], split[1])),
		Key:              "auto-heartbeat",
		Value:            "ok",
		Data:             "ok",
	}
	_, err := ForwardAndStoreEvent(heartbeatEvent, c)
	if err != nil {
		slog.Debug("unable to create heartbeat event", "error", err)
	}
}

// ForwardAndStoreEvent .
func ForwardAndStoreEvent(v events.Event, c Cache) (bool, error) {
	if len(v.GeneratingSystem) > 0 && !events.ContainsAnyTags(v, events.Heartbeat) {
		// Try if we can
		updateHeartbeat(v, c)
	}
	//Forward All if they are not "fake" heartbeats
	if v.Key != "auto-heartbeat" {
		list := forwarding.GetManagersForType(config.EVENT, config.ALL)
		for i := range list {
			list[i].Send(v)
		}
	}

	//if it's an doesn't correspond to core or detail state we don't want to store it.
	if !events.ContainsAnyTags(v, events.CoreState, events.DetailState, events.Heartbeat) {
		return false, nil
	}

	//Cache
	changes, newDev, err := c.StoreDeviceEvent(sd.State{
		ID:    v.TargetDevice.DeviceID,
		Key:   v.Key,
		Time:  v.Timestamp,
		Value: v.Value,
		Tags:  v.EventTags,
	})

	if err != nil {
		return false, fmt.Errorf("Couldn't store and forward device event: %w", err)
	}

	list := forwarding.GetManagersForType(config.DEVICE, config.ALL)
	for i := range list {
		list[i].Send(newDev)
	}

	//if there are changes and it's not a heartbeat/hardware event
	if changes && !events.ContainsAnyTags(v, events.Heartbeat, events.HardwareInfo) {

		slog.Debug("Event resulted in changes")

		//get the event stuff to forward
		list = forwarding.GetManagersForType(config.EVENT, config.DELTA)
		for i := range list {
			list[i].Send(v)
		}

		list = forwarding.GetManagersForType(config.DEVICE, config.DELTA)
		for i := range list {
			list[i].Send(newDev)
		}
	}

	return changes, nil
}

// ForwardRoom .
func ForwardRoom(room sd.StaticRoom, changes bool, c Cache) error {
	list := forwarding.GetManagersForType(config.ROOM, config.ALL)
	for i := range list {
		list[i].Send(room)
	}

	if changes {
		list = forwarding.GetManagersForType(config.ROOM, config.DELTA)
		for i := range list {
			list[i].Send(room)
		}
	}
	return nil
}

// ForwardDevice .
func ForwardDevice(device sd.StaticDevice, changes bool, c Cache) error {
	list := forwarding.GetManagersForType(config.DEVICE, config.ALL)
	for i := range list {
		list[i].Send(device)
	}

	if changes {
		list = forwarding.GetManagersForType(config.DEVICE, config.DELTA)
		for i := range list {
			list[i].Send(device)
		}
	}
	return nil
}

/*
SetDeviceField returns the new device, as well as a boolean denoting if the field was already set to the provided value.

If passing in an alert, we assume that the value is a statdefinition.Alert. Alerts are denoted by alert.<alertName>. Alerts always return true.

NOTE: If in the code you can formulate a separate StaticDevice and compare it, defer to that approach, as the performance gain is quite significant.
*/
func SetDeviceField(key string, value interface{}, updateTime time.Time, t sd.StaticDevice) (bool, sd.StaticDevice, error) {
	val := reflect.TypeOf(t)
	slog.Debug("Kind", "kind", val.Kind())

	//check the update times case to see if we even need to proceed.
	v, ok := t.UpdateTimes[key]
	if ok {
		if v.After(updateTime) { //the current update is more recent
			slog.Debug("Discarding update as we have a more recent update", "key", key, "value", value, "deviceID", t.DeviceID)
			return false, t, nil
		}
	}

	// Alert special case:
	if alertRegex.MatchString(key) {
		v, ok := value.(sd.Alert)
		if !ok {
			return false, t, fmt.Errorf("Can't assign a non alert %v to alert value %v.", value, key)
		}

		if t.Alerts == nil {
			t.Alerts = make(map[string]sd.Alert)
		}

		//take just the name following the '.' value
		s := strings.Split(key, ".")

		t.Alerts[s[1]] = v
		return true, t, nil
	}

	var strvalue string

	//we translate to a string as this is the default use (events coming from individual room's event systems), and it makes the rest of the code cleaner.
	switch value.(type) {
	case int:
		strvalue = fmt.Sprintf("%v", value)
	case *int:
		strvalue = fmt.Sprintf("%v", *(value.(*int)))
	case bool:
		strvalue = fmt.Sprintf("%v", value)
	case *bool:
		strvalue = fmt.Sprintf("%v", *(value.(*bool)))
	case float64:
		strvalue = fmt.Sprintf("%v", value)
	case *float64:
		strvalue = fmt.Sprintf("%v", *(value.(*float64)))
	case time.Time:
		strvalue = fmt.Sprintf("\"%v\"", value.(time.Time).Format(time.RFC3339Nano))
	case string:
		strvalue = value.(string)
	case sd.Alert:
		return false, t, fmt.Errorf("Unsupported type %v. Alerts may only be used in an alert field (alert.X", reflect.TypeOf(value))
	default:
		return false, t, fmt.Errorf("Unsupported type %v.", reflect.TypeOf(value))

	}

	for i := 0; i < val.NumField(); i++ {
		cur := val.Field(i)
		jsonTag := cur.Tag.Get("json")

		jsonTag = strings.Split(jsonTag, ",")[0] //remove the 'omitempty' if any
		if jsonTag == key {
			slog.Debug("Found field", "key", key, "value", strvalue)
		} else {
			continue
		}

		curval := reflect.ValueOf(&t).Elem().Field(i)
		slog.Debug("Type", "type", curval.Type())

		if curval.CanSet() {
			//check for nil UpdateTimes map
			if t.UpdateTimes == nil {
				t.UpdateTimes = make(map[string]time.Time)
			}

			thistype := curval.Type()
			switch thistype {
			case stringtype:
				slog.Debug("string type")
				var a string
				err := json.Unmarshal([]byte("\""+strvalue+"\""), &a)
				if err != nil {
					slog.Debug("ERROR unmarshalling string", "error", err)
					return false, t, fmt.Errorf("Couldn't unmarshal strvalue %v into the field %v: %w", strvalue, key, err)
				}

				//update the time that it was 'last' set
				t.UpdateTimes[key] = updateTime

				prevValue := curval.Interface().(string)

				slog.Debug("PrevValue", "prev", prevValue, "cur", a)

				if a == prevValue {
					//no change
					return false, t, nil
				}

				//set it
				curval.SetString(a)
				return true, t, nil

			case timetype:
				slog.Debug("time type")
				var a time.Time
				err := json.Unmarshal([]byte(strvalue), &a)
				if err != nil {
					return false, t, fmt.Errorf("Couldn't unmarshal strvalue %v into the field %v: %w", strvalue, key, err)
				}

				//update the time that it was 'last' set
				t.UpdateTimes[key] = updateTime

				prevValue := curval.Interface().(time.Time)
				if prevValue.Equal(a) {
					//no change
					return false, t, nil
				}

				//set it
				curval.Set(reflect.ValueOf(a))
				return true, t, nil

			case booltype:
				slog.Debug("bool type")
				var a bool
				err := json.Unmarshal([]byte(strvalue), &a)
				if err != nil {
					return false, t, fmt.Errorf("Couldn't unmarshal strvalue %v into the field %v: %w", strvalue, key, err)
				}

				//update the time that it was 'last' set
				t.UpdateTimes[key] = updateTime

				prevValue := curval.Interface().(*bool)
				if prevValue != nil && *prevValue == a {
					//no change
					return false, t, nil
				}

				//set it
				curval.Set(reflect.ValueOf(&a))
				return true, t, nil

			case inttype:
				slog.Debug("int type")
				var a int
				err := json.Unmarshal([]byte(strvalue), &a)
				if err != nil {
					slog.Warn("Error unmarshalling int", "error", err)
					return false, t, fmt.Errorf("Couldn't unmarshal strvalue %v into the field %v: %w", strvalue, key, err)
				}

				//update the time that it was 'last' set
				t.UpdateTimes[key] = updateTime

				prevValue := curval.Interface().(*int)
				if prevValue != nil && *prevValue == a {
					//no change
					return false, t, nil
				}

				//set it
				curval.Set(reflect.ValueOf(&a))

				return true, t, nil
			case float64type:
				slog.Debug("float64 type")
				var a float64
				err := json.Unmarshal([]byte(strvalue), &a)
				if err != nil {
					slog.Warn("Error unmarshalling float64", "error", err)
					return false, t, fmt.Errorf("Couldn't unmarshal strvalue %v into the field %v: %w", strvalue, key, err)
				}

				//update the time that it was 'last' set
				t.UpdateTimes[key] = updateTime

				prevValue := curval.Interface().(*float64)
				if prevValue != nil && *prevValue == a {
					//no change
					return false, t, nil
				}

				//set it
				curval.Set(reflect.ValueOf(&a))

				return true, t, nil
			default:
				return false, t, fmt.Errorf("Field %v is an unsupported type %v", key, thistype)
			}

		} else {
			return false, t, fmt.Errorf("There was a problem setting field %v, field is not settable", key)
		}
	}

	//if we made it here, it means that the field isn't found
	return false, t, fmt.Errorf("Field %v isn't a valid field for a device.", key)
}

// HasTag .
func HasTag(toCheck string, tags []string) bool {
	for i := range tags {
		if toCheck == tags[i] {
			return true
		}
	}
	return false
}

var translationMap = map[string]string{
	"D":  "display",
	"CP": "control-processor",

	"DSP":     "digital-signal-processor",
	"DMPS":    "dmps",
	"PC":      "computer",
	"SW":      "video-switcher",
	"MICJK":   "microphone-jack",
	"SP":      "scheduling-panel",
	"MIC":     "microphone",
	"DS":      "divider-sensor",
	"GW":      "gateway",
	"VIA":     "via",
	"HDMI":    "hdmi",
	"RX":      "receiver",
	"TX":      "transmitter",
	"RCV":     "microphone-reciever",
	"EN":      "encoder",
	"LIN":     "line-in",
	"OF":      "overflow",
	"MEDIA":   "media",
	"TECLITE": "tec-lite",
	"CUSTOM":  "custom",
	"SD":      "tec-sd",
}

// GetDeviceTypeByID .
func GetDeviceTypeByID(id string) string {

	split := strings.Split(id, "-")
	if len(split) != 3 {
		slog.Warn("[dispatcher] Invalid hostname for device", "id", id)
		return ""
	}

	for pos, char := range split[2] {
		if unicode.IsDigit(char) {
			val, ok := translationMap[split[2][:pos]]
			if !ok {
				slog.Warn("Invalid device type", "type", split[2][:pos])
				return "unknown"
			}
			return val
		}
	}

	slog.Warn("no valid translation", "type", split[2])
	return ""
}
