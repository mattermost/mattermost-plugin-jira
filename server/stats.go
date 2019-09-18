package main

import (
	goexpvar "expvar"
	"os"
	"sync"
	"time"

	"github.com/mattermost/mattermost-plugin-jira/server/expvar"
)

const statsAutosaveInterval = 1 * time.Hour

var initStatsOnce sync.Once

type stats struct {
	*expvar.Stats
	jira             *expvar.Service
	legacyWebhook    *expvar.Service
	subscribeWebhook *expvar.Service
}

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

	expstats := expvar.NewStatsFromData(data, statsAutosaveInterval, p.saveStatsF)

	stats := stats{
		jira:             expstats.EnsureService("api/jira", false),
		legacyWebhook:    expstats.EnsureService("webhook/jira/legacy", false),
		subscribeWebhook: expstats.EnsureService("webhook/jira/subscribe", true),
		Stats:            expstats,
	}

	p.updateConfig(func(conf *config) {
		initStatsOnce.Do(func() {
			if conf.DisableStats || conf.stats != nil {
				return
			}
			conf.stats = &stats
		})
	})

	p.initUserCounter()
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

	stats.jira = stats.EnsureService("api/jira", false)
	stats.legacyWebhook = stats.EnsureService("webhook/jira/legacy", false)
	stats.subscribeWebhook = stats.EnsureService("webhook/jira/subscribe", true)

	return nil
}

func (p *Plugin) initUserCounter() {
	goexpvar.Publish("counters/mapped_users", goexpvar.Func(func() interface{} {
		ji, err := p.currentInstanceStore.LoadCurrentJIRAInstance()
		if err != nil {
			return err.Error()
		}
		c, err := p.userStore.CountUsers(ji)
		if err != nil {
			return err.Error()
		}
		return c
	}))
}
