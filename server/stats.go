package main

import (
	"encoding/json"
	goexpvar "expvar"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/mattermost/mattermost-plugin-jira/server/expvar"
)

const statsAutosaveInterval = 1 * time.Minute
const statsAutosaveMaxDither = 10 // seconds

var initStatsOnce sync.Once

func (p *Plugin) initStats() {
	conf := p.getConfig()
	if conf.DisableStats || conf.stats != nil {
		return
	}

	hostname, _ := os.Hostname()
	key := prefixStats + hostname
	data, appErr := p.API.KVGet(key)
	if appErr != nil {
		return
	}
	stats := expvar.NewStatsFromData(data)

	p.updateConfig(func(c *config) {
		initStatsOnce.Do(func() {
			if !c.DisableStats {
				c.stats = stats
				c.webhookResponseStats = stats.EnsureEndpoint("jira/webhook")
				c.subscribeResponseStats = stats.EnsureEndpoint("jira/subscribe/response")
				c.subscribeProcessingStats = stats.EnsureEndpoint("jira/subscribe/processing")
			}
		})
	})

	initUserCounter(p.currentInstanceStore, p.userStore)
	initUptime()

	p.startAutosaveStats()
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
	hostname, _ := os.Hostname()
	appErr := p.API.KVSet(prefixStats+hostname, data)
	if appErr != nil {
		return appErr
	}
	p.debugf("Saved stats, %q", prefixStats+hostname)
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

	hostname, _ := os.Hostname()
	appErr := p.API.KVDelete(prefixStats + hostname)
	if appErr != nil {
		return appErr
	}

	p.updateConfig(func(conf *config) {
		conf.webhookResponseStats = stats.EnsureEndpoint("jira/webhook")
		conf.subscribeResponseStats = stats.EnsureEndpoint("jira/subscribe/response")
		conf.subscribeProcessingStats = stats.EnsureEndpoint("jira/subscribe/processing")
	})

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
