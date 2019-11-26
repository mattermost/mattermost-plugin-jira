package expvar

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/mattermost/mattermost-plugin-jira/server/utils"
	"github.com/stretchr/testify/require"
)

func TestStatsBasic(t *testing.T) {
	stats := NewUnpublishedStats(nil)

	i, ok := stats.endpoints.Load("xxx")
	require.False(t, ok)
	require.Nil(t, i)

	e := stats.EnsureEndpoint("xxx")

	i, ok = stats.endpoints.Load("xxx")
	require.True(t, ok)
	require.NotNil(t, i)
	require.Equal(t, i, e)

	require.Equal(t, e, stats.EnsureEndpoint("xxx"), "same value second time")

	stats.EnsureEndpoint("xxx").Record(10, 10, 300*time.Millisecond, false, false)
	require.Equal(t,
		`{"Elapsed":{"P10":"301ms","P50":"305ms","P85":"308.5ms","P95":"309.5ms","P98":"309.8ms","P99":"309.9ms"},"Errors":0,"RequestSize":{"P10":"10b","P50":"10b","P85":"10b","P95":"10b","P98":"10b","P99":"10b"},"ResponseSize":{"P10":"10b","P50":"10b","P85":"10b","P95":"10b","P98":"10b","P99":"10b"},"Total":1}`,
		stats.EnsureEndpoint("xxx").String())
}

func TestStats(t *testing.T) {
	stats := NewUnpublishedStats(nil)
	for _, s := range sample {
		stats.EnsureEndpoint("myapi1").Record(utils.ByteSize(s.requestSize), utils.ByteSize(s.responseSize),
			time.Duration(s.elapsed)*time.Millisecond, s.isError, s.isIgnored)
		stats.EnsureEndpoint("myapi2").Record(utils.ByteSize(s.requestSize), utils.ByteSize(s.responseSize),
			time.Duration(s.elapsed)*time.Millisecond, s.isError, s.isIgnored)
		stats.EnsureEndpoint("myapi3").Record(utils.ByteSize(s.requestSize), utils.ByteSize(s.responseSize),
			time.Duration(s.elapsed)*time.Millisecond, s.isError, s.isIgnored)
	}
	checkSample(t, stats.EnsureEndpoint("myapi1"))
	checkSample(t, stats.EnsureEndpoint("myapi2"))
	checkSample(t, stats.EnsureEndpoint("myapi3"))

	data, err := json.Marshal(stats)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	stats = NewUnpublishedStats(data)
	checkSample(t, stats.EnsureEndpoint("myapi1"))
	checkSample(t, stats.EnsureEndpoint("myapi2"))
	checkSample(t, stats.EnsureEndpoint("myapi3"))
}

func TestStatsMerge(t *testing.T) {
	stats1 := NewUnpublishedStats(nil)
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

	stats2 := NewUnpublishedStats(nil)
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

	stats := NewUnpublishedStats(nil)
	stats.Merge(stats1, stats2)
	e1 = stats.EnsureEndpoint("e1")
	e2 = stats.EnsureEndpoint("e2")
	require.Equal(t, `{"Elapsed":{"P10":"1.1s","P50":"15s","P85":"50.5s","P95":"1m45s","P98":"1m48s","P99":"1m49s"},"Errors":2,"Ignored":2,"RequestSize":{"P10":"1b","P50":"105b","P85":"1Kb","P95":"1.1Kb","P98":"1.1Kb","P99":"1.1Kb"},"ResponseSize":{"P10":"1b","P50":"11b","P85":"108b","P95":"1Kb","P98":"1.1Kb","P99":"1.1Kb"},"Total":10}`, e1.String())
	require.Equal(t, `{"Elapsed":{"P10":"21s","P50":"41s","P85":"5m5s","P95":"6m45s","P98":"6m48s","P99":"6m49s"},"Errors":2,"RequestSize":{"P10":"2b","P50":"220b","P85":"2.2Kb","P95":"2.3Kb","P98":"2.3Kb","P99":"2.3Kb"},"ResponseSize":{"P10":"2b","P50":"24b","P85":"234b","P95":"2Kb","P98":"2Kb","P99":"2Kb"},"Total":10}`, e2.String())
}
