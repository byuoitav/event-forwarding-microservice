package helpers

import (
	"context"
	"log/slog"
	"sync"

	"github.com/byuoitav/event-forwarding-microservice/cache"
	"github.com/byuoitav/event-forwarding-microservice/events"
)

// A ForwardManager manages events to efficiently forward them
type ForwardManager struct {
	Workers     int
	EventStream chan events.Event
	EventCache  string

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
			EventCache:  "default",
		}
	})

	return fm
}

// Start initializes the forward manager
func (f *ForwardManager) Start(ctx context.Context) error {
	f.ctx = ctx
	f.wg = &sync.WaitGroup{}

	if f.Workers <= 0 {
		f.Workers = 1
	}

	slog.Info("Starting forward manager", "workers", f.Workers)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel() // clean up resources if the forward manager ever exits

	for i := 0; i < f.Workers; i++ {
		f.wg.Add(1)

		go func(index int) {
			defer f.wg.Done()
			slog.Info("Started forward manager worker", "worker", index)
			defer slog.Info("Closed forward manager worker", "worker", index)

			for {
				select {
				case <-ctx.Done():
					return
				case event, ok := <-f.EventStream:
					if !ok {
						slog.Warn("forward manager event stream closed")
						return
					}

					if len(f.EventCache) > 0 {
						//get the cache and submit for persistence
						cache.GetCache(f.EventCache).StoreAndForwardEvent(event)
					}
				}
			}
		}(i)
	}

	f.wg.Wait()
	slog.Info("forward manager stopped.")

	return nil
}
