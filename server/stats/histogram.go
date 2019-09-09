package stats

import (
	"encoding/json"
	"expvar"
	"sync"
	"time"

	"github.com/circonus-labs/circonusllhist"
	"github.com/pkg/errors"
)

// map of *Histogram, needed to ensure we only register expvars
// once - there is no direct access to its builtin sync.Map
var histograms sync.Map

func initHistogram(key string, initFunc func()) *Histogram {
	ifc, loaded := histograms.LoadOrStore(key, &Histogram{circonusllhist.New()})
	if !loaded {
		expvar.Publish(key, expvar.Func(func() interface{} {
			ifc, _ := histograms.Load(key)
			if ifc == nil {
				return nil
			}
			h, _ := ifc.(*Histogram)
			return h
		}))

		if initFunc != nil {
			initFunc()
		}
	}
	hist, _ := ifc.(*Histogram)
	return hist
}

// RecordHistogramValue records value v into the histogram specified by key. If
// the histogram was not registered before, it invokes initFunc usually used to
// publish related expvar variables.
func recordHistogramValue(key string, v float64, initFunc func()) {
	initHistogram(key, initFunc).RecordValue(v)
}

// Histogram wraps circonusllhist.Histogram to implement expvar.Var
type Histogram struct {
	*circonusllhist.Histogram
}

func (h *Histogram) MarshalJSON() ([]byte, error) {
	pp := []string{"P50", "P85", "P95", "P98", "P99"}
	ppf := []float64{.50, .85, .95, .98, .99}

	quantiles, err := h.ApproxQuantile(ppf)
	if err != nil {
		return nil, err
	}
	if len(quantiles) != len(pp) {
		return nil, errors.Errorf("wrong number of quantiles returned, %v", len(quantiles))
	}

	out := map[string]string{}
	for i, p := range pp {
		d := time.Duration(quantiles[i])
		out[p] = d.String()
		// out[p] = strconv.FormatFloat(quantiles[i], 'e', 2, 64)
	}

	data, _ := json.Marshal(out)
	return data, nil
}

func (h *Histogram) String() string {
	data, _ := h.MarshalJSON()
	return string(data)
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
