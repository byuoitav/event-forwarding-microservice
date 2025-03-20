package shared

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"regexp"
	"strings"
	"time"
	"unicode"

	sd "github.com/byuoitav/common/state/statedefinition"
	"github.com/byuoitav/common/v2/events"
	"github.com/byuoitav/event-forwarding-microservice/config"
	"github.com/byuoitav/event-forwarding-microservice/forwarding"
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

func updateHeartbeat(v events.Event) {
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
	_, err := ForwardAndStoreEvent(heartbeatEvent)
	if err != nil {
		slog.Debug("unable to create heartbeat event", "error", err.Error())
	}
}

// ForwardAndStoreEvent .
func ForwardAndStoreEvent(v events.Event) (bool, error) {
	if len(v.GeneratingSystem) > 0 && !events.ContainsAnyTags(v, events.Heartbeat) {
		// Try if we can
		updateHeartbeat(v)
	}
	// Forward All if they are not "fake" heartbeats
	if v.Key != "auto-heartbeat" {
		list := forwarding.GetManagersForType(config.EVENT, config.ALL)
		for i := range list {
			list[i].Send(v)
		}
	}

	// if it doesn't correspond to core or detail state we don't want to store it.
	if !events.ContainsAnyTags(v, events.CoreState, events.DetailState, events.Heartbeat) {
		return false, nil
	}

	return !events.ContainsAnyTags(v, events.Heartbeat, events.HardwareInfo), nil
}

