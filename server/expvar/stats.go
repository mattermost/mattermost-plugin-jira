package expvar

import (
	"encoding/json"
	goexpvar "expvar"
	"fmt"
	"regexp"
	"sync"
)

type Stats struct {
	disableExpvars bool
	endpoints      sync.Map // *Endpoint
}

// NewStatsFromData creates and publishes a new Stats, from previously
// serialized data. If it fails to unmarshal the data, it returns an empty Stats.
// If saveInterval and savef are provided, it starts the autosave goroutine for
// the stats.
func NewStatsFromData(data []byte) *Stats {
	stats := Stats{}
	json.Unmarshal(data, &stats)
	stats.Do(func(name string, e *Endpoint) {
		e = stats.ensureEndpoint(name, e, false)
		e.publishExpvar()
	})
	return &stats
}

func newStats(data []byte, disableExpvars bool) *Stats {
	// ignore the error - just return an empty set if failed to unmarshal
	stats := Stats{
		disableExpvars: disableExpvars,
	}
	json.Unmarshal(data, &stats)
	stats.Do(func(name string, e *Endpoint) {
		stats.ensureEndpoint(name, e, disableExpvars)
	})
	return &stats
}

func (stats *Stats) ensureEndpoint(name string, initialValue *Endpoint, disableExpvar bool) *Endpoint {
	if initialValue == nil {
		// Make an Endpoint, but don't publish to expvar just yet
		initialValue = newEndpoint(name)
	}
	ifc, loaded := stats.endpoints.LoadOrStore(name, initialValue)
	e := ifc.(*Endpoint)

	// Publish the expvar 1-time only,
	if !loaded && !disableExpvar {
		e.publishExpvar()
	}
	return e
}

func (stats *Stats) Do(f func(name string, e *Endpoint)) {
	stats.endpoints.Range(func(key, value interface{}) bool {
		name := key.(string)
		e := value.(*Endpoint)
		f(name, e)
		return true
	})
}

func (stats *Stats) EnsureEndpoint(name string) *Endpoint {
	e := stats.ensureEndpoint(name, nil, stats.disableExpvars)
	return e
}

func (stats *Stats) Reset() {
	stats.Do(func(name string, e *Endpoint) {
		e.Reset()
	})
}

// MarshalJSON implements json.Marshaller.
func (stats *Stats) MarshalJSON() ([]byte, error) {
	v := map[string]*Endpoint{}
	stats.Do(func(name string, e *Endpoint) {
		v[name] = e
	})
	return json.Marshal(v)
}

// UnmarshalJSON implements json.Unmarshaller.
func (stats *Stats) UnmarshalJSON(data []byte) error {
	v := map[string]*Endpoint{}
	err := json.Unmarshal(data, &v)
	if err != nil {
		return err
	}
	for k, e := range v {
		stats.endpoints.Store(k, e)
	}
	return nil
}

// PrintExpvars outputs all expvars that match pattern, as markdown
func PrintExpvars(pattern string) (string, error) {
	var re *regexp.Regexp
	var err error

	if pattern != "" {
		re, err = regexp.Compile(pattern)
		if err != nil {
			return "", err
		}
	}

	bullet := func(cond bool, k string, v interface{}) string {
		if !cond {
			return ""
		}
		return fmt.Sprintf(" * %s: `%v`\n", k, v)
	}

	sbullet := func(k, v string) string {
		return bullet(v != "", k, v)
	}

	resp := ""
	goexpvar.Do(func(kv goexpvar.KeyValue) {
		if re == nil || re.MatchString(kv.Key) {
			resp += sbullet(kv.Key, kv.Value.String())
		}
	})
	return resp, nil
}
