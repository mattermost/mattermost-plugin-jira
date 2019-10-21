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

// NewStats creates and publishes a new Stats expvar, from previously
// serialized data. If it fails to unmarshal the data, it returns an empty Stats.
func NewStats(data []byte) *Stats {
	stats := Stats{}
	_ = json.Unmarshal(data, &stats)
	stats.Do(func(name string, e *Endpoint) {
		e.publishExpvar()
	})
	return &stats
}

// NewUnpublishedStats creates a new Stats, from previously
// serialized data. If it fails to unmarshal the data, it returns an empty Stats.
func NewUnpublishedStats(data []byte) *Stats {
	stats := Stats{}
	_ = json.Unmarshal(data, &stats)
	stats.disableExpvars = true
	return &stats
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
	initialValue := NewUnpublishedEndpoint(name)
	ifc, loaded := stats.endpoints.LoadOrStore(name, initialValue)
	e := ifc.(*Endpoint)

	// Publish the expvar 1-time only,
	if !loaded && !stats.disableExpvars {
		e.publishExpvar()
	}
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

// UnmarshalJSON implements json.Unmarshaler.
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

func (stats *Stats) Merge(multi ...*Stats) {
	for _, fromStats := range multi {
		fromStats.Do(func(name string, from *Endpoint) {
			to := stats.EnsureEndpoint(name)
			to.Merge(from)
		})
	}
}

// PrintConsolidated outputs all endpoints that match pattern, as markdown.
func (stats *Stats) PrintConsolidated(pattern string) (string, error) {
	var re *regexp.Regexp
	var err error

	if pattern != "" {
		re, err = regexp.Compile(pattern)
		if err != nil {
			return "", err
		}
	}

	bullet := func(k, v string) string {
		if v == "" || v == "{}" {
			return ""
		}
		return fmt.Sprintf(" * %s: `%s`\n", k, v)
	}

	resp := ""
	stats.Do(func(name string, e *Endpoint) {
		if re == nil || re.MatchString(name) {
			resp += bullet(name, e.String())
		}
	})
	return resp, nil
}

// PrintExpvars outputs all expvars that match pattern, as markdown.
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