// ForwardRoom .
func ForwardRoom(room sd.StaticRoom, changes bool) error {
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
func ForwardDevice(device sd.StaticDevice, changes bool) error {
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

If passing in an alert, we assume that the value is a statdefinition.Alert.	Alerts are denoted by alert.<alertName>. Alerts always return true.

NOTE: If in the code you can formulate a separate StaticDevice and compare it, defer to that approach, as the performance gain is quite significant.
*/
func SetDeviceField(key string, value interface{}, updateTime time.Time, t sd.StaticDevice) (bool, sd.StaticDevice, error) {
	val := reflect.TypeOf(t)
	slog.Debug("Kind", "kind", val.Kind())

	// check the update times case to see if we even need to proceed.
	v, ok := t.UpdateTimes[key]
	if ok {
		if v.After(updateTime) { // the current update is more recent
			slog.Info("Discarding update for device as we have a more recent update", "key", key, "value", value, "device", t.DeviceID)
			return false, t, nil
		}
	}

	/*
		Alert special case:

			We need to check key to see if it's an alert, if it is, we just check the type of alert, and assume that we're completely overhauling the whole subvalue.
			Assume that alert updates always result in an 'update'
	*/
	if alertRegex.MatchString(key) {
		v, ok := value.(sd.Alert)
		if !ok {
			return false, t, fmt.Errorf("can't assign a non alert %v to alert value %v", value, key)
		}

		if t.Alerts == nil {
			t.Alerts = make(map[string]sd.Alert)
		}

		// take just the name following the '.' value
		s := strings.Split(key, ".")

		t.Alerts[s[1]] = v
		return true, t, nil
	}

	var strvalue string

	// we translate to a string as this is the default use (events coming from individual room's event systems), and it makes the rest of the code cleaner.
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
		return false, t, errors.New("unsupported type. Alerts may only be used in an alert field (alert.X)")
	default:
		return false, t, fmt.Errorf("unsupported type %v", reflect.TypeOf(value))
	}

	for i := 0; i < val.NumField(); i++ {
		cur := val.Field(i)
		jsonTag := cur.Tag.Get("json")

		jsonTag = strings.Split(jsonTag, ",")[0] // remove the 'omitempty' if any
		if jsonTag == key {
			slog.Debug("Found", "key", key, "value", strvalue)
		} else {
			continue
		}

		curval := reflect.ValueOf(&t).Elem().Field(i)
		slog.Debug("Type", "type", curval.Type())

		if curval.CanSet() {
			// check for nil UpdateTimes map
			if t.UpdateTimes == nil {
				t.UpdateTimes = make(map[string]time.Time)
			}

			thistype := curval.Type()
			switch thistype {
			case stringtype:
				slog.Debug("string")
				var a string
				err := json.Unmarshal([]byte("\""+strvalue+"\""), &a)
				if err != nil {
					slog.Debug("ERROR", "error", err.Error())
					return false, t, fmt.Errorf("couldn't unmarshal strvalue %v into the field %v: %w", strvalue, key, err)
				}

				// update the time that it was 'last' set
				t.UpdateTimes[key] = updateTime

				prevValue := curval.Interface().(string)

				slog.Debug("PrevValue", "prevValue", prevValue, "curValue", a)

				if a == prevValue {
					// no change
					return false, t, nil
				}

				// set it
				curval.SetString(a)
				return true, t, nil

			case timetype:
				slog.Debug("time")
				var a time.Time
				err := json.Unmarshal([]byte(strvalue), &a)
				if err != nil {
					return false, t, fmt.Errorf("couldn't unmarshal strvalue %v into the field %v: %w", strvalue, key, err)
				}

				// update the time that it was 'last' set
				t.UpdateTimes[key] = updateTime

				prevValue := curval.Interface().(time.Time)
				if prevValue.Equal(a) {
					// no change
					return false, t, nil
				}

				// set it
				curval.Set(reflect.ValueOf(a))
				return true, t, nil

			case booltype:
				slog.Debug("bool")
				var a bool
				err := json.Unmarshal([]byte(strvalue), &a)
				if err != nil {
					return false, t, fmt.Errorf("couldn't unmarshal strvalue %v into the field %v: %w", strvalue, key, err)
				}

				// update the time that it was 'last' set
				t.UpdateTimes[key] = updateTime

				prevValue := curval.Interface().(*bool)
				if prevValue != nil && *prevValue == a {
					// no change
					return false, t, nil
				}

				// set it
				curval.Set(reflect.ValueOf(&a))
				return true, t, nil

			case inttype:
				slog.Debug("int")
				var a int
				err := json.Unmarshal([]byte(strvalue), &a)
				if err != nil {
					slog.Warn("Error unmarshaling int", "error", err)
					return false, t, fmt.Errorf("couldn't unmarshal strvalue %v into the field %v: %w", strvalue, key, err)
				}

				// update the time that it was 'last' set
				t.UpdateTimes[key] = updateTime

				prevValue := curval.Interface().(*int)
				if prevValue != nil && *prevValue == a {
					// no change
					return false, t, nil
				}

				// set it
				curval.Set(reflect.ValueOf(&a))

				return true, t, nil
			case float64type:
				slog.Debug("float64")
				var a float64
				err := json.Unmarshal([]byte(strvalue), &a)
				if err != nil {
					slog.Warn("Error unmarshaling float64", "error", err)
					return false, t, fmt.Errorf("couldn't unmarshal strvalue %v into the field %v: %w", strvalue, key, err)
				}

				// update the time that it was 'last' set
				t.UpdateTimes[key] = updateTime

				prevValue := curval.Interface().(*float64)
				if prevValue != nil && *prevValue == a {
					// no change
					return false, t, nil
				}

				// set it
				curval.Set(reflect.ValueOf(&a))

				return true, t, nil
			default:
				return false, t, fmt.Errorf("field %v is an unsupported type %v", key, thistype)
			}

		} else {
			return false, t, fmt.Errorf("there was a problem setting field %v, field is not settable", key)
		}
	}

	// if we made it here, it means that the field isn't found
	return false, t, fmt.Errorf("field %v isn't a valid field for a device", key)
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
	"DEV":     "development-flag", // fake device type for testing purposes
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

	slog.Warn("no valid translation", "value", split[2])
	return ""
}
