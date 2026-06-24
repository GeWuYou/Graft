package realtime

import (
	"sync"
	"testing"
)

func TestMemoryHubUnsubscribeCanRaceWithPublishWithoutPanicking(_ *testing.T) {
	hub := NewHub()
	events, unsubscribe := hub.Subscribe("topic.test")

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			hub.Publish("topic.test", i)
		}
	}()

	go func() {
		defer wg.Done()
		unsubscribe()
	}()

	wg.Wait()

	select {
	case <-events:
	default:
	}
}
