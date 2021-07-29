package status

import (
	"context"
	"errors"
	"time"
)

var ErrStalled = errors.New("stalled")

type Checker interface {
	Ping()
	Check() error
	Start() error
	Close()
}

func NewChecker(ctx context.Context, cancel context.CancelFunc, interval time.Duration) Checker {
	return &checker{
		ctx:       ctx,
		cancel:    cancel,
		heartbeat: make(chan struct{}, 1),
		interval:  interval}
}

type checker struct {
	ctx       context.Context
	cancel    context.CancelFunc
	heartbeat chan struct{}
	interval  time.Duration
}

func (c *checker) Ping() {
	//clear channel if already pinged
	select {
	case <-c.heartbeat:
	default:
	}

	c.heartbeat <- struct{}{}
}

func (c *checker) Check() error {
	select {
	case <-c.ctx.Done():
		return ErrStalled
	default:
		return context.Canceled
	}
}

func (c *checker) Start() error {
	go func() {
		for {
			select {
			case <-c.heartbeat:
			case <-c.ctx.Done():
				return
			default:
				c.cancel()
				return
			}

			select {
			case <-c.ctx.Done():
				return
			case <-time.After(c.interval):
			}
		}
	}()
	return nil
}

func (c *checker) Close() {
	c.cancel()
}
