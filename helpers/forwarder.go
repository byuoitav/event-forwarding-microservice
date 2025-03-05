package helpers

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	//"github.com/byuoitav/common/v2/events"
	"github.com/byuoitav/event-forwarding-microservice/cache"
	"github.com/byuoitav/event-forwarding-microservice/cache/shared"
	"github.com/byuoitav/event-forwarding-microservice/events"
)

// A ForwardManager manages events to efficiently forward them
type ForwardManager struct {
	Workers     int
	EventStream chan events.Event

	wg  *sync.WaitGroup
	ctx context.Context // the context passed in when Start() was called
}

var (
	fm   *ForwardManager
	once = sync.Once{}
)

// GetForwardManager .
func GetForwardManager() *ForwardManager {
	once.Do(func() {
		fm = &ForwardManager{
			Workers:     10,
			EventStream: make(chan events.Event, 10000),
		}
	})
	slog.Debug("GetForwardManager()", "ForwardManager Info", fmt.Sprintf("%v", fm))
	return fm
}

// Start initializes the forward manager
func (f *ForwardManager) Start(ctx context.Context) error {
	f.ctx = ctx
	f.wg = &sync.WaitGroup{}

	if f.Workers <= 0 {
		f.Workers = 1
	}

	//prev, _ := log.GetLevel()
	//log.SetLevel("info")

	slog.Info(fmt.Sprintf("Starting forward manager with %d workers", f.Workers))

	ctx, cancel := context.WithCancel(ctx)
	defer cancel() // clean up resources if the forward manager ever exits

	//log.SetLevel(prev)

	for i := 0; i < f.Workers; i++ {
		f.wg.Add(1)

		go func(index int) {
			defer f.wg.Done()
			defer slog.Info(fmt.Sprintf("Closed forward manager worker %d", index))
			slog.Debug(fmt.Sprintf("Starting worker %v", index))
			for {
				select {
				case <-ctx.Done():
					return
				case event, ok := <-f.EventStream:
					slog.Debug("DEBUG", "Event in ForwardManagers Start", fmt.Sprintf("%v", event))
					if !ok {
						slog.Warn("forward manager event stream closed")
						return
					}
					slog.Debug("Storing and Forwarding Event", "EventCache", f.EventCache)
					shared.StoreAndForwardEvent(event)
					}
					/*
						if len(f.EventCache) > 0 {
							slog.Debug("Storing and Forwarding Event", "Event", fmt.Sprintf(f.EventCache))
							//get the cache and submit for persistence
							cache.GetCache(f.EventCache).StoreAndForwardEvent(event)
						}*/
				}
			}
		}(i)
	}

	f.wg.Wait()
	slog.Info("forward manager stopped.")

	return nil
}
