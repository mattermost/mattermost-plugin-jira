package stats

import (
	"expvar"
	"fmt"
	"time"
)

func init() {
	expvar.NewInt("client/_/total")
	expvar.NewInt("client/_/errors")
}

func initClientAPI(apiKey string) func() {
	return func() {
		expvar.NewInt(fmt.Sprintf("client/%s/total", apiKey))
		expvar.NewInt(fmt.Sprintf("client/%s/errors", apiKey))
	}
}

func RecordClientAPI(apiKey string, isError bool, elapsed time.Duration) {
	recordHistogramValue(fmt.Sprintf("client/%s/latency", apiKey), float64(elapsed), initClientAPI(apiKey))

	if isError {
		incrementCounter("client/_/errors")
		incrementCounter(fmt.Sprintf("client/%s/errors", apiKey))
	}
	incrementCounter("client/_/total")
	incrementCounter(fmt.Sprintf("client/%s/total", apiKey))
}
