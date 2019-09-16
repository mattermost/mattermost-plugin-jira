package expvar

import (
	"io"
	"net/http"
	"time"

	"github.com/mattermost/mattermost-plugin-jira/server/utils"
)

type roundtripper struct {
	http.RoundTripper
	limit               utils.ByteSize
	stats               *Service
	endpointFromRequest func(*http.Request) string
}

type readCloser struct {
	inner     io.ReadCloser
	read      utils.ByteSize
	remaining utils.ByteSize
	start     time.Time
	name      string
	stats     *Service
}

func WrapClient(c *http.Client, limit utils.ByteSize, stats *Service, endpointFromRequest func(*http.Request) string) *http.Client {
	client := *c
	rt := c.Transport
	if rt == nil {
		rt = http.DefaultTransport
	}
	client.Transport = roundtripper{
		RoundTripper:        rt,
		limit:               limit,
		stats:               stats,
		endpointFromRequest: endpointFromRequest,
	}
	return &client
}

func (rt roundtripper) RoundTrip(r *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := rt.RoundTripper.RoundTrip(r)
	if err != nil || resp == nil || resp.Body == nil {
		// TODO report stats
		return resp, err
	}
	resp.Body = &readCloser{
		inner:     resp.Body,
		remaining: rt.limit,
		start:     start,
		name:      rt.endpointFromRequest(r),
		stats:     rt.stats,
	}
	return resp, err
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

func (r *readCloser) Close() error {

	err := r.inner.Close()
	if r.stats != nil {
		r.stats.Response(r.name, r.read, time.Since(r.start), err != nil, false)
	}
	return err
}
