package main

import (
	goexpvar "expvar"
	"os"
	"sync"
	"time"

	"github.com/mattermost/mattermost-plugin-jira/server/expvar"
)

const statsAutosaveInterval = 1 * time.Minute

// const statsAutosaveInterval = 1 * time.Hour

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

	p.updateConfig(func(conf *config) {
		initStatsOnce.Do(func() {
			if !conf.DisableStats {
				conf.stats = stats
				conf.webhookResponseStats = stats.Endpoint("jira/webhook")
				conf.subscribeResponseStats = stats.Endpoint("jira/subscribe/response")
				conf.subscribeProcessingStats = stats.Endpoint("jira/subscribe/processing")
			}
		})
	})

	initUserCounter(p.currentInstanceStore, p.userStore)
	initUptime()

	go stats.Autosave(statsAutosaveInterval, p.saveStatsF)
}

func (p *Plugin) saveStatsF(data []byte) {
	hostname, _ := os.Hostname()
	appErr := p.API.KVSet(prefixStats+hostname, data)
	if appErr != nil {
		return
	}
}

func (p *Plugin) resetStats() error {
	stats := p.getConfig().stats
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
		if conf.stats != nil {
			conf.webhookResponseStats = stats.Endpoint("jira/webhook")
			conf.subscribeResponseStats = stats.Endpoint("jira/subscribe/response")
			conf.subscribeProcessingStats = stats.Endpoint("jira/subscribe/processing")
		}
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
