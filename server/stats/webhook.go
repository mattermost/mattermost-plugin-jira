package stats

import (
	"sync"
)

const (
	WebhookLegacy    = "legacy"
	WebhookSubscribe = "subscribe"
)

type Webhook struct {
	HTTP      *Endpoint
	Processed *Endpoint
}

func NewWebhook(kind string) *Webhook {
	prefix := "webhook/" + kind + "/"
	return &Webhook{
		lock:      &sync.RWMutex{},
		HTTP:      EnsureEndpoint(prefix + "http"),
		Processed: EnsureEndpoint(prefix + "processed"),
	}
}
