package cache

import (
	"fmt"
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
	debugLog := fmt.Sprintf("Cache type: %s", cacheType)
	slog.Debug(debugLog)
	toReturn, ok := Caches[cacheType]
	if !ok {
		warnLog := fmt.Sprintf("Cache of type: %s does not exist", cacheType)
		slog.Warn(warnLog)
		return nil
	}
	return toReturn
}
