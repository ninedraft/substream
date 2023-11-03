package broadcast

import (
	"errors"
	"sync"
	"sync/atomic"
)

var (
	ErrBroadcasterClosed = errors.New("broadcaster is closed")
	ErrSubsriberClosed   = errors.New("subscriber is closed")
)

type Broadcaster[E any] struct {
	isClosed atomic.Bool

	mu       *sync.RWMutex
	notifier *sync.Cond

	value E
	clone func(E) E

	maxClients int
	subs       map[*subscriber]struct{}
}

func New[E any](value E) *Broadcaster[E] {
	return NewClone[E](value, cloneNoop)
}

func cloneNoop[E any](e E) E { return e }

func NewClone[E any](value E, clone func(E) E) *Broadcaster[E] {
	mu := &sync.RWMutex{}
	notifier := sync.NewCond(mu.RLocker())
	return &Broadcaster[E]{
		mu:       mu,
		notifier: notifier,
		clone:    clone,
		value:    value,
		subs:     make(map[*subscriber]struct{}),
	}
}

func (broadcaster *Broadcaster[E]) Broadcast(value E) error {
	return broadcaster.Update(func(E) E { return value })
}

func (broadcaster *Broadcaster[E]) Update(fn func(E) E) error {
	if broadcaster.isClosed.Load() {
		return ErrBroadcasterClosed
	}

	broadcaster.mu.Lock()
	broadcaster.value = fn(broadcaster.value)
	broadcaster.mu.Unlock()

	broadcaster.notifier.Broadcast()

	return nil
}

func (broadcaster *Broadcaster[E]) Listen(fn func(E) error) error {
	if broadcaster.isClosed.Load() {
		return ErrBroadcasterClosed
	}

	sub := new(subscriber)

	broadcaster.addSub(sub)
	defer broadcaster.removeSub(sub)

	broadcaster.mu.RLock()
	defer broadcaster.mu.RUnlock()

	for !sub.isClosed.Load() {
		value := broadcaster.clone(broadcaster.value)

		if err := fn(value); err != nil {
			return err
		}

		broadcaster.notifier.Wait()
	}

	if broadcaster.isClosed.Load() {
		return ErrBroadcasterClosed
	}

	return ErrSubsriberClosed
}

func (streamer *Broadcaster[E]) addSub(sub *subscriber) {
	streamer.mu.Lock()
	defer streamer.mu.Unlock()

	streamer.subs[sub] = struct{}{}
	streamer.maxClients = max(streamer.maxClients, len(streamer.subs))
}

func (streamer *Broadcaster[E]) removeSub(sub *subscriber) {
	streamer.mu.Lock()
	defer streamer.mu.Unlock()

	delete(streamer.subs, sub)
}

func (streamer *Broadcaster[E]) NSubscribers() int {
	streamer.mu.RLock()
	defer streamer.mu.RUnlock()

	return len(streamer.subs)
}

type subscriber struct {
	isClosed atomic.Bool
}

func (s *Broadcaster[E]) Close() error {
	if s.isClosed.Load() {
		return nil
	}

	s.isClosed.Store(true)

	s.mu.Lock()
	for sub := range s.subs {
		sub.isClosed.Store(true)
	}
	s.mu.RUnlock()

	s.notifier.Broadcast()

	return nil
}
