package jobs

import (
	"context"
	"sync"
)

type cancelRegistry struct {
	mu      sync.Mutex
	cancels map[string]context.CancelFunc
}

var registry = cancelRegistry{
	cancels: make(map[string]context.CancelFunc),
}

func RegisterCancel(jobID string, cancel context.CancelFunc) {
	if jobID == "" || cancel == nil {
		return
	}
	registry.mu.Lock()
	registry.cancels[jobID] = cancel
	registry.mu.Unlock()
}

func UnregisterCancel(jobID string) {
	if jobID == "" {
		return
	}
	registry.mu.Lock()
	delete(registry.cancels, jobID)
	registry.mu.Unlock()
}

func Cancel(jobID string) bool {
	if jobID == "" {
		return false
	}
	registry.mu.Lock()
	cancel, ok := registry.cancels[jobID]
	if ok {
		delete(registry.cancels, jobID)
	}
	registry.mu.Unlock()
	if ok {
		cancel()
	}
	return ok
}

func IsRegistered(jobID string) bool {
	if jobID == "" {
		return false
	}
	registry.mu.Lock()
	_, ok := registry.cancels[jobID]
	registry.mu.Unlock()
	return ok
}
