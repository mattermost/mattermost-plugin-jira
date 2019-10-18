package expvar

import (
	"fmt"
	"net/http"
	"time"

	"github.com/mattermost/mattermost-plugin-jira/server/utils"
)

type roundtripper struct {
	http.RoundTripper
	requestMaxSize  utils.ByteSize
	responseMaxSize utils.ByteSize

	stats                   *Stats
	endpointNameFromRequest func(*http.Request) string
}

type roundtrip struct {
	roundtripper *roundtripper
	started      time.Time
	requestSize  utils.ByteSize
	endpoint     *Endpoint
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
	transport := c.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	client.Transport = &roundtripper{
		RoundTripper:            transport,
		requestMaxSize:          maxSize,
		responseMaxSize:         maxSize,
		stats:                   stats,
		endpointNameFromRequest: endpointNameFromRequest,
	}
	return &client
}

func (roundtripper *roundtripper) RoundTrip(req *http.Request) (*http.Response, error) {
	fmt.Println("<><> roundtripper 0")
	rt := &roundtrip{
		roundtripper: roundtripper,
		started:      time.Now(),
	}

	if roundtripper.stats != nil && roundtripper.endpointNameFromRequest != nil {
		endpointName := roundtripper.endpointNameFromRequest(req)
		rt.endpoint = roundtripper.stats.EnsureEndpoint(endpointName)
	}

	// Wrap the request body, **only** if it's there
	if req.Body != nil {
		fmt.Println("<><> roundtripper 1 - req.Body not nil")
		req.Body = &utils.LimitReadCloser{
			ReadCloser: req.Body,
			Limit:      roundtripper.requestMaxSize,
			OnClose:    rt.OnCloseRequest,
		}
	}

	resp, err := roundtripper.RoundTripper.RoundTrip(req)
	if err != nil || resp == nil || resp.Body == nil {
		if rt.endpoint != nil {
			rt.endpoint.Record(rt.requestSize, 0, time.Since(rt.started), true, false)
		}
		return resp, err
	}

	resp.Body = &utils.LimitReadCloser{
		ReadCloser: resp.Body,
		Limit:      roundtripper.requestMaxSize,
		OnClose:    rt.OnCloseResponse,
	}

	return resp, err
}

func (rt *roundtrip) OnCloseRequest(lr *utils.LimitReadCloser) error {
	rt.requestSize = lr.TotalRead
	return nil
}

func (rt *roundtrip) OnCloseResponse(lr *utils.LimitReadCloser) error {
	if rt.endpoint != nil {
		rt.endpoint.Record(rt.requestSize, lr.TotalRead, time.Since(rt.started), false, false)
	}
	return nil
}
