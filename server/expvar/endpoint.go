package expvar

import (
	"encoding/json"
	"expvar"
	"fmt"
	"sync"
	"time"

	"github.com/circonus-labs/circonusllhist"
	"github.com/mattermost/mattermost-plugin-jira/server/utils"
)

var endpoints = sync.Map{} // *Var

// Endpoint implements a expvar.Var and json.[Un-]Marshaller interfaces.
// Its String() method returns aggregated values, the JSON methods serialize
// and deserialize complete data and can be used to persist/restore.
// Note that Get and Update methods should be used to safely access Endpoint
// data in goroutines.
type Endpoint struct {
	lock *sync.RWMutex

	Name    string
	Total   int64
	Errors  int64
	Ignored int64
	Elapsed *circonusllhist.Histogram // time.Durations
	Size    *circonusllhist.Histogram // byte sizes
}

// NewEndpoint creates and publishes a new expvar for the endpoint. Unlike the
// rest of the expvar.New functions, this can be called repeteadly for the same
// name and will return the same Var pointer for it.
func NewEndpoint(name string) *Endpoint {
	e := &Endpoint{
		lock:    &sync.RWMutex{},
		Name:    name,
		Elapsed: circonusllhist.NewNoLocks(),
		Size:    circonusllhist.NewNoLocks(),
	}
	ifc, loaded := endpoints.LoadOrStore(name, e)
	if !loaded {
		expvar.Publish(name, e)
		fmt.Printf("<><> EndpointExpvar: published %q\n", name)
	}
	e = ifc.(*Endpoint)
	return e
}

// Record records a single event
func (e *Endpoint) Record(size utils.ByteSize, dur time.Duration, isError, isIgnored bool) {
	if e.lock != nil {
		e.lock.Lock()
		defer e.lock.Unlock()
	}

	e.Size.RecordValue(float64(size))
	e.Elapsed.RecordValue(float64(dur))

	if isError {
		e.Errors++
	}
	if isIgnored {
		e.Ignored++
	}
	e.Total++
	fmt.Printf("<><> expvar Endpoint.Record: recorded %v %v into %q (%p)\n", size, dur, e.Name, e)
}

// Get returns a copy of the Endpoint that is safe to read from in goroutines.
func (e *Endpoint) Get() Endpoint {
	if e.lock != nil {
		e.lock.RLock()
		defer e.lock.RUnlock()
	}
	ep := *e
	ep.lock = nil
	return ep
}

// Update updates all values from another Endpoint, leaves the lock untouched. It
// is safe to use from goroutines.
func (e *Endpoint) Update(ep Endpoint) {
	if e.lock != nil {
		e.lock.Lock()
		defer e.lock.Unlock()
	}
	lock := e.lock
	*e = ep
	fmt.Printf("<><> Endpoint.Set: %+v\n", e)
	e.lock = lock
}

// String implements expvar.Var interface
func (e *Endpoint) String() string {
	if e == nil || e.Total == 0 {
		return "{}"
	}
	ep := e.Get()
	m := map[string]interface{}{
		"Total":  ep.Total,
		"Errors": ep.Errors,
		"Elapsed": mapPercentiles(ep.Elapsed, func(f float64) string {
			return time.Duration(f).String()
		}),
		"Size": mapPercentiles(ep.Size, func(f float64) string {
			return utils.ByteSize(f).String()
		}),
	}
	if ep.Ignored != 0 {
		m["Ignored"] = ep.Ignored
	}
	data, _ := json.Marshal(m)
	return string(data)
}

// MarshalJSON implements json.Marshaler interface
func (e *Endpoint) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.Get())
}

// UnmarshalJSON implements json.Unarshaler interface
func (e *Endpoint) UnmarshalJSON(data []byte) error {
	err := json.Unmarshal(data, &e)
	if err != nil {
		return err
	}
	if e.lock == nil {
		e.lock = &sync.RWMutex{}
	}
	fmt.Printf("<><> Endpoint.UnmarshalJSON: %+v \n", e)
	return nil
}

func mapPercentiles(h *circonusllhist.Histogram, toString func(f float64) string) map[string]string {
	pp := []string{"P10", "P50", "P85", "P95", "P98", "P99"}
	ppf := []float64{.10, .50, .85, .95, .98, .99}

	quantiles, err := h.ApproxQuantile(ppf)
	if err != nil {
		return nil
	}
	if len(quantiles) != len(pp) {
		return nil
	}

	out := map[string]string{}
	for i, p := range pp {
		out[p] = toString(quantiles[i])
	}
	return out
}
