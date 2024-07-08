package wrapper

import (
	"context"

	"go.uber.org/zap"
)

type EventQueue struct {
	logger *zap.Logger

	ch chan func()

	ctx         context.Context
	ctxCancelFn context.CancelFunc
}

func NewEventQueue(logger *zap.Logger, workerCount int, queueSize int) *EventQueue {
	b := &EventQueue{
		logger: logger,

		ch: make(chan func(), queueSize),
	}
	b.ctx, b.ctxCancelFn = context.WithCancel(context.Background())

	// Start a fixed number of workers.
	for i := 0; i < workerCount; i++ {
		go func() {
			for {
				select {
				case <-b.ctx.Done():
					return
				case fn := <-b.ch:
					fn()
				}
			}
		}()
	}

	return b
}

func (b *EventQueue) Queue(fn func()) {
	select {
	case b.ch <- fn:
		// Event queued successfully.
	default:
		b.logger.Warn("Event queue full, events may be lost")
	}
}

func (b *EventQueue) Stop() {
	b.ctxCancelFn()
}
