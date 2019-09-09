package stats

import (
	"expvar"
	"fmt"
	"time"
)

const (
	WebhookLegacy    = "legacy"
	WebhookSubscribe = "subscribe"
)

func init() {
	initWebhook := func(version string) {
		prefix := fmt.Sprintf("webhook/%s/", version)
		initHistogram(prefix+"response", nil)
		initHistogram(prefix+"processed", nil)
		expvar.NewInt(prefix + "total")
		expvar.NewInt(prefix + "ignored")
		expvar.NewInt(prefix + "errors")
	}

	initWebhook(WebhookLegacy)
	initWebhook(WebhookSubscribe)
}

func RecordWebhookResponse(version string, isError bool, elapsed time.Duration) {
	prefix := fmt.Sprintf("webhook/%s/", version)

	recordHistogramValue(prefix+"response", float64(elapsed), nil)
	if isError {
		incrementCounter(prefix + "errors")
	}
	incrementCounter(prefix + "total")
}

func RecordWebhookProcessed(version string, isError, isIgnored bool, elapsed time.Duration) {
	prefix := fmt.Sprintf("webhook/%s/", version)

	recordHistogramValue(prefix+"processed", float64(elapsed), nil)
	if isError {
		incrementCounter(prefix + "errors")
	}
	if isIgnored {
		incrementCounter(prefix + "ignored")
	}
}
