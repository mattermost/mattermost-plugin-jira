package stats

import (
	"bytes"
	"encoding/json"
	"expvar"
	"fmt"
	"sync"
	"time"

	"github.com/circonus-labs/circonusllhist"
)

type Endpoints struct {
	sync.Map
}

func (known *Endpoints) EnsureEndpoint(name string) *Endpoint {
	v := &Endpoint{
		name:    name,
		Latency: circonusllhist.NewNoLocks(),
	}
	loaded := false
	if known != nil {
		var i interface{}
		i, loaded = known.LoadOrStore(name, v)
		v = i.(*Endpoint)
	}

	// If it's the first time, publish the expvar
	if !loaded {
		expvar.Publish(name, v)
	}
	return v
}

func (known *Endpoints) MarshalJSON() ([]byte, error) {
	b := bytes.Buffer{}
	fmt.Fprintf(&b, "{")
	first := true
	known.Range(func(key, value interface{}) bool {
		if !first {
			fmt.Fprintf(&b, ", ")
		}
		// value implenents String()
		fmt.Fprintf(&b, "%q: %s", key, value)
		first = false
		return true
	})
	fmt.Fprintf(&b, "}")
	return b.Bytes(), nil
}

func (known *Endpoints) UnmarshalJSON(data []byte) error {
	m := map[string]Endpoint{}
	err := json.Unmarshal(data, &m)
	if err != nil {
		return err
	}
	for k, v := range m {
		known.EnsureEndpoint(k).Set(v)
	}
	return nil
}

// Endpoint implements a expvar.Var and json.[Un-]Marshaller interfaces.
// Its String() method returns aggregated values, the JSON methods serialize
// and deserialize complete data and can be used to persist/restore.
type Endpoint struct {
	lock *sync.RWMutex
	name string

	Total   int64
	Errors  int64
	Ignored int64
	Latency *circonusllhist.Histogram
}

func (e *Endpoint) RecordEvent(elapsed time.Duration, isError, isIgnored bool) {
	if e.lock != nil {
		e.lock.Lock()
		defer e.lock.Unlock()
	}

	e.Latency.RecordValue(float64(elapsed))

	if isError {
		e.Errors++
	}
	if isIgnored {
		e.Ignored++
	}
	e.Total++
}

func (e *Endpoint) Value() Endpoint {
	if e.lock != nil {
		e.lock.RLock()
		defer e.lock.RUnlock()
	}
	ep := *e
	ep.lock = nil
	return ep
}

// String() implements expvar.Var interface
func (e *Endpoint) String() string {
	ep := e.Value()
	m := map[string]interface{}{
		"Total":   ep.Total,
		"Errors":  ep.Errors,
		"Latency": mapPercentiles(ep.Latency),
	}
	if ep.Ignored != 0 {
		m["Ignored"] = ep.Ignored
	}
	data, _ := json.Marshal(m)
	return string(data)
}

func (e *Endpoint) Set(ep Endpoint) {
	if e.lock != nil {
		e.lock.Lock()
		defer e.lock.Unlock()
	}
	lock := e.lock
	*e = ep
	e.lock = lock
}

func mapPercentiles(h *circonusllhist.Histogram) map[string]string {
	pp := []string{"P50", "P85", "P95", "P98", "P99"}
	ppf := []float64{.50, .85, .95, .98, .99}

	quantiles, err := h.ApproxQuantile(ppf)
	if err != nil {
		return nil
	}
	if len(quantiles) != len(pp) {
		return nil
	}

	out := map[string]string{}
	for i, p := range pp {
		out[p] = time.Duration(quantiles[i]).String()
	}
	return out
}
