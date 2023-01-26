package helper

import (
	"context"
)

type Gate struct {
	ctx context.Context
	ch  chan struct{}
}

func NewGate(ctx context.Context, max int) *Gate {
	ch := make(chan struct{}, max)
	for i := 0; i < max; i++ {
		ch <- struct{}{}
	}
	return &Gate{ctx, ch}
}

func (g *Gate) Enter() (entered bool) {
	if cap(g.ch) > 0 {
		select {
		case <-g.ch:
			return true
		case <-g.ctx.Done():
			return false
		}
	} else {
		// unlimited, just check if ctx is done
		select {
		case <-g.ctx.Done():
			return false
		default:
			return true
		}
	}
}

func (g *Gate) Leave() {
	if cap(g.ch) > 0 {
		go func() {
			g.ch <- struct{}{}
		}()
	}
}
