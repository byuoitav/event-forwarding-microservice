package helpers

import (
	"context"
	"sync"

	"github.com/byuoitav/common/log"
	"github.com/byuoitav/common/v2/events"
	"github.com/byuoitav/event-forwarding-microservice/shared"
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

	return fm
}

// Start initializes the forward manager
func (f *ForwardManager) Start(ctx context.Context) error {
	f.ctx = ctx
	f.wg = &sync.WaitGroup{}

	if f.Workers <= 0 {
		f.Workers = 1
	}

	prev, _ := log.GetLevel()
	log.SetLevel("info")

	log.L.Infof("Starting forward manager with %d workers", f.Workers)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel() // clean up resources if the forward manager ever exits

	log.SetLevel(prev)

	for i := 0; i < f.Workers; i++ {
		f.wg.Add(1)

		go func(index int) {
			defer f.wg.Done()
			defer log.L.Infof("Closed forward manager worker %d", index)

			for {
				select {
				case <-ctx.Done():
					return
				case event, ok := <-f.EventStream:
					if !ok {
						log.L.Warnf("forward manager event stream closed")
						return
					}
					shared.ForwardAndStoreEvent(event)
				}
			}
		}(i)
	}

	f.wg.Wait()
	log.L.Infof("forward manager stopped.")

	return nil
}
