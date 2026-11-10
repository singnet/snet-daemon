// Package lifecycle tracks cancellable work owned by a daemon instance.
package lifecycle

import (
	"context"
	"sync"
)

type Group struct {
	ctx      context.Context
	cancel   context.CancelFunc
	mu       sync.Mutex
	stopping bool
	wg       sync.WaitGroup
}

func New() *Group {
	ctx, cancel := context.WithCancel(context.Background())
	return &Group{ctx: ctx, cancel: cancel}
}

func (g *Group) Context() context.Context { return g.ctx }

// Begin preserves caller values and cancellation while adding daemon cancellation.
func (g *Group) Begin(parent context.Context) (context.Context, func(), bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.stopping {
		return nil, nil, false
	}
	g.wg.Add(1)
	ctx, cancel := context.WithCancel(parent)
	stop := context.AfterFunc(g.ctx, cancel)
	return ctx, func() { stop(); cancel(); g.wg.Done() }, true
}

func (g *Group) Go(work func(context.Context)) bool {
	if g == nil {
		go work(context.Background())
		return true
	}
	ctx, done, ok := g.Begin(g.ctx)
	if !ok {
		return false
	}
	go func() { defer done(); work(ctx) }()
	return true
}

// StopAccepting must precede Wait, so no work can be added after the wait starts.
func (g *Group) StopAccepting() { g.mu.Lock(); g.stopping = true; g.mu.Unlock() }
func (g *Group) Cancel()        { g.StopAccepting(); g.cancel() }
func (g *Group) Wait()          { g.wg.Wait() }
