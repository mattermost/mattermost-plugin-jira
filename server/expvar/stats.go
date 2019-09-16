package expvar

import (
	"encoding/json"
	goexpvar "expvar"
	"fmt"
	"math/rand"
	"regexp"
	"time"
)

// StatsSaveInterval specifies how often the stats are auto-saved.
const StatsSaveInterval = 1 * time.Minute

const statsSaveMaxDither = 10 // seconds

// Stats is a collection of Service metrics that can be persisted to/loaded
// from a store.
type Stats struct {
	Services map[string]*Service
}

// NewStatsFromData creates and initializes a new Stats, from previously
// serialized data. If it fails to unmarshal the data, it returns an empty Stats.
func NewStatsFromData(data []byte) (*Stats, error) {
	stats := &Stats{
		Services: map[string]*Service{},
	}
	err := json.Unmarshal(data, stats)
	fmt.Println("<><> expvar.NewStatsFromData: err:", err, stats)
	return stats, err
}

func (stats *Stats) Init(savef func([]byte)) {
	for _, service := range stats.Services {
		service.Init()
	}

	go func() {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		dither := time.Duration(r.Intn(statsSaveMaxDither)) * time.Second
		time.Sleep(dither)

		ticker := time.NewTicker(StatsSaveInterval)

		for now := range ticker.C {
			fmt.Println("<><> TIME TO SAVE!", now)
			data, err := json.Marshal(stats)
			if err != nil {
				fmt.Println("<><> failed to marshal", err)
				continue
			}
			savef(data)
			fmt.Println("<><> SAVED")
		}
	}()
}

func (stats *Stats) Reset() {
	for _, service := range stats.Services {
		service.Reset()
	}
}

// EnsureService makes sure that a service is registered in Stats in case it
// was not present in the initial configuration.
func (stats *Stats) EnsureService(name string, isAsync bool) *Service {
	fmt.Println("<><> expvar.Stats.EnsureService: ", name)
	service := stats.Services[name]
	if service == nil {
		fmt.Println("<><> expvar.Stats.EnsureService: not found")
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
