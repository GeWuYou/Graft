// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package realtimeauth

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type fixedClock struct {
	now time.Time
}

func (c *fixedClock) Now() time.Time {
	return c.now
}

func TestMemoryServiceConsumesTicketOnce(t *testing.T) {
	t.Parallel()

	service := NewMemoryServiceWithClock(&fixedClock{now: time.Date(2026, 6, 19, 10, 0, 0, 0, time.UTC)})
	issued, err := service.Issue(context.Background(), IssueRequest{
		UserID:       7,
		ResourceType: "container",
		ResourceID:   "abc123",
		Scope:        "container.shell",
		Command:      "sh",
		Cols:         120,
		Rows:         32,
	})
	if err != nil {
		t.Fatalf("issue ticket: %v", err)
	}

	consumed, err := service.Consume(context.Background(), ConsumeRequest{
		Ticket:       issued.Ticket,
		ResourceType: "container",
		ResourceID:   "abc123",
		Scope:        "container.shell",
	})
	if err != nil {
		t.Fatalf("consume ticket: %v", err)
	}
	if consumed.UserID != 7 || consumed.SessionID == "" {
		t.Fatalf("unexpected consumed ticket: %#v", consumed)
	}

	_, err = service.Consume(context.Background(), ConsumeRequest{
		Ticket:       issued.Ticket,
		ResourceType: "container",
		ResourceID:   "abc123",
		Scope:        "container.shell",
	})
	if !errors.Is(err, ErrUsedTicket) {
		t.Fatalf("expected used ticket error, got %v", err)
	}
}

func TestMemoryServiceRejectsExpiredTicket(t *testing.T) {
	t.Parallel()

	clock := &fixedClock{now: time.Date(2026, 6, 19, 10, 0, 0, 0, time.UTC)}
	service := NewMemoryServiceWithClock(clock)
	issued, err := service.Issue(context.Background(), IssueRequest{
		UserID:       7,
		ResourceType: "container",
		ResourceID:   "abc123",
		Scope:        "container.shell",
		TTL:          time.Second,
	})
	if err != nil {
		t.Fatalf("issue ticket: %v", err)
	}

	clock.now = clock.now.Add(2 * time.Second)
	_, err = service.Consume(context.Background(), ConsumeRequest{
		Ticket:       issued.Ticket,
		ResourceType: "container",
		ResourceID:   "abc123",
		Scope:        "container.shell",
	})
	if !errors.Is(err, ErrExpiredTicket) {
		t.Fatalf("expected expired ticket, got %v", err)
	}
}

func TestMemoryServiceRejectsWrongResource(t *testing.T) {
	t.Parallel()

	service := NewMemoryServiceWithClock(&fixedClock{now: time.Date(2026, 6, 19, 10, 0, 0, 0, time.UTC)})
	issued, err := service.Issue(context.Background(), IssueRequest{
		UserID:       7,
		ResourceType: "container",
		ResourceID:   "abc123",
		Scope:        "container.shell",
	})
	if err != nil {
		t.Fatalf("issue ticket: %v", err)
	}

	_, err = service.Consume(context.Background(), ConsumeRequest{
		Ticket:       issued.Ticket,
		ResourceType: "container",
		ResourceID:   "other",
		Scope:        "container.shell",
	})
	if !errors.Is(err, ErrResourceMismatch) {
		t.Fatalf("expected resource mismatch, got %v", err)
	}
}

func TestMemoryServiceAtomicConsume(t *testing.T) {
	t.Parallel()

	service := NewMemoryServiceWithClock(&fixedClock{now: time.Date(2026, 6, 19, 10, 0, 0, 0, time.UTC)})
	issued, err := service.Issue(context.Background(), IssueRequest{
		UserID:       7,
		ResourceType: "container",
		ResourceID:   "abc123",
		Scope:        "container.shell",
	})
	if err != nil {
		t.Fatalf("issue ticket: %v", err)
	}

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, consumeErr := service.Consume(context.Background(), ConsumeRequest{
				Ticket:       issued.Ticket,
				ResourceType: "container",
				ResourceID:   "abc123",
				Scope:        "container.shell",
			})
			errs <- consumeErr
		}()
	}
	wg.Wait()
	close(errs)

	successes := 0
	usedFailures := 0
	for err := range errs {
		if err == nil {
			successes++
			continue
		}
		if errors.Is(err, ErrUsedTicket) {
			usedFailures++
			continue
		}
		t.Fatalf("unexpected consume error: %v", err)
	}
	if successes != 1 || usedFailures != 1 {
		t.Fatalf("expected one success and one used failure, got success=%d used=%d", successes, usedFailures)
	}
}

func TestValidateOrigin(t *testing.T) {
	t.Parallel()

	if err := ValidateOrigin("https://console.example.com", []string{"https://console.example.com"}); err != nil {
		t.Fatalf("expected allowed origin, got %v", err)
	}
	if err := ValidateOrigin("https://evil.example.com", []string{"https://console.example.com"}); !errors.Is(err, ErrOriginDenied) {
		t.Fatalf("expected origin denied, got %v", err)
	}
}
