package memorycache

import (
	"log/slog"

	"github.com/byuoitav/event-forwarding-microservice/config"
	"github.com/robfig/cron"
)

// MakeMemoryCache .
func MakeMemoryCache(pushCron string, c config.Cache) (*Memorycache, error) {

	toReturn := Memorycache{
		cacheType: "memory",
		pushCron:  cron.New(),
		name:      c.Name,
	}

	slog.Info("adding the cron push")
	//build our push cron
	er := toReturn.pushCron.AddFunc(pushCron, toReturn.PushAllDevices)
	if er != nil {
		slog.Error("Couldn't add the push all devices cron job to the cache")

	}

	//starting the cron job
	toReturn.pushCron.Start()

	return &toReturn, nil
}
