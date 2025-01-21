// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package utils

import (
	"net/http"
)

type transport struct {
	RequestSizeLimit  ByteSize
	ResponseSizeLimit ByteSize
	RequestPreClose   func(*LimitedReadCloser) error
	ResponsePreClose  func(*LimitedReadCloser) error
	http.RoundTripper
}

func WithRequestSizeLimit(limit ByteSize) func(t *transport) {
	return func(t *transport) {
		t.RequestSizeLimit = limit
	}
}

func WithResponseSizeLimit(limit ByteSize) func(t *transport) {
	return func(t *transport) {
		t.ResponseSizeLimit = limit
	}
}

func WithRequestPreClose(preCloseF func(*LimitedReadCloser) error) func(t *transport) {
	return func(t *transport) {
		t.RequestPreClose = preCloseF
	}
}

func WithResponsePreClose(preCloseF func(*LimitedReadCloser) error) func(t *transport) {
	return func(t *transport) {
		t.ResponsePreClose = preCloseF
	}
}

// WrapHTTPClient wraps an http client's request and response with LimitedReadCloser's
func WrapHTTPClient(c *http.Client, optFuncs ...func(t *transport)) *http.Client {
	client := *c

	underlyingT := c.Transport
	if underlyingT == nil {
		underlyingT = http.DefaultTransport
	}
	t := &transport{
		RoundTripper:      underlyingT,
		RequestSizeLimit:  -1,
		ResponseSizeLimit: -1,
	}
	for _, optf := range optFuncs {
		if optf == nil {
			continue
		}
		optf(t)
	}
	client.Transport = t
	return &client
}

func (transport *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Wrap the request body, **only** if it's there
	if req.Body != nil {
		req.Body = NewLimitedReadCloser(req.Body, transport.RequestSizeLimit, transport.RequestPreClose)
	}

	resp, err := transport.RoundTripper.RoundTrip(req)
	if resp != nil && resp.Body != nil {
		resp.Body = NewLimitedReadCloser(resp.Body, transport.ResponseSizeLimit, transport.ResponsePreClose)
	}
	return resp, err
}
