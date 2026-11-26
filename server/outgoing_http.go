// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/httpservice"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/utils"
)

const untrustedHTTPClientTimeout = 30 * time.Second

// outgoingHTTP holds the server's outgoing connection settings and the
// transports built from them. It is replaced as a whole when the settings
// change, so pooled connections never outlive the settings they were made
// under.
type outgoingHTTP struct {
	insecure           bool
	allowedConnections []string

	mu sync.Mutex
	// Transports for installed instances, keyed by instance host.
	transports map[string]*http.Transport
}

// newOutgoingHTTP reads the settings from mmConfig. A nil config yields an
// empty allow list, so internal destinations stay blocked.
func newOutgoingHTTP(mmConfig *model.Config) *outgoingHTTP {
	o := &outgoingHTTP{transports: map[string]*http.Transport{}}
	if mmConfig == nil {
		return o
	}
	settings := mmConfig.ServiceSettings
	o.insecure = settings.EnableInsecureOutgoingConnections != nil && *settings.EnableInsecureOutgoingConnections
	if settings.AllowedUntrustedInternalConnections != nil {
		o.allowedConnections = strings.FieldsFunc(*settings.AllowedUntrustedInternalConnections, func(c rune) bool {
			return unicode.IsSpace(c) || c == ','
		})
	}
	return o
}

func (o *outgoingHTTP) sameSettings(other *outgoingHTTP) bool {
	return o.insecure == other.insecure && slices.Equal(o.allowedConnections, other.allowedConnections)
}

// restrictedTransport rejects reserved and own IPs at dial time, which also
// covers every redirect hop, unless they are in allowedConnections. trustedHost
// is dialed without the IP check.
func (o *outgoingHTTP) restrictedTransport(trustedHost string) *http.Transport {
	allowHost := func(host string) bool {
		if trustedHost != "" && strings.EqualFold(host, trustedHost) {
			return true
		}
		return slices.Contains(o.allowedConnections, host)
	}

	allowIP := func(ip net.IP) error {
		ownIP, err := httpservice.IsOwnIP(ip)
		if err != nil {
			return errors.New("unable to determine if IP is own IP")
		}
		if !httpservice.IsReservedIP(ip) && !ownIP {
			return nil
		}
		for _, allowed := range o.allowedConnections {
			if _, ipRange, err := net.ParseCIDR(allowed); err == nil && ipRange.Contains(ip) {
				return nil
			}
		}
		return errors.New("IP is reserved or own IP and is not in AllowedUntrustedInternalConnections")
	}

	// The underlying transport is used directly so that the User-Agent set by
	// callers is not replaced.
	return httpservice.NewTransport(o.insecure, allowHost, allowIP).Transport.(*http.Transport)
}

func (o *outgoingHTTP) instanceTransport(host string) *http.Transport {
	o.mu.Lock()
	defer o.mu.Unlock()
	t, ok := o.transports[host]
	if !ok {
		t = o.restrictedTransport(host)
		o.transports[host] = t
	}
	return t
}

func (o *outgoingHTTP) closeIdleConnections() {
	o.mu.Lock()
	defer o.mu.Unlock()
	for _, t := range o.transports {
		t.CloseIdleConnections()
	}
}

func (p *Plugin) updateOutgoingHTTP(mmConfig *model.Config) {
	next := newOutgoingHTTP(mmConfig)
	if prev := p.outgoingHTTP.Load(); prev != nil && prev.sameSettings(next) {
		return
	}
	if prev := p.outgoingHTTP.Swap(next); prev != nil {
		prev.closeIdleConnections()
	}
}

func (p *Plugin) getOutgoingHTTP() *outgoingHTTP {
	if o := p.outgoingHTTP.Load(); o != nil {
		return o
	}
	return newOutgoingHTTP(nil)
}

// makeUntrustedHTTPClient returns a client for requests to admin- or
// user-supplied URLs that have not been installed as an instance yet.
func (p *Plugin) makeUntrustedHTTPClient() *http.Client {
	return &http.Client{
		Transport: p.getOutgoingHTTP().restrictedTransport(""),
		Timeout:   untrustedHTTPClientTimeout,
	}
}

// makeInstanceHTTPClient returns a client for requests to an installed
// instance. The instance's own host was chosen by an admin and is trusted even
// if it is internal; any other host, such as a redirect target, gets the same
// checks as makeUntrustedHTTPClient. A zero timeout means no timeout.
func (p *Plugin) makeInstanceHTTPClient(instanceURL string, timeout time.Duration) *http.Client {
	return &http.Client{
		Transport: p.getOutgoingHTTP().instanceTransport(jiraURLHost(instanceURL)),
		Timeout:   timeout,
	}
}

func redactURLForLog(rawURL string) string {
	normalized, err := utils.NormalizeJiraURL(rawURL)
	if err != nil {
		return ""
	}
	u, err := url.Parse(normalized)
	if err != nil {
		return ""
	}
	u.User = nil
	return u.String()
}

func (p *Plugin) checkJiraURL(rawURL string) (string, error) {
	client := p.jiraURLCheckClientOverride
	if client == nil {
		client = p.makeUntrustedHTTPClient()
		defer client.CloseIdleConnections()
	}
	jiraURL, err := utils.CheckJiraURL(client, p.GetSiteURL(), rawURL, false)
	var checkErr *utils.JiraURLCheckError
	if errors.As(err, &checkErr) {
		p.client.Log.Warn("Failed to validate the connection to the Jira server", "jira_url", redactURLForLog(rawURL), "error", checkErr.Cause.Error())
	}
	return jiraURL, err
}
