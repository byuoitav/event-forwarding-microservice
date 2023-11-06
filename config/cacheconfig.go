package config

//Cache .
type Cache struct {
	Name string `json:"name"`

	//Legacy or Default
	CacheType string `json:"cache-type"`

	CouchInfo CouchCache `json:"couch-cache"`
	ELKinfo   ElkCache   `json:"elk-cache"`
	RedisInfo RedisCache `json:"redis-cache"`
}

//CouchCache .
type CouchCache struct {
	DatabaseName string `json:"database-name"`
	URL          string `json:"url"`
}

//ElkCache .
type ElkCache struct {
	DeviceIndex string `json:"device-index"`
	RoomIndex   string `json:"room-index"`
	URL         string `json:"url"`
}

//RedisCache .
type RedisCache struct {
	DevDatabase  int    `json:"device-database"`
	RoomDatabase int    `json:"room-database"`
	Password     string `json:"password"`
	URL          string `json:"url"`
}
