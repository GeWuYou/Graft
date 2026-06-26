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

func TestMemoryHubTopicObserverTracksActiveSubscriberTransitions(t *testing.T) {
	hub := NewHub()
	memoryHub, ok := hub.(*memoryHub)
	if !ok {
		t.Fatal("expected memory hub implementation")
	}

	var activeCalls []string
	var inactiveCalls []string
	unregister, err := memoryHub.RegisterTopicObserver("topic.test", func(topic string) {
		activeCalls = append(activeCalls, topic)
	}, func(topic string) {
		inactiveCalls = append(inactiveCalls, topic)
	})
	if err != nil {
		t.Fatalf("register topic observer: %v", err)
	}
	defer unregister()

	_, unsubscribeFirst := memoryHub.Subscribe("topic.test")
	_, unsubscribeSecond := memoryHub.Subscribe("topic.test")
	if len(activeCalls) != 1 || activeCalls[0] != "topic.test" {
		t.Fatalf("expected one active transition, got %#v", activeCalls)
	}

	unsubscribeFirst()
	if len(inactiveCalls) != 0 {
		t.Fatalf("expected no inactive transition while one subscriber remains, got %#v", inactiveCalls)
	}

	unsubscribeSecond()
	if len(inactiveCalls) != 1 || inactiveCalls[0] != "topic.test" {
		t.Fatalf("expected one inactive transition, got %#v", inactiveCalls)
	}
}
