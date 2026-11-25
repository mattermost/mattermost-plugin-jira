// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakeInstanceHTTPClientAppliesAllowListChange(t *testing.T) {
	var newConns, closedConns int32
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	srv.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		switch state {
		case http.StateNew:
			atomic.AddInt32(&newConns, 1)
		case http.StateClosed:
			atomic.AddInt32(&closedConns, 1)
		}
	}
	srv.Start()
	t.Cleanup(srv.Close)

	p, _ := setupInstallURLCheckPlugin(t, "127.0.0.0/8")
	// A non-instance host, as a redirect target would be.
	const instanceURL = "https://jira.example.com"

	get := func(c *http.Client) error {
		resp, err := c.Get(srv.URL)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		_, err = io.Copy(io.Discard, resp.Body)
		return err
	}

	oldClient := p.makeInstanceHTTPClient(instanceURL, 0)
	require.NoError(t, get(oldClient))
	require.NoError(t, get(oldClient))
	assert.Equal(t, int32(1), atomic.LoadInt32(&newConns), "the second request should reuse the pooled connection")
	assert.Same(t, oldClient.Transport, p.makeInstanceHTTPClient(instanceURL, 0).Transport, "an unchanged allow-list keeps the cached transport")

	*p.API.GetConfig().ServiceSettings.AllowedUntrustedInternalConnections = ""

	newClient := p.makeInstanceHTTPClient(instanceURL, 0)
	assert.NotSame(t, oldClient.Transport, newClient.Transport)
	assert.Same(t, newClient.Transport, p.makeInstanceHTTPClient(instanceURL, 0).Transport)

	assert.Eventually(t, func() bool { return atomic.LoadInt32(&closedConns) == 1 }, 2*time.Second, 10*time.Millisecond,
		"idle connections of the replaced transport should be closed")
	assert.Error(t, get(oldClient), "the old transport must not reuse a pooled connection to a host that is no longer allowed")
	assert.Error(t, get(newClient))
	assert.Equal(t, int32(1), atomic.LoadInt32(&newConns))
}
