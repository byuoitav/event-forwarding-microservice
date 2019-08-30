package shared

import (
	"testing"
	"time"

	"github.com/byuoitav/common/log"
	sd "github.com/byuoitav/common/state/statedefinition"
	"github.com/stretchr/testify/assert"
)

func TestSetDeviceField(t *testing.T) {

	base := sd.StaticDevice{}

	//string field tests
	update, new, err := SetDeviceField("ID", "This is a test", time.Now(), base)
	if err != nil {
		t.Error(err.Error())
		t.FailNow()
	}
	assert.Equal(t, new.DeviceID, "This is a test")

	update, new, err = SetDeviceField("ID", "This is a test", time.Now(), new)
	if err != nil {
		t.Error(err.Error())
		t.FailNow()
	}
	assert.Equal(t, false, update)

	update, new, err = SetDeviceField("ID", "Shoudn't make it in.", time.Now().Add(-1*time.Hour), new)
	if err != nil {
		t.Error(err.Error())
		t.FailNow()
	}
	assert.Equal(t, false, update)
	assert.NotEqual(t, new.DeviceID, "Shouldn't make it in.")

	update, new, err = SetDeviceField("input", "abc", time.Now().Add(-1*time.Hour), new)
	if err != nil {
		t.Error(err.Error())
		t.FailNow()
	}
	assert.Equal(t, true, update)
	assert.Equal(t, new.Input, "abc")

	//Note that if we pass in supported non-string primatives to a string field, the field will be filled with the string represntation of them
	update, new, err = SetDeviceField("input", true, time.Now(), new)
	if err != nil {
		t.Error(err.Error())
		t.FailNow()
	}
	assert.Equal(t, true, update)
	assert.Equal(t, new.Input, "true")

	update, new, err = SetDeviceField("input", 123, time.Now(), new)
	if err != nil {
		t.Error(err.Error())
		t.FailNow()
	}
	assert.Equal(t, true, update)
	assert.Equal(t, new.Input, "123")

	//non supported primatives and structs will fail, even in a string field.
	update, new, err = SetDeviceField("input", byte(0x1), time.Now(), new)
	assert.NotNil(t, err)
	assert.Equal(t, false, update)
	assert.Equal(t, new.Input, "123")

	var testBadStruct struct {
		A string
		B int
	}
	testBadStruct.A = "abc"
	testBadStruct.B = 123

	update, new, err = SetDeviceField("input", testBadStruct, time.Now(), new)
	assert.NotNil(t, err)
	assert.Equal(t, false, update)
	assert.Equal(t, new.Input, "123")

	//integer field tests

	update, new, err = SetDeviceField("volume", "12", time.Now(), new)
	if err != nil {
		log.L.Warnf("%s", err.Stack)
		t.Error(err.Error())
		t.FailNow()
	}
	assert.Equal(t, true, update)
	assert.Equal(t, *new.Volume, 12)

	update, new, err = SetDeviceField("volume", "12", time.Now(), new)
	if err != nil {
		log.L.Warnf("%s", err.Stack)
		t.Error(err.Error())
		t.FailNow()
	}
	assert.Equal(t, false, update)
	assert.Equal(t, *new.Volume, 12)

	update, new, err = SetDeviceField("volume", 100, time.Now(), new)
	if err != nil {
		log.L.Warnf("%s", err.Stack)
		t.Error(err.Error())
		t.FailNow()
	}
	assert.Equal(t, true, update)
	assert.Equal(t, *new.Volume, 100)

	update, new, err = SetDeviceField("volume", "50", time.Now(), new)
	if err != nil {
		log.L.Warnf("%s", err.Stack)
		t.Error(err.Error())
		t.FailNow()
	}
	assert.Equal(t, true, update)
	assert.Equal(t, *new.Volume, 50)

	update, new, err = SetDeviceField("volume", "abc", time.Now(), new)
	assert.NotNil(t, err)
	assert.Equal(t, false, update)
	assert.Equal(t, *new.Volume, 50)

	//bool field tests
	update, new, err = SetDeviceField("blanked", "true", time.Now(), new)
	if err != nil {
		log.L.Warnf("%s", err.Stack)
		t.Error(err.Error())
		t.FailNow()
	}
	assert.Equal(t, true, update)
	assert.Equal(t, *new.Blanked, true)

	update, new, err = SetDeviceField("blanked", "false", time.Now(), new)
	if err != nil {
		log.L.Warnf("%s", err.Stack)
		t.Error(err.Error())
		t.FailNow()
	}
	assert.Equal(t, true, update)
	assert.Equal(t, *new.Blanked, false)

	update, new, err = SetDeviceField("blanked", "false", time.Now(), new)
	if err != nil {
		log.L.Warnf("%s", err.Stack)
		t.Error(err.Error())
		t.FailNow()
	}
	assert.Equal(t, false, update)
	assert.Equal(t, *new.Blanked, false)

	update, new, err = SetDeviceField("blanked", "abc", time.Now(), new)
	assert.NotNil(t, err)
	assert.Equal(t, false, update)
	assert.Equal(t, *new.Blanked, false)

	update, new, err = SetDeviceField("blanked", "123", time.Now(), new)
	assert.NotNil(t, err)
	assert.Equal(t, false, update)
	assert.Equal(t, *new.Blanked, false)

	update, new, err = SetDeviceField("blanked", 123, time.Now(), new)
	assert.NotNil(t, err)
	assert.Equal(t, false, update)
	assert.Equal(t, *new.Blanked, false)

	//time field tests
	pretime := time.Now()
	time.Sleep(5)
	curtime := time.Now()

	update, new, err = SetDeviceField("last-state-received", curtime, curtime, new)
	if err != nil {
		log.L.Warnf("%s", err.Stack)
		t.Error(err.Error())
		t.FailNow()
	}
	assert.Equal(t, true, update)
	assert.Equal(t, new.LastStateReceived.Format(time.RFC3339Nano), curtime.Format(time.RFC3339Nano))

	update, new, err = SetDeviceField("last-state-received", pretime, pretime, new)
	if err != nil {
		log.L.Warnf("%s", err.Stack)
		t.Error(err.Error())
		t.FailNow()
	}
	assert.Equal(t, false, update)
	assert.Equal(t, new.LastStateReceived.Format(time.RFC3339Nano), curtime.Format(time.RFC3339Nano))
	assert.NotEqual(t, new.LastStateReceived.Format(time.RFC3339Nano), pretime.Format(time.RFC3339Nano))

	update, new, err = SetDeviceField("last-state-received", "abc", time.Now(), new)
	assert.NotNil(t, err)
	assert.Equal(t, false, update)
	assert.Equal(t, new.LastStateReceived.Format(time.RFC3339Nano), curtime.Format(time.RFC3339Nano))
	assert.NotEqual(t, new.LastStateReceived.Format(time.RFC3339Nano), pretime.Format(time.RFC3339Nano))

	//alert field tests

	Alert := sd.Alert{
		Alerting:  true,
		AlertSent: time.Now(),
		Message:   "ABC",
	}

	NewAlert := sd.Alert{
		Alerting:  true,
		AlertSent: time.Now(),
		Message:   "DEF",
	}

	update, new, err = SetDeviceField("alerts.TestAlert", Alert, time.Now(), new)
	if err != nil {
		t.Error(err.Error())
		t.FailNow()
	}
	assert.Equal(t, new.Alerts["TestAlert"].Message, "ABC")
	assert.True(t, update)

	update, new, err = SetDeviceField("alerts.TestAlert", Alert, time.Now(), new)
	if err != nil {
		t.Error(err.Error())
		t.FailNow()
	}
	assert.Equal(t, new.Alerts["TestAlert"].Message, "ABC")
	assert.True(t, update)

	update, new, err = SetDeviceField("alerts.TestAlert", NewAlert, time.Now(), new)
	if err != nil {
		t.Error(err.Error())
		t.FailNow()
	}
	assert.Equal(t, new.Alerts["TestAlert"].Message, "DEF")
	assert.True(t, update)

	update, new, err = SetDeviceField("alerts.TestAlert2", Alert, time.Now(), new)
	if err != nil {
		t.Error(err.Error())
		t.FailNow()
	}
	assert.Equal(t, new.Alerts["TestAlert"].Message, "DEF")
	assert.Equal(t, new.Alerts["TestAlert2"].Message, "ABC")
}

var UpdateRes bool

func BenchmarkUpdateDevice(b *testing.B) {
	log.SetLevel("fatal")
	base := sd.StaticDevice{}
	var update bool

	for n := 0; n < b.N; n++ {
		update, base, _ = SetDeviceField("DeviceID", "This is a test", time.Now(), base)
	}
	UpdateRes = update
}

func BenchmarkCompareDevice(b *testing.B) {
	log.SetLevel("fatal")
	base := sd.StaticDevice{}
	new := sd.StaticDevice{DeviceID: "This is a test"}

	var update bool

	for n := 0; n < b.N; n++ {
		_, _, update, _ = sd.CompareDevices(base, new)
	}

	UpdateRes = update
}
