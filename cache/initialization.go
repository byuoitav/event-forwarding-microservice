package cache

import (
	"github.com/byuoitav/common/log"
	"github.com/byuoitav/common/nerr"
	"github.com/byuoitav/event-forwarding-microservice/cache/memorycache"
	"github.com/byuoitav/event-forwarding-microservice/cache/rediscache"
	"github.com/byuoitav/event-forwarding-microservice/cache/shared"
	"github.com/byuoitav/event-forwarding-microservice/config"
)

const maxSize = 10000

const pushCron = "0 0 0 * * *"

func InitializeCaches() {
	log.L.Infof("Initializing Caches")
	Caches = make(map[string]shared.Cache)

	c := config.GetConfig()
	for _, i := range c.Caches {
		log.L.Infof("Initializing cache %v", i.Name)
		cache, err := makeCache(i)
		if err != nil {
			log.L.Fatalf("Couldn't make cache: %v", err.Error())
		}
		Caches[i.Name] = cache
		log.L.Infof("Cache %v initialized with type %v. ", i.Name, i.CacheType)
	}

	log.L.Infof("Cache Check Done.")
}

func makeCache(config config.Cache) (shared.Cache, *nerr.E) {
	switch config.CacheType {
	case "memory":
		return memorycache.MakeMemoryCache(pushCron, config)
	case "redis":
		return rediscache.MakeRedisCache(pushCron, config)
	}

	return nil, nerr.Create("Unkown cache type %v", config.CacheType)

}
