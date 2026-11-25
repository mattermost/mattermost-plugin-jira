// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/mattermost/mattermost/server/public/shared/httpservice"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/utils"
)

const untrustedHTTPClientTimeout = 30 * time.Second

func splitAllowedConnections(c rune) bool {
	return unicode.IsSpace(c) || c == ','
}

// allowedUntrustedInternalConnections is read on every call so that changes to
// the server config apply without a plugin restart. A missing config yields an
// empty list, so internal destinations stay blocked.
func (p *Plugin) allowedUntrustedInternalConnections() []string {
	mmConfig := p.API.GetConfig()
	if mmConfig == nil || mmConfig.ServiceSettings.AllowedUntrustedInternalConnections == nil {
		return nil
	}
	return strings.FieldsFunc(*mmConfig.ServiceSettings.AllowedUntrustedInternalConnections, splitAllowedConnections)
}

func (p *Plugin) insecureOutgoingConnectionsEnabled() bool {
	mmConfig := p.API.GetConfig()
	return mmConfig != nil && mmConfig.ServiceSettings.EnableInsecureOutgoingConnections != nil &&
		*mmConfig.ServiceSettings.EnableInsecureOutgoingConnections
}

// makeRestrictedTransport rejects reserved and own IPs at dial time, which
// also covers every redirect hop, unless they are allowed by the server's
// AllowedUntrustedInternalConnections setting. Hosts in trustedHosts are
// dialed without the IP check.
func (p *Plugin) makeRestrictedTransport(insecure bool, trustedHosts ...string) http.RoundTripper {
	allowHost := func(host string) bool {
		for _, trusted := range trustedHosts {
			if trusted != "" && strings.EqualFold(host, trusted) {
				return true
			}
		}
		for _, allowed := range p.allowedUntrustedInternalConnections() {
			if host == allowed {
				return true
			}
		}
		return false
	}

	allowIP := func(ip net.IP) error {
		ownIP, err := httpservice.IsOwnIP(ip)
		if err != nil {
			return errors.New("unable to determine if IP is own IP")
		}
		if !httpservice.IsReservedIP(ip) && !ownIP {
			return nil
		}
		for _, allowed := range p.allowedUntrustedInternalConnections() {
			if _, ipRange, err := net.ParseCIDR(allowed); err == nil && ipRange.Contains(ip) {
				return nil
			}
		}
		return errors.New("IP is reserved or own IP and is not in AllowedUntrustedInternalConnections")
	}

	// The underlying transport is used directly so that the User-Agent set by
	// callers is not replaced.
	return httpservice.NewTransport(insecure, allowHost, allowIP).Transport
}

// makeUntrustedHTTPClient returns a client for requests to admin- or
// user-supplied URLs that have not been installed as an instance yet.
func (p *Plugin) makeUntrustedHTTPClient() *http.Client {
	return &http.Client{
		Transport: p.makeRestrictedTransport(p.insecureOutgoingConnectionsEnabled()),
		Timeout:   untrustedHTTPClientTimeout,
	}
}

// instanceTransport is a cached transport together with the allow-list it
// was created under.
type instanceTransport struct {
	http.RoundTripper
	allowList string
}

// makeInstanceHTTPClient returns a client for requests to an installed
// instance. The instance's own host was chosen by an admin and is trusted even
// if it is internal; any other host, such as a redirect target, gets the same
// checks as makeUntrustedHTTPClient. A zero timeout means no timeout.
func (p *Plugin) makeInstanceHTTPClient(instanceURL string, timeout time.Duration) *http.Client {
	host := jiraURLHost(instanceURL)
	insecure := p.insecureOutgoingConnectionsEnabled()
	allowList := strings.Join(p.allowedUntrustedInternalConnections(), " ")
	// Transports are cached so that connections to an instance are reused
	// across clients. A changed allow-list replaces the transport, because
	// pooled idle connections would otherwise bypass the dial-time checks.
	key := strconv.FormatBool(insecure) + "|" + host
	var transport http.RoundTripper
	for transport == nil {
		cached, ok := p.instanceTransports.Load(key)
		if ok && cached.(*instanceTransport).allowList == allowList {
			transport = cached.(*instanceTransport).RoundTripper
			break
		}
		fresh := &instanceTransport{allowList: allowList, RoundTripper: p.makeRestrictedTransport(insecure, host)}
		if !ok {
			if _, loaded := p.instanceTransports.LoadOrStore(key, fresh); !loaded {
				transport = fresh.RoundTripper
			}
			continue
		}
		if p.instanceTransports.CompareAndSwap(key, cached, fresh) {
			if closer, ok := cached.(*instanceTransport).RoundTripper.(interface{ CloseIdleConnections() }); ok {
				closer.CloseIdleConnections()
			}
			transport = fresh.RoundTripper
		}
	}
	return &http.Client{
		Transport: transport,
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
