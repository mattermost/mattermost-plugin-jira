package stats

import (
	"expvar"
	"fmt"
	"sync"
	"time"

	"github.com/circonus-labs/circonusllhist"
)

// map of *circonusllhist.Histogram by API name
var clientHistograms sync.Map

var totalAPI = expvar.NewInt("/api/_/total")
var totalAPIErrors = expvar.NewInt("api/_/errors")

func RecordAPI(api string, isError bool, elapsed time.Duration) {
	ifc, loaded := clientHistograms.LoadOrStore(api, circonusllhist.New())
	if !loaded {
		registerAPI(api)
	}
	hist, _ := ifc.(*circonusllhist.Histogram)
	if hist == nil {
		return
	}
	millis := elapsed / time.Millisecond
	hist.RecordValue(float64(millis))

	if isError {
		incrementCounter("api/_/errors")
		incrementCounter(fmt.Sprintf("api/%s/errors", api))
	}
	incrementCounter("api/_/total")
	incrementCounter(fmt.Sprintf("api/%s/total", api))
}

// RegisterAPI creates expvar for an API entry. For an API X it will register
// - `/api/X/total`: total number of API requests processed
// - `/api/X/errors`: number of errors
// - `/api/X/p[50,85,93,97,99]`: Percentiles, 50 is the average time
func registerAPI(api string) {
	expvar.NewInt(fmt.Sprintf("api/%s/total", api))
	expvar.NewInt(fmt.Sprintf("api/%s/errors", api))

	pX := func(p float64) func() interface{} {
		return func() interface{} {
			ifc, _ := clientHistograms.Load(api)
			if ifc == nil {
				return nil
			}
			hist, _ := ifc.(*circonusllhist.Histogram)
			if hist == nil {
				return nil
			}
			millis := hist.ValueAtQuantile(p)
			d := time.Duration(millis) * time.Millisecond
			return d.String()
		}
	}

	for _, p := range []int{50, 85, 93, 97, 99} {
		expvar.Publish(
			fmt.Sprintf("api/%s/p%v", api, p),
			expvar.Func(pX(float64(p)/100.0)))
	}
}

func incrementCounter(key string) {
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
