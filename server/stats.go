package main

import (
	"encoding/json"
	goexpvar "expvar"
	"math/rand"
	"time"

	"github.com/mattermost/mattermost-plugin-jira/server/expvar"
)

// StatsSaveInterval specifies how often the stats are persisted into the database
const StatsSaveInterval = 1 * time.Hour

const statsSaveMaxDither = 10 // seconds
var statsSaveDither time.Duration

func init() {
	rand.Seed(int64(time.Now().Nanosecond()))
	statsSaveDither = time.Duration(rand.Intn(statsSaveMaxDither)) * time.Second
}

type Stats struct {
	ticker *time.Ticker

	Jira             expvar.Service
	LegacyWebhook    expvar.AsyncService
	SubscribeWebhook expvar.AsyncService
}

func (s *Stats) init(currentInstanceStore CurrentInstanceStore, userStore UserStore) *Stats {
	s.Jira = expvar.NewService("jira")
	s.LegacyWebhook = expvar.NewAsyncService("webhook/legacy")
	s.SubscribeWebhook = expvar.NewAsyncService("webhook/subscribe")

	goexpvar.Publish("counters/mapped_users", goexpvar.Func(func() interface{} {
		ji, err := currentInstanceStore.LoadCurrentJIRAInstance()
		if err != nil {
			return err.Error()
		}
		count, err := userStore.CountUsers(ji)
		if err != nil {
			return err.Error()
		}
		return count
	}))
	return s
}

func NewStatsWithData(data []byte, currentInstanceStore CurrentInstanceStore, userStore UserStore) (*Stats, error) {
	s := &Stats{}
	err := json.Unmarshal(data, s)
	if err != nil {
		return nil, err
	}
	return s.init(currentInstanceStore, userStore), nil
}

func NewStats(currentInstanceStore CurrentInstanceStore, userStore UserStore) *Stats {
	return (&Stats{}).init(currentInstanceStore, userStore)
}
