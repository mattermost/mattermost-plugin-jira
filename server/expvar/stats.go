package expvar

import (
	"encoding/json"
	goexpvar "expvar"
	"fmt"
	"math/rand"
	"regexp"
	"time"
)

const statsSaveMaxDither = 10 // seconds

// Stats is a collection of Service metrics that can be persisted to/loaded
// from a store.
type Stats struct {
	Services map[string]*Service
}

// NewStatsFromData creates and initializes a new Stats, from previously
// serialized data. If it fails to unmarshal the data, it returns an empty Stats.
// If saveInterval and savef are provided, it starts the autosave goroutine for
// the stats.
func NewStatsFromData(data []byte, saveInterval time.Duration, savef func([]byte)) *Stats {
	// ignore the error - just return an empty set if failed to unmarshal
	stats := Stats{}
	json.Unmarshal(data, &stats)
	if stats.Services == nil {
		stats.Services = map[string]*Service{}
	}
	for _, service := range stats.Services {
		service.Init()
	}

	// autosave
	if saveInterval > 0 && savef != nil {
		go func() {
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			dither := time.Duration(r.Intn(statsSaveMaxDither)) * time.Second
			time.Sleep(dither)

			ticker := time.NewTicker(saveInterval)
			for range ticker.C {
				stats.Save(savef)
			}
		}()
	}

	return &stats
}

func (stats *Stats) Save(savef func([]byte)) {
	data, err := json.Marshal(stats)
	if err != nil {
		return
	}
	savef(data)
}

func (stats *Stats) Reset() {
	for _, service := range stats.Services {
		service.Reset()
	}
	stats.Services = map[string]*Service{}
}

// EnsureService makes sure that a service is registered in Stats in case it
// was not present in the initial configuration.
func (stats *Stats) EnsureService(name string, isAsync bool) *Service {
	service := stats.Services[name]
	if service == nil {
		service = NewService(name, isAsync)
		stats.Services[name] = service
	}
	return service
}

// PrintStats outputs all expvars that match pattern, as markdown
func PrintStats(pattern string) (string, error) {
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
