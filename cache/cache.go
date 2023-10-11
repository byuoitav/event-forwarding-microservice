package cache

import (
	"sync"

	"github.com/byuoitav/common/log"
	"github.com/byuoitav/event-forwarding-microservice/cache/shared"
)

// Caches .
var Caches map[string]shared.Cache
var cachesInit sync.Once

// GetCache .
func GetCache(cacheType string) shared.Cache {
	cachesInit.Do(InitializeCaches)
	log.L.Debugf("Cache type: %s", cacheType)
	toReturn, ok := Caches[cacheType]
	if !ok {
		log.L.Warnf("Cache of type: %s does not exist", cacheType)
		return nil
	}
	return toReturn
}
