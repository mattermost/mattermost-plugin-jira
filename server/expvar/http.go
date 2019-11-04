package expvar

import (
	"net/http"
	"time"

	"github.com/mattermost/mattermost-plugin-jira/server/utils"
)

type transport struct {
	http.RoundTripper
	stats                   *Stats
	endpointNameFromRequest func(*http.Request) string
}

type roundtrip struct {
	transport   *transport
	started     time.Time
	requestSize utils.ByteSize
	endpoint    *Endpoint
}

// WrapHTTPClient wraps an http  client, establishing limits for request and response sizes,
// and automating recording the stats. The metric name  is derived from the request by the
// endpointNameFromRequest function.
func WrapHTTPClient(c *http.Client, stats *Stats, endpointNameFromRequest func(*http.Request) string) *http.Client {
	client := *c
	t := c.Transport
	if t == nil {
		t = http.DefaultTransport
	}
	client.Transport = &transport{
		RoundTripper:            t,
		stats:                   stats,
		endpointNameFromRequest: endpointNameFromRequest,
	}
	return &client
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	rt := &roundtrip{
		transport: t,
		started:   time.Now(),
	}

	if t.stats != nil && t.endpointNameFromRequest != nil {
		endpointName := t.endpointNameFromRequest(req)
		rt.endpoint = t.stats.EnsureEndpoint(endpointName)
	}

	// Wrap the request body, **only** if it's there
	if req.Body != nil {
		req.Body = utils.NewLimitedReadCloser(req.Body, -1, rt.OnCloseRequest)
	}

	resp, err := t.RoundTripper.RoundTrip(req)
	if err != nil || resp == nil || resp.Body == nil {
		if rt.endpoint != nil {
			rt.endpoint.Record(rt.requestSize, 0, time.Since(rt.started), true, false)
		}
		return resp, err
	}

	resp.Body = utils.NewLimitedReadCloser(resp.Body, -1, rt.OnCloseResponse)
	return resp, err
}

func (rt *roundtrip) OnCloseRequest(lrc *utils.LimitedReadCloser) error {
	rt.requestSize = lrc.TotalRead
	return nil
}

func (rt *roundtrip) OnCloseResponse(lrc *utils.LimitedReadCloser) error {
	if rt.endpoint != nil {
		rt.endpoint.Record(rt.requestSize, lrc.TotalRead, time.Since(rt.started), false, false)
	}
	return nil
}
