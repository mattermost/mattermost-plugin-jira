package main

import (
	goexpvar "expvar"
	"fmt"
	"os"
	"sync"

	"github.com/mattermost/mattermost-plugin-jira/server/expvar"
)

type stats struct {
	*expvar.Stats

	jira             *expvar.Service
	legacyWebhook    *expvar.Service
	subscribeWebhook *expvar.Service
}

var initStatsOnce sync.Once

func (p *Plugin) loadStats() {
	conf := p.getConfig()
	if conf.DisableStats || conf.stats != nil {
		fmt.Println("<><> loadStats: no need to load")
		return
	}

	hostname, _ := os.Hostname()
	key := prefixStats + hostname
	data, appErr := p.API.KVGet(key)
	if appErr != nil {
		fmt.Println("<><> loadStats: error loading", key, appErr.Error())
		return
	}

	fmt.Println("<><> loadStats: loaded from KV", key, string(data))
	p.updateConfig(func(conf *config) {
		if conf.DisableStats || conf.stats != nil {
			fmt.Println("<><> loadStats: already loaded in between")
			return
		}

		initStatsOnce.Do(func() {
			stats, err := newStatsFromData(data, p.currentInstanceStore, p.userStore, p.saveStats)
			if err != nil {
				p.errorf("Ignored invalid previous stats data: %v", err)
				return
			}
			conf.stats = stats

			fmt.Printf("<><> loadStats: loaded\nJira: %v\nLegacy: %v, Subscribe: %v",
				stats.jira,
				stats.legacyWebhook,
				stats.subscribeWebhook)
		})
	})
}

func (p *Plugin) saveStats(data []byte) {
	hostname, _ := os.Hostname()
	appErr := p.API.KVSet(prefixStats+hostname, data)
	if appErr != nil {
		fmt.Println("<><> saveStats: error savining", appErr.Error())
		return
	}
	fmt.Println("<><> saveStats: saved", string(data))
}

func (p *Plugin) resetStats() error {
	hostname, _ := os.Hostname()
	appErr := p.API.KVDelete(prefixStats + hostname)
	if appErr != nil {
		return appErr
	}

	fmt.Println("<><> !!!!!!!!!!!!!!!!!!!!!!!!", p.getConfig().stats)
	return nil
}

func newStatsFromData(data []byte, currentInstanceStore CurrentInstanceStore,
	userStore UserStore, savef func([]byte)) (*stats, error) {

	goexpvar.Publish("counters/mapped_users", goexpvar.Func(func() interface{} {
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

	// NewStatsFromData always returns a stats pointer. If the data intitalization
	// failed, err is set.
	expstats, err := expvar.NewStatsFromData(data)
	stats := &stats{
		Stats:            expstats,
		jira:             expstats.EnsureService("api/jira", false),
		legacyWebhook:    expstats.EnsureService("webhook/jira/legacy", true),
		subscribeWebhook: expstats.EnsureService("webhook/jira/subscribe", true),
	}

	stats.Init(savef)
	return stats, err
}
