package expvar

import (
	"io"
	"net/http"
	"time"

	"github.com/mattermost/mattermost-plugin-jira/server/utils"
)

type roundtripper struct {
	http.RoundTripper
	limit                   utils.ByteSize
	endpointNameFromRequest func(*http.Request) string
	stats                   *Stats
}

type readCloser struct {
	inner     io.ReadCloser
	read      utils.ByteSize
	remaining utils.ByteSize
	start     time.Time
	endpoint  *Endpoint
}

func WrapHTTPClient(c *http.Client, limit utils.ByteSize, stats *Stats, endpointNameFromRequest func(*http.Request) string) *http.Client {
	return wrapHTTPClient(c, limit, stats, endpointNameFromRequest, false)
}

func wrapHTTPClient(c *http.Client, limit utils.ByteSize, stats *Stats, endpointNameFromRequest func(*http.Request) string, disableExpvars bool) *http.Client {
	client := *c
	rt := c.Transport
	if rt == nil {
		rt = http.DefaultTransport
	}
	client.Transport = roundtripper{
		RoundTripper:            rt,
		limit:                   limit,
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
		endpoint.Record(0, time.Since(start), true, false)
		return resp, err
	}

	resp.Body = &readCloser{
		inner:     resp.Body,
		remaining: rt.limit,
		start:     start,
		endpoint:  endpoint,
	}
	return resp, err
}

func (r *readCloser) Close() error {
	err := r.inner.Close()
	if r.endpoint != nil {
		r.endpoint.Record(r.read, time.Since(r.start), err != nil, false)
	}
	return err
}

func (r *readCloser) Read(data []byte) (int, error) {
	if r.remaining <= 0 {
		return 0, io.EOF
	}
	if len(data) > int(r.remaining) {
		data = data[0:r.remaining]
	}
	n, err := r.inner.Read(data)
	r.remaining -= utils.ByteSize(n)
	r.read += utils.ByteSize(n)
	return n, err
}
