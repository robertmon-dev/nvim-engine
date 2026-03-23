package provider

import "sync/atomic"

type Dispatcher struct {
	providers []Provider
	current   uint64
}

func NewDispatcher(providers ...Provider) *Dispatcher {
	var active []Provider

	for _, p := range providers {
		if p.IsReady() {
			active = append(active, p)
		}
	}

	return &Dispatcher{
		providers: active,
	}
}

func (r *Dispatcher) getNext() Provider {
	if len(r.providers) == 0 {
		return nil
	}

	idx := atomic.AddUint64(&r.current, 1) - 1
	return r.providers[idx%uint64(len(r.providers))]
}
