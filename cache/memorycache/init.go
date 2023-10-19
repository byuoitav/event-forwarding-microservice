package memorycache

import (
	"github.com/byuoitav/common/log"
	"github.com/byuoitav/common/nerr"
	"github.com/byuoitav/event-forwarding-microservice/config"
	"github.com/robfig/cron"
)

// MakeMemoryCache .
func MakeMemoryCache(pushCron string, c config.Cache) (*Memorycache, *nerr.E) {

	toReturn := Memorycache{
		cacheType: "memory",
		pushCron:  cron.New(),
		name:      c.Name,
	}

	log.L.Infof("adding the cron push")
	//build our push cron
	er := toReturn.pushCron.AddFunc(pushCron, toReturn.PushAllDevices)
	if er != nil {
		log.L.Errorf("Couldn't add the push all devices cron job to the cache")

	}

	//starting the cron job
	toReturn.pushCron.Start()

	return &toReturn, nil
}
