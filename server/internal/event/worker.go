package event

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"time"

	"go.uber.org/zap"
)

func (d *Dispatcher) runWorker(ctx context.Context) {
	defer d.workers.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-d.queue:
			d.deliver(ctx, event)
		case delivery := <-d.durable:
			d.deliverDurable(ctx, delivery)
		}
	}
}

func (d *Dispatcher) runOutboxPoller(ctx context.Context) {
	defer d.poller.Done()
	ticker := time.NewTicker(d.options.OutboxPoll)
	defer ticker.Stop()
	for {
		if !d.claimOutbox(ctx) {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (d *Dispatcher) claimOutbox(ctx context.Context) bool {
	if d.store == nil {
		return false
	}
	claimed, err := d.store.Claim(ctx, d.ownerID, time.Now().UTC(), d.options.LeaseDuration, d.options.BufferSize)
	if err != nil {
		if ctx.Err() == nil {
			d.logger.Error("claim durable event deliveries", zap.Error(err))
		}
		return ctx.Err() == nil
	}
	for _, delivery := range claimed {
		d.work.Add(1)
		select {
		case <-ctx.Done():
			d.work.Done()
			return false
		case d.durable <- delivery:
		}
	}
	return true
}

func (d *Dispatcher) deliver(ctx context.Context, event Event) {
	defer d.work.Done()
	for attempt := 1; attempt <= d.options.MaxAttempts; attempt++ {
		err := d.handle(ctx, event)
		if err == nil {
			return
		}
		if errors.Is(err, ErrNoHandlers) || attempt == d.options.MaxAttempts {
			d.logger.Error("event delivery failed",
				zap.String("event_id", event.ID),
				zap.String("event_type", string(event.Type)),
				zap.String("source", event.Source),
				zap.Int("attempt", attempt),
				zap.Error(err),
			)
			return
		}
		timer := time.NewTimer(time.Duration(attempt) * d.options.RetryDelay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (d *Dispatcher) deliverDurable(ctx context.Context, delivery ClaimedDelivery) {
	defer d.work.Done()
	handler := d.handlerFor(delivery.Event.Type, delivery.ConsumerID)
	if handler == nil {
		_ = d.store.Retry(ctx, delivery, time.Now().UTC().Add(d.options.RetryDelay), fmt.Errorf("%w: consumer %s", ErrNoHandlers, delivery.ConsumerID))
		return
	}
	handlerCtx, cancel := context.WithTimeout(ctx, d.options.HandlerTimeout)
	err := d.invoke(handlerCtx, handler, delivery.Event)
	cancel()
	if err == nil {
		if completeErr := d.store.Complete(ctx, delivery); completeErr != nil {
			d.logger.Error("complete durable event delivery", zap.String("event_id", delivery.Event.ID), zap.String("consumer", delivery.ConsumerID), zap.Error(completeErr))
		}
		return
	}
	retryAt := time.Now().UTC().Add(time.Duration(delivery.Attempt) * d.options.RetryDelay)
	if retryErr := d.store.Retry(ctx, delivery, retryAt, err); retryErr != nil {
		d.logger.Error("reschedule durable event delivery", zap.String("event_id", delivery.Event.ID), zap.String("consumer", delivery.ConsumerID), zap.Error(retryErr))
		return
	}
	d.logger.Warn("durable event delivery failed and was rescheduled",
		zap.String("event_id", delivery.Event.ID), zap.String("event_type", string(delivery.Event.Type)),
		zap.String("consumer", delivery.ConsumerID), zap.Int("attempt", delivery.Attempt), zap.Error(err))
}

func (d *Dispatcher) handle(ctx context.Context, event Event) error {
	handlers := d.handlersFor(event.Type)
	if len(handlers) == 0 {
		return fmt.Errorf("%w: %s", ErrNoHandlers, event.Type)
	}
	var result error
	for _, handler := range handlers {
		handlerCtx, cancel := context.WithTimeout(ctx, d.options.HandlerTimeout)
		err := d.invoke(handlerCtx, handler, event)
		cancel()
		if err != nil {
			result = errors.Join(result, err)
		}
	}
	return result
}

func (d *Dispatcher) invoke(ctx context.Context, handler Handler, event Event) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("event handler %s panicked: %v", handler.ID(), recovered)
			d.logger.Error("event handler panicked",
				zap.String("handler", handler.ID()),
				zap.String("event_id", event.ID),
				zap.String("event_type", string(event.Type)),
				zap.Any("panic", recovered),
				zap.String("stacktrace", string(debug.Stack())),
			)
		}
	}()
	if err := handler.Handle(ctx, event); err != nil {
		return fmt.Errorf("handle event with %s: %w", handler.ID(), err)
	}
	return nil
}
