package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest"

	"github.com/mattermost/mattermost-plugin-jira/server/expvar"
)

func TestConsolidatedStoredStats(t *testing.T) {
	p := &Plugin{}
	api := &plugintest.API{}

	stats1 := expvar.NewUnpublishedStats(nil)
	e1 := stats1.EnsureEndpoint("e1")
	e1.Record(100, 10, 1*time.Second, false, false)
	e1.Record(100, 10, 2*time.Second, false, false)
	e1.Record(100, 10, 10*time.Second, false, false)
	e1.Record(10, 100, 14*time.Second, true, false)
	e1.Record(1, 1, 5*time.Second, false, true)

	e2 := stats1.EnsureEndpoint("e2")
	e2.Record(200, 20, 20*time.Second, false, false)
	e2.Record(220, 22, 21*time.Second, false, false)
	e2.Record(230, 23, 22*time.Second, false, false)
	e2.Record(21, 201, 30*time.Second, true, false)
	e2.Record(2, 2, 40*time.Second, false, false)

	stats1Data, err := json.Marshal(stats1)
	require.Nil(t, err)

	stats2 := expvar.NewUnpublishedStats(nil)
	e1 = stats2.EnsureEndpoint("e1")
	e1.Record(1000, 100, 17*time.Second, false, false)
	e1.Record(1000, 100, 17*time.Second, false, false)
	e1.Record(1000, 100, 20*time.Second, false, false)
	e1.Record(100, 1000, 100*time.Second, true, false)
	e1.Record(10, 10, 50*time.Second, false, true)

	e2 = stats2.EnsureEndpoint("e2")
	e2.Record(2000, 200, 200*time.Second, false, false)
	e2.Record(2200, 220, 210*time.Second, false, false)
	e2.Record(2300, 230, 220*time.Second, false, false)
	e2.Record(210, 2010, 300*time.Second, true, false)
	e2.Record(20, 20, 400*time.Second, false, false)

	e3 := stats2.EnsureEndpoint("e3")
	e3.Record(2000, 200, 200*time.Second, false, false)
	e3.Record(2200, 220, 210*time.Second, false, false)
	e3.Record(2300, 230, 220*time.Second, false, false)
	e3.Record(210, 2010, 300*time.Second, true, false)
	e3.Record(20, 20, 400*time.Second, false, false)

	stats2Data, err := json.Marshal(stats2)
	require.Nil(t, err)

	api.On("KVList", 0, 2).Return([]string{
		"key1",
		"stats_hostname1",
	}, (*model.AppError)(nil))
	api.On("KVList", 1, 2).Return([]string{
		"key2",
		"stats_hostname2",
	}, (*model.AppError)(nil))
	api.On("KVList", 2, 2).Return([]string{
		"key3",
		"key4",
	}, (*model.AppError)(nil))
	api.On("KVList", 3, 2).Return([]string{}, (*model.AppError)(nil))

	api.On("KVGet", "stats_hostname1").Return(stats1Data, (*model.AppError)(nil))
	api.On("KVGet", "stats_hostname2").Return(stats2Data, (*model.AppError)(nil))

	p.SetAPI(api)

	listPerPage = 2
	consolidated, keys, _ := p.consolidatedStoredStats()
	require.Equal(t, []string{"stats_hostname1", "stats_hostname2"}, keys)

	// Calculate the "expected" merge result
	stats := expvar.NewUnpublishedStats(nil)
	stats.Merge(stats1, stats2)
	require.Equal(t, `{"Elapsed":{"P10":"1.1s","P50":"15s","P85":"50.5s","P95":"1m45s","P98":"1m48s","P99":"1m49s"},"Errors":2,"Ignored":2,"RequestSize":{"P10":"1b","P50":"105b","P85":"1Kb","P95":"1.1Kb","P98":"1.1Kb","P99":"1.1Kb"},"ResponseSize":{"P10":"1b","P50":"11b","P85":"108b","P95":"1Kb","P98":"1.1Kb","P99":"1.1Kb"},"Total":10}`,
		stats.EnsureEndpoint("e1").String())
	require.Equal(t, `{"Elapsed":{"P10":"21s","P50":"41s","P85":"5m5s","P95":"6m45s","P98":"6m48s","P99":"6m49s"},"Errors":2,"RequestSize":{"P10":"2b","P50":"220b","P85":"2.2Kb","P95":"2.3Kb","P98":"2.3Kb","P99":"2.3Kb"},"ResponseSize":{"P10":"2b","P50":"24b","P85":"234b","P95":"2Kb","P98":"2Kb","P99":"2Kb"},"Total":10}`,
		stats.EnsureEndpoint("e2").String())
	require.Equal(t, `{"Elapsed":{"P10":"3m25s","P50":"3m45s","P85":"6m42.5s","P95":"6m47.5s","P98":"6m49s","P99":"6m49.5s"},"Errors":1,"RequestSize":{"P10":"20b","P50":"2Kb","P85":"2.3Kb","P95":"2.3Kb","P98":"2.3Kb","P99":"2.3Kb"},"ResponseSize":{"P10":"20b","P50":"225b","P85":"2Kb","P95":"2Kb","P98":"2Kb","P99":"2Kb"},"Total":5}`,
		stats.EnsureEndpoint("e3").String())

	require.Equal(t, `{"Elapsed":{"P10":"1.1s","P50":"15s","P85":"50.5s","P95":"1m45s","P98":"1m48s","P99":"1m49s"},"Errors":2,"Ignored":2,"RequestSize":{"P10":"1b","P50":"105b","P85":"1Kb","P95":"1.1Kb","P98":"1.1Kb","P99":"1.1Kb"},"ResponseSize":{"P10":"1b","P50":"11b","P85":"108b","P95":"1Kb","P98":"1.1Kb","P99":"1.1Kb"},"Total":10}`,
		consolidated.EnsureEndpoint("e1").String())
	require.Equal(t, `{"Elapsed":{"P10":"21s","P50":"41s","P85":"5m5s","P95":"6m45s","P98":"6m48s","P99":"6m49s"},"Errors":2,"RequestSize":{"P10":"2b","P50":"220b","P85":"2.2Kb","P95":"2.3Kb","P98":"2.3Kb","P99":"2.3Kb"},"ResponseSize":{"P10":"2b","P50":"24b","P85":"234b","P95":"2Kb","P98":"2Kb","P99":"2Kb"},"Total":10}`,
		consolidated.EnsureEndpoint("e2").String())
	require.Equal(t, `{"Elapsed":{"P10":"3m25s","P50":"3m45s","P85":"6m42.5s","P95":"6m47.5s","P98":"6m49s","P99":"6m49.5s"},"Errors":1,"RequestSize":{"P10":"20b","P50":"2Kb","P85":"2.3Kb","P95":"2.3Kb","P98":"2.3Kb","P99":"2.3Kb"},"ResponseSize":{"P10":"20b","P50":"225b","P85":"2Kb","P95":"2Kb","P98":"2Kb","P99":"2Kb"},"Total":5}`,
		consolidated.EnsureEndpoint("e3").String())
}
