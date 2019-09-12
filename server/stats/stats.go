package stats

import (
	"expvar"
)

type stats struct {
	LegacyWebhook    webhookData
	SubscribeWebhook webhookData
}

// IncrementCounter increments a counter value
func (s *stats) IncrementCounter(key string) {
	v := expvar.Get(key)
	if v == nil {
		return
	}
	i, _ := v.(*expvar.Int)
	if i == nil {
		return
	}
	i.Add(1)
}
