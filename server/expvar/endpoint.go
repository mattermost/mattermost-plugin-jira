package expvar

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"expvar"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/circonus-labs/circonusllhist"

	"github.com/mattermost/mattermost-plugin-jira/server/utils"
)

// Endpoint implements a expvar.Var and json.[Un-]Marshaller interfaces.
// Its String() method returns aggregated values, the JSON methods serialize
// and deserialize complete data and can be used to persist/restore.
// Note that Get should be used to safely access Endpoint data in goroutines.
type Endpoint struct {
	lock *sync.RWMutex

	Name         string
	Total        int64
	Errors       int64
	Ignored      int64
	Elapsed      *circonusllhist.Histogram // time.Durations
	RequestSize  *circonusllhist.Histogram // byte sizes
	ResponseSize *circonusllhist.Histogram // byte sizes
}

func NewEndpoint(name string) *Endpoint {
	e := NewUnpublishedEndpoint(name)
	e.publishExpvar()
	return e
}

func NewUnpublishedEndpoint(name string) *Endpoint {
	return &Endpoint{
		lock:         &sync.RWMutex{},
		Name:         name,
		Elapsed:      circonusllhist.NewNoLocks(),
		RequestSize:  circonusllhist.NewNoLocks(),
		ResponseSize: circonusllhist.NewNoLocks(),
	}
}

func (e *Endpoint) publishExpvar() {
	expvar.Publish(e.Name, e)
}

// Reset clears all values in the endpoint
func (e *Endpoint) Reset() {
	if e == nil {
		return
	}
	if e.lock != nil {
		e.lock.Lock()
		defer e.lock.Unlock()
	}

	e.Total = 0
	e.Errors = 0
	e.Ignored = 0
	e.Elapsed.Reset()
	e.RequestSize.Reset()
	e.ResponseSize.Reset()
}

// Record records a single event
func (e *Endpoint) Record(reqSize, respSize utils.ByteSize, dur time.Duration, isError, isIgnored bool) {
	if e == nil {
		return
	}
	if e.lock != nil {
		e.lock.Lock()
		defer e.lock.Unlock()
	}

	e.RequestSize.RecordValue(float64(reqSize))
	e.ResponseSize.RecordValue(float64(respSize))
	e.Elapsed.RecordValue(float64(dur))

	if isError {
		e.Errors++
	}
	if isIgnored {
		e.Ignored++
	}
	e.Total++
}

// Get returns a copy of the Endpoint that is safe to read from in goroutines.
func (e *Endpoint) Get() Endpoint {
	if e == nil {
		return Endpoint{}
	}
	if e.lock != nil {
		e.lock.RLock()
		defer e.lock.RUnlock()
	}
	ep := *e
	ep.lock = nil
	return ep
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
		"RequestSize": mapPercentiles(ep.RequestSize, func(f float64) string {
			return utils.ByteSize(f).String()
		}),
		"ResponseSize": mapPercentiles(ep.ResponseSize, func(f float64) string {
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

// UnmarshalJSON implements json.Unmarshaler interface
func (e *Endpoint) UnmarshalJSON(data []byte) error {
	if e.lock != nil {
		e.lock.Lock()
		defer e.lock.Unlock()
	}

	ee := struct {
		Name                               string
		Total, Errors, Ignored             int64
		Elapsed, RequestSize, ResponseSize string
	}{}
	err := json.Unmarshal(data, &ee)
	if err != nil {
		return err
	}

	unmarshalHistogram := func(s string) (*circonusllhist.Histogram, error) {
		var hdata []byte
		hdata, err = base64.StdEncoding.DecodeString(s)
		if err != nil {
			return nil, err
		}
		return circonusllhist.Deserialize(bytes.NewBuffer(hdata))
	}

	elapsed, err := unmarshalHistogram(ee.Elapsed)
	if err != nil {
		return err
	}
	reqSize, err := unmarshalHistogram(ee.RequestSize)
	if err != nil {
		return err
	}
	respSize, err := unmarshalHistogram(ee.ResponseSize)
	if err != nil {
		return err
	}
	e.Name = ee.Name
	e.Total = ee.Total
	e.Errors = ee.Errors
	e.Ignored = ee.Ignored
	e.Elapsed = elapsed
	e.RequestSize = reqSize
	e.ResponseSize = respSize

	if e.lock == nil {
		e.lock = &sync.RWMutex{}
	}
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

var decStringsRegexp = regexp.MustCompile(`^H\[(.+)\]=([0-9]+)$`)

// UnmarshalJSON implements json.Unmarshaler interface
func (e *Endpoint) Merge(multi ...*Endpoint) {
	mergeHistogram := func(to, from *circonusllhist.Histogram) {
		if from == nil {
			return
		}
		decStrings := from.DecStrings()

		for _, s := range decStrings {
			// Parse the strings as `H[1.0e+09]=4`
			found := decStringsRegexp.FindStringSubmatch(s)
			if len(found) != 3 {
				continue
			}
			value, err := strconv.ParseFloat(found[1], 64)
			if err != nil {
				continue
			}
			count, err := strconv.ParseInt(found[2], 10, 64)
			if err != nil {
				continue
			}
			to.RecordValues(value, count)
		}
	}

	for _, from := range multi {
		e.Total += from.Total
		e.Errors += from.Errors
		e.Ignored += from.Ignored
		mergeHistogram(e.Elapsed, from.Elapsed)
		mergeHistogram(e.RequestSize, from.RequestSize)
		mergeHistogram(e.ResponseSize, from.ResponseSize)
	}
}
