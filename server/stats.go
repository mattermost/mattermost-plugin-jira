package main

import (
	"encoding/json"
	goexpvar "expvar"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mattermost/mattermost-plugin-jira/server/expvar"
	"github.com/pkg/errors"
)

const statsAutosaveInterval = 10 * time.Minute
const statsAutosaveMaxDither = 60 // seconds

var initStatsOnce sync.Once

func (p *Plugin) initStats() {
	conf := p.getConfig()
	if conf.DisableStats || conf.stats != nil {
		return
	}

	data, appErr := p.API.KVGet(statsKeyName())
	if appErr != nil {
		return
	}
	stats := expvar.NewStats(data)

	p.updateConfig(func(c *config) {
		initStatsOnce.Do(func() {
			if !c.DisableStats {
				c.stats = stats
			}
		})
	})

	initUserCounter(p.currentInstanceStore, p.userStore)
	initUptime()

	p.startAutosaveStats()
}

func httpAPIStats(p *Plugin, w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodGet {
		return http.StatusMethodNotAllowed,
			errors.New("method " + r.Method + " is not allowed, must be GET")
	}

	conf := p.getConfig()

	//TODO protect from unauthorized access?
	// status, err := verifyWebhookRequestSecret(conf, r)
	// if err != nil {
	// 	return status, err
	// }

	if conf.stats == nil {
		return http.StatusNotFound, errors.New("No stats available")
	}

	out := "{"
	first := true
	conf.stats.Do(func(name string, e *expvar.Endpoint) {
		if first {
			first = false
		} else {
			out += ","
		}
		out += `"` + name + `":` + e.String()
	})
	out += "}"
	_, err := w.Write([]byte(out))
	if err != nil {
		return http.StatusInternalServerError, errors.WithMessage(err, "failed to write response")
	}
	return http.StatusOK, nil
}

// To save the stats periodically, use `go Autosave(...)``
func (p *Plugin) startAutosaveStats() {
	stop := make(chan bool)
	go func() {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		dither := time.Duration(r.Intn(statsAutosaveMaxDither)) * time.Second
		time.Sleep(dither)

		ticker := time.NewTicker(statsAutosaveInterval)
		for {
			select {
			case _ = <-stop:
				return
			case _ = <-ticker.C:
				p.saveStats()
			}
		}
	}()

	p.updateConfig(func(c *config) {
		c.statsStopAutosave = stop
	})
}

// To save the stats periodically, use `go Autosave(...)``
func (p *Plugin) stopAutosaveStats(conf config) {
	stop := conf.statsStopAutosave
	if stop == nil {
		return
	}
	p.updateConfig(func(c *config) {
		c.statsStopAutosave = nil
	})
	stop <- true
}

func (p *Plugin) saveStats() error {
	stats := p.getConfig().stats
	if stats == nil {
		return nil
	}
	data, err := json.Marshal(stats)
	if err != nil {
		return err
	}
	appErr := p.API.KVSet(statsKeyName(), data)
	if appErr != nil {
		return appErr
	}
	return nil
}

// This is only useful in a single-server context, so can not be used in production
// TODO: Need a way to reset all stats in production?
func (p *Plugin) debugResetStats() error {
	conf := p.getConfig()
	p.stopAutosaveStats(conf)
	defer p.startAutosaveStats()

	stats := conf.stats
	if stats == nil {
		return nil
	}
	stats.Reset()

	for i := 0; ; i++ {
		keys, appErr := p.API.KVList(i, listPerPage)
		if appErr != nil {
			return appErr
		}
		for _, key := range keys {
			if !strings.HasPrefix(key, prefixStats) {
				continue
			}
			appErr := p.API.KVDelete(key)
			if appErr != nil {
				return appErr
			}
		}
		if len(keys) < listPerPage {
			break
		}
	}

	return nil
}

func initUserCounter(currentInstanceStore CurrentInstanceStore, userStore UserStore) {
	goexpvar.Publish("jira/mapped_users", goexpvar.Func(func() interface{} {
		ji, err := currentInstanceStore.LoadCurrentJIRAInstance()
		if err != nil {
			return err.Error()
		}
		c, err := userStore.CountUsers(ji)
		if err != nil {
			return err.Error()
		}
		return c
	}))
}

var startedAt = time.Now()

// EnsureUptime adds an "uptime" expvar
func initUptime() {
	goexpvar.Publish("uptime", goexpvar.Func(func() interface{} {
		up := (time.Since(startedAt) + time.Second/2) / time.Second * time.Second
		return up.String()
	}))
}

func (p *Plugin) consolidatedStoredStats() (*expvar.Stats, []string, error) {
	stats := expvar.NewUnpublishedStats(nil)
	var statsKeys []string
	for i := 0; ; i++ {
		keys, appErr := p.API.KVList(i, listPerPage)
		if appErr != nil {
			return nil, nil, appErr
		}

		for _, key := range keys {
			if !strings.HasPrefix(key, prefixStats) {
				continue
			}
			var data []byte
			data, appErr = p.API.KVGet(key)
			if appErr != nil {
				return nil, nil, appErr
			}
			from := expvar.NewUnpublishedStats(data)
			stats.Merge(from)
			statsKeys = append(statsKeys, key)
		}

		if len(keys) < listPerPage {
			break
		}
	}
	return stats, statsKeys, nil
}

func statsKeyName() string {
	hostname, _ := os.Hostname()
	return prefixStats + hostname
}
