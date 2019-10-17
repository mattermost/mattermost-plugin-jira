package expvar

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/mattermost/mattermost-plugin-jira/server/utils"
	"github.com/stretchr/testify/require"
)

func TestStats(t *testing.T) {
	stats := newStats(nil, true)
	for _, s := range sample {
		stats.Endpoint("myapi1").Record(utils.ByteSize(s.requestSize), utils.ByteSize(s.responseSize),
			time.Duration(s.elapsed)*time.Millisecond, s.isError, s.isIgnored)
		stats.Endpoint("myapi2").Record(utils.ByteSize(s.requestSize), utils.ByteSize(s.responseSize),
			time.Duration(s.elapsed)*time.Millisecond, s.isError, s.isIgnored)
		stats.Endpoint("myapi3").Record(utils.ByteSize(s.requestSize), utils.ByteSize(s.responseSize),
			time.Duration(s.elapsed)*time.Millisecond, s.isError, s.isIgnored)
	}
	checkSample(t, stats.Endpoint("myapi1"))
	checkSample(t, stats.Endpoint("myapi2"))
	checkSample(t, stats.Endpoint("myapi3"))

	data, err := json.Marshal(stats)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	stats = newStats(data, true)
	checkSample(t, stats.Endpoint("myapi1"))
	checkSample(t, stats.Endpoint("myapi2"))
	checkSample(t, stats.Endpoint("myapi3"))
}
