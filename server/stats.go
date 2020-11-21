package main

import (
	"crypto/md5" // #nosec G501
	"encoding/json"
	goexpvar "expvar"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/expvar"
)

const statsKeyExpiration = 30 * 24 * time.Hour // 30 days
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

	initUserCounter(p.userStore)
	initUptime()

	p.startAutosaveStats()
}

func (p *Plugin) httpAPIStats(w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodGet {
		return respondErr(w, http.StatusMethodNotAllowed,
			errors.New("method "+r.Method+" is not allowed, must be GET"))
	}
	conf := p.getConfig()

	isAdmin, err := authorizedSysAdmin(p, r.Header.Get("Mattermost-User-Id"))
	if err != nil {
		return http.StatusInternalServerError, errors.Wrap(err, "failed to authorize")
	}

	if !isAdmin {
		if conf.StatsSecret == "" {
			return respondErr(w, http.StatusForbidden,
				errors.New("access forbidden: must be authenticated as an admin, or provide the stats API secret"))
		}
		var status int
		status, err = verifyHTTPSecret(conf.StatsSecret, r.FormValue("secret"))
		if err != nil {
			return respondErr(w, status, err)
		}
	}
	if conf.stats == nil {
		return respondErr(w, http.StatusNotFound, errors.New("no stats available"))
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
	_, err = w.Write([]byte(out))
	if err != nil {
		return http.StatusInternalServerError, errors.WithMessage(err, "failed to write response")
	}
	return http.StatusOK, nil
}

func (p *Plugin) startAutosaveStats() {
	stop := make(chan bool)
	go func() {
		r := rand.New(rand.NewSource(time.Now().UnixNano())) // #nosec G404
		dither := time.Duration(r.Intn(statsAutosaveMaxDither)) * time.Second
		time.Sleep(dither)

		ticker := time.NewTicker(statsAutosaveInterval)
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				_ = p.saveStats()
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
	appErr := p.API.KVSetWithExpiry(statsKeyName(), data, int64(statsKeyExpiration.Seconds()))
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

func initUserCounter(userStore UserStore) {
	goexpvar.Publish("jira/mapped_users", goexpvar.Func(func() interface{} {
		c, err := userStore.CountUsers()
		if err != nil {
			return err.Error()
		}
		return c
	}))
}

var startedAt = time.Now()

// initUptime adds an "uptime" expvar
func initUptime() {
	goexpvar.Publish("uptime", goexpvar.Func(func() interface{} {
		up := time.Since(startedAt).Round(time.Second)
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
	h := md5.New() // #nosec G401
	_, _ = h.Write([]byte(hostname))
	return fmt.Sprintf("%s%x", prefixStats, h.Sum(nil))
}
