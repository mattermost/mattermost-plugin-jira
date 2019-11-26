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

func TestEndpointMerge(t *testing.T) {
	e1 := NewUnpublishedEndpoint("e1")
	e1.Record(100, 10, 1*time.Second, false, false)
	e1.Record(100, 10, 1*time.Second, false, false)
	e1.Record(100, 10, 1*time.Second, false, false)
	e1.Record(10, 100, 10*time.Second, true, false)
	e1.Record(1, 1, 5*time.Second, false, true)

	e2 := NewUnpublishedEndpoint("e2")
	e2.Record(200, 20, 20*time.Second, false, false)
	e2.Record(220, 22, 21*time.Second, false, false)
	e2.Record(230, 23, 22*time.Second, false, false)
	e2.Record(21, 201, 30*time.Second, true, false)
	e2.Record(2, 2, 40*time.Second, false, false)

	e := NewUnpublishedEndpoint("e")
	require.Equal(t, `{}`, e.String())

	e.Merge(e1, e2)
	require.Equal(t, `{"Elapsed":{"P10":"1.033333333s","P50":"11s","P85":"30.5s","P95":"40.5s","P98":"40.8s","P99":"40.9s"},"Errors":2,"Ignored":1,"RequestSize":{"P10":"1b","P50":"103b","P85":"225b","P95":"234b","P98":"237b","P99":"238b"},"ResponseSize":{"P10":"1b","P50":"11b","P85":"105b","P95":"205b","P98":"208b","P99":"209b"},"Total":10}`, e.String())
}

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
const sampleRequestSize = `"AGQKBAACDAQAAyAEAAEhBAABOQQAAUAEAAErBQABYQcAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"`
const sampleResponseSize = `"AGQKAgABDAIAAQoDAAEMAwACIAMAATIDAAE8AwABIQQAASIFAAEtBwABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"`
const sampleJSON = `{"Elapsed":{"P10":"103.666666ms","P50":"145ms","P85":"323.5ms","P95":"374.5ms","P98":"377.8ms","P99":"378.9ms"},"Errors":1,"Ignored":3,"RequestSize":{"P10":"10.3Kb","P50":"31.7Kb","P85":"423.3Kb","P95":"92.9Mb","P98":"93.3Mb","P99":"93.4Mb"},"ResponseSize":{"P10":"121b","P50":"3.2Kb","P85":"335.4Kb","P95":"43.3Mb","P98":"43.7Mb","P99":"43.8Mb"},"Total":11}`

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
	require.Equal(t, sampleElapsed, jsonString(t, e.Elapsed), "Elapsed")
	require.Equal(t, sampleRequestSize, jsonString(t, e.RequestSize), "RequestSize")
	require.Equal(t, sampleResponseSize, jsonString(t, e.ResponseSize), "ResponseSize")
}

func jsonString(t testing.TB, v interface{}) string {
	bb, err := json.Marshal(v)
	require.Nil(t, err)
	return string(bb)
}
