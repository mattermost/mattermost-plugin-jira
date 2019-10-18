package expvar

import (
	"net/http"
	"time"

	"github.com/mattermost/mattermost-plugin-jira/server/utils"
)

type roundtripper struct {
	http.RoundTripper
	requestLimit  int64
	responseLimit int64

	stats                   *Stats
	endpointNameFromRequest func(*http.Request) string
}

type recorder struct {
	*utils.LimitReadCloser
	requestStarted time.Time
	requestSize    utils.ByteSize
	endpoint       *Endpoint
}

// WrapHTTPClient wraps an http  client, establishing limits for request and response sizes,
// and automating recording the stats. The metric name  is derived from the request by the
// endpointNameFromRequest function.
func WrapHTTPClient(c *http.Client, maxSize utils.ByteSize, stats *Stats, endpointNameFromRequest func(*http.Request) string) *http.Client {
	return wrapHTTPClient(c, maxSize, stats, endpointNameFromRequest, false)
}

func wrapHTTPClient(c *http.Client, maxSize utils.ByteSize, stats *Stats,
	endpointNameFromRequest func(*http.Request) string, disableExpvars bool) *http.Client {

	client := *c
	rt := c.Transport
	if rt == nil {
		rt = http.DefaultTransport
	}
	client.Transport = roundtripper{
		RoundTripper:            rt,
		requestLimit:            int64(maxSize),
		responseLimit:           int64(maxSize),
		stats:                   stats,
		endpointNameFromRequest: endpointNameFromRequest,
	}
	return &client
}

func (rt roundtripper) RoundTrip(r *http.Request) (*http.Response, error) {
	start := time.Now()
	var endpoint *Endpoint
	endpointName := ""

	if rt.stats != nil && rt.endpointNameFromRequest != nil {
		endpointName = rt.endpointNameFromRequest(r)
		endpoint = rt.stats.Endpoint(endpointName)
	}

	resp, err := rt.RoundTripper.RoundTrip(r)
	if err != nil || resp == nil || resp.Body == nil {
		rr := r.Body.(*utils.LimitReadCloser)
		endpoint.Record(rr.TotalRead, 0, time.Since(start), true, false)
		return resp, err
	}

	resp.Body = &recorder{
		LimitReadCloser: utils.NewLimitReadCloser(resp.Body, rt.responseLimit),
		requestStarted:  start,
		endpoint:        endpoint,
	}
	return resp, err
}

func (r *recorder) Close() error {
	err := r.LimitReadCloser.Close()
	if r.endpoint != nil {
		r.endpoint.Record(r.requestSize, r.LimitReadCloser.TotalRead, time.Since(r.requestStarted), err != nil, false)
	}
	return err
}
