package expvar

import (
	"encoding/json"
	"expvar"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-jira/server/utils"
)

func newTestEndpoint(t testing.TB) *Endpoint {
	name := fmt.Sprintf("test_%v", uuid.New().String())
	e := NewEndpoint(name)
	require.NotNil(t, e)
	return e
}

func TestNewEndpoint(t *testing.T) {
	e := newTestEndpoint(t)
	p := expvar.Get(e.Name)
	require.Equal(t, e, p)
	require.Equal(t, `{}`, e.String())
}

func TestEndpointNilString(t *testing.T) {
	var e *Endpoint
	require.Equal(t, `{}`, e.String())
}

func TestEndpointRecord(t *testing.T) {
	e := newTestEndpoint(t)
	recordSampleToEndpoint(t, e)
	checkSampleRecorded(t, e)
}

func TestEndpointUnmarshal(t *testing.T) {
	e := newTestEndpoint(t)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < 10; i++ {
		e.Record(utils.ByteSize(r.Intn(1000*1000*1000)),
			utils.ByteSize(r.Intn(2*1000*1000)),
			time.Duration(r.Intn(1000*1000*1000)),
			false, false)
	}
	estr := e.String()

	for i := 0; i < 10; i++ {
		data, err := e.MarshalJSON()
		require.Nil(t, err)
		err = json.Unmarshal(data, &e)
		require.Nil(t, err)

		// The stored data gets re-binned every time and changes, so the next
		// line would usually fail.
		// assert.Equal(t, edata, jsonString(t, e))

		assert.Equal(t, estr, e.String())
	}
}

var sample = []struct {
	requestSize  int64
	responseSize int64
	elapsed      int // seconds
	isError      bool
	isIgnored    bool
}{
	{97532457, 100, 100, false, false},
	{436789, 120, 110, false, false},
	{33700, 1000, 130, false, false},
	{32550, 1200, 105, false, true},
	{64000, 1200, 105, false, true},
	{57000, 5000, 215, true, false},
	{12100, 6000, 375, false, true},
	{12900, 3250, 145, false, false},
	{10800, 33700, 198, false, false},
	{12000, 348000, 271, false, false},
	{10500, 45603123, 321, false, false},
}

const sampleErrors = int64(1)
const sampleIgnored = int64(3)
const sampleTotal = int64(11)
const sampleElapsed = `"AGQKCAADCwgAAQ0IAAEOCAABEwgAARUIAAEbCAABIAgAASUIAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"`
const sampleRequestSize = `"AGQKAgABDAIAAQoDAAEMAwACIAMAATIDAAE8AwABIQQAASIFAAEtBwABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"`
const sampleResponseSize = `""`
const sampleJSON = `{"Elapsed":{"P10":"103.666666ms","P50":"145ms","P85":"323.5ms","P95":"374.5ms","P98":"377.8ms","P99":"378.9ms"},"Errors":1,"Ignored":3,"Size":{"P10":"121b","P50":"3.2Kb","P85":"335.4Kb","P95":"43.3Mb","P98":"43.7Mb","P99":"43.8Mb"},"Total":11}`

func recordSampleToEndpoint(t testing.TB, e *Endpoint) {
	for _, s := range sample {
		e.Record(utils.ByteSize(s.requestSize), utils.ByteSize(s.responseSize),
			time.Duration(s.elapsed)*time.Millisecond, s.isError, s.isIgnored)
	}
}

func checkSample(t testing.TB, e *Endpoint) {
	require.Equal(t, sampleJSON, e.String())
	require.Equal(t, sampleErrors, e.Errors)
	require.Equal(t, sampleIgnored, e.Ignored)
	require.Equal(t, sampleTotal, e.Total)
}

func checkSampleRecorded(t testing.TB, e *Endpoint) {
	checkSample(t, e)
	require.Equal(t, sampleElapsed, jsonString(t, e.Elapsed))
	require.Equal(t, sampleRequestSize, jsonString(t, e.RequestSize))
	require.Equal(t, sampleResponseSize, jsonString(t, e.ResponseSize))
}

func jsonString(t testing.TB, v interface{}) string {
	bb, err := json.Marshal(v)
	require.Nil(t, err)
	return string(bb)
}
