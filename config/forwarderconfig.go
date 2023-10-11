package config

// CONST
const (
	//Event Types

	ALL   = "all"
	DELTA = "delta"

	//Data types

	DEVICE = "device"
	ROOM   = "room"
	EVENT  = "event"

	//Cache Types

	LEGACY  = "legacy"
	DEFAULT = "default"

	//Forwarder Types

	ELKSTATIC     = "elkstatic"
	ELKTIMESERIES = "elktimeseries"
	COUCH         = "couch"
	WEBSOCKET     = "websocket"
	HUMIO         = "humio"

	//Rotation Intervals

	WEEKLY   = "weekly"
	DAILY    = "daily"
	MONTHLY  = "monthly"
	YEARLY   = "yearly"
	NOROTATE = "norotate"
)

//Forwarder .
type Forwarder struct {
	Name string `json:"name"`

	//SupportedValues:
	//elkstatic, elktimeseries, couch, humio
	Type string `json:"type"`

	//Supported Values:
	//delta, all
	EventType string `json:"event-type"`

	//Interval in seconds between change pushes
	Interval int `json:"interval"`

	//Supported Values:
	//device, event
	DataType string `json:"data-type"`

	//Supported Values;
	//legacy, default
	CacheName string `json:"cache-name"`

	Couch CouchForwarder `json:"couch"`
	Elk   ElkForwarder   `json:"elk"`
	Humio HumioForwarder `json:"humio"`
}

//CouchForwader .
type CouchForwarder struct {
	URL          string `json:"url"`
	DatabaseName string `json:"database-name"`
}

//ElkForwarder .
type ElkForwarder struct {
	URL          string `json:"url"`
	IndexPattern string `json:"index-pattern"`
	Upsert       bool   `json:"upsert"`

	//Supported Values:
	//daily, weekly, monthly, yearly
	IndexRotationInterval string `json:"index-rotation-interval"`
}

type HumioForwarder struct {
	Interval   int `json:"update-interval"`
	BufferSize int `json:"buffer-size"`
}
