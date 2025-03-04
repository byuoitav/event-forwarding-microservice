package cache

import (
	"fmt"
	"log/slog"

	//"github.com/byuoitav/common/log"
	//"github.com/byuoitav/common/nerr"
	"github.com/byuoitav/event-forwarding-microservice/cache/memorycache"
	"github.com/byuoitav/event-forwarding-microservice/cache/rediscache"
	"github.com/byuoitav/event-forwarding-microservice/cache/shared"
	"github.com/byuoitav/event-forwarding-microservice/config"
	customerror "github.com/byuoitav/event-forwarding-microservice/error"
)

const maxSize = 10000

const pushCron = "0 0 0 * * *"

func InitializeCaches() {
	slog.Info("Initializing Caches")
	Caches = make(map[string]shared.Cache)

	c := config.GetConfig()
	for _, i := range c.Caches {
		infoLog := fmt.Sprintf("Initializing cache %v", i.Name)
		slog.Info(infoLog)
		cache, err := makeCache(i)
		if err != nil {
			errorLog := fmt.Sprintf("Couldn't make cache: %v", err.Error())
			slog.Error(errorLog)
		}
		Caches[i.Name] = cache
		initLog := fmt.Sprintf("Cache %v initialized with type %v. ", i.Name, i.CacheType)
		slog.Info(initLog)
	}

	slog.Info("Cache Check Done.")
}

func makeCache(config config.Cache) (shared.Cache, error) {
	switch config.CacheType {
	case "memory":
		return memorycache.MakeMemoryCache(pushCron, config)
	case "redis":
		return rediscache.MakeRedisCache(pushCron, config)
	}
	cErr := &customerror.StandardError{
		Message: fmt.Sprintf("Unkown cache type %v", config.CacheType),
	}
	return nil, cErr

}
