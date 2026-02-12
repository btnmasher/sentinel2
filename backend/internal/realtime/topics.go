package realtime

import (
	"encoding/json"
	"errors"
	"sync"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/tools/subscriptions"
)

const (
	TopicUploaderConfig = "uploader.config"
	TopicIntelUploaders = "intel.uploaders_count"
	DefaultQueueSize    = 256
)

type Publisher struct {
	app    *pocketbase.PocketBase
	queue  chan subscriptions.Message
	done   chan struct{}
	once   sync.Once
	closed bool
	mu     sync.RWMutex
}

func NewPublisher(app *pocketbase.PocketBase) *Publisher {
	p := &Publisher{
		app:   app,
		queue: make(chan subscriptions.Message, DefaultQueueSize),
		done:  make(chan struct{}),
	}
	go p.run()
	return p
}

func (p *Publisher) PublishJSON(topic string, payload any) (int, error) {
	if p == nil || p.app == nil || p.app.SubscriptionsBroker() == nil {
		return 0, nil
	}

	data, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		return 0, marshalErr
	}

	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.closed {
		return 0, errors.New("realtime publisher closed")
	}

	select {
	case p.queue <- subscriptions.Message{Name: topic, Data: data}:
		return 1, nil
	default:
		// queue full: drop newest to protect caller latency
		p.app.Logger().Warn("realtime publish dropped (queue full)")
		return 0, errors.New("realtime publish queue full")
	}
}

func (p *Publisher) Stop() {
	if p == nil {
		return
	}
	p.once.Do(func() {
		p.mu.Lock()
		p.closed = true
		p.mu.Unlock()
		close(p.done)
	})
}

func (p *Publisher) run() {
	for {
		select {
		case <-p.done:
			return
		case msg := <-p.queue:
			p.deliver(msg)
		}
	}
}

func (p *Publisher) deliver(msg subscriptions.Message) {
	if p == nil || p.app == nil || p.app.SubscriptionsBroker() == nil {
		return
	}
	for _, client := range p.app.SubscriptionsBroker().Clients() {
		if !client.HasSubscription(msg.Name) {
			continue
		}
		client.Send(msg)
	}
}
