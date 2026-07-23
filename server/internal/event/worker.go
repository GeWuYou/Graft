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
		}
	}
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
