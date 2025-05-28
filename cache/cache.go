package cache

import (
	"log/slog"
	"sync"

	"github.com/byuoitav/event-forwarding-microservice/cache/shared"
)

// Caches .
var Caches map[string]shared.Cache
var cachesInit sync.Once

// GetCache .
func GetCache(cacheType string) shared.Cache {
	cachesInit.Do(InitializeCaches)
	slog.Info("Cache type", "type", cacheType)
	toReturn, ok := Caches[cacheType]
	if !ok {
		slog.Warn("Cache of type does not exist", "type", cacheType)
		return nil
	}
	return toReturn
}
