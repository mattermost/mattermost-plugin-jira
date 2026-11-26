// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-jira/server/utils"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

type singleInstanceStore struct {
	recordingInstanceStore
	instance Instance
}

func (s *singleInstanceStore) LoadInstance(types.ID) (Instance, error) {
	return s.instance, nil
}

// newLocalhostRedirector stands in for a public host the admin controls: it is
// trusted by hostname ("localhost" in AllowedUntrustedInternalConnections) and
// redirects every request to target, keeping the method and path.
func newLocalhostRedirector(t *testing.T, target string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target+r.URL.Path, http.StatusTemporaryRedirect) //nolint:gosec // test fixture: the redirect target is a local test server
	}))
	t.Cleanup(srv.Close)
	return strings.Replace(srv.URL, "127.0.0.1", "localhost", 1)
}

func newServerInstanceForURLTest(t *testing.T, p *Plugin, jiraURL string) *serverInstance {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	p.updateConfig(func(conf *config) {
		conf.rsaKey = key
		conf.maxAttachmentSize = defaultMaxAttachmentSize
	})
	return &serverInstance{
		InstanceCommon: newInstanceCommon(p, ServerInstanceType, types.ID(jiraURL)),
		MattermostKey:  "mattermost-key",
	}
}

// Requests made to an installed server instance after the install-time check
// must get the same protection: a host that passed the check (or an instance
// stored before it existed) must not be able to steer the server to internal
// addresses.
func TestServerInstanceRequestsBlockInternalAddresses(t *testing.T) {
	entryPoints := map[string]func(si *serverInstance) error{
		"user connect OAuth1 request token": func(si *serverInstance) error {
			_, _, err := si.GetUserConnectURL("user1")
			return err
		},
		"Jira REST client": func(si *serverInstance) error {
			client, err := si.GetClient(&Connection{Oauth1AccessToken: "token", Oauth1AccessSecret: "secret"})
			if err != nil {
				return err
			}
			_, err = client.GetSelf()
			return err
		},
	}

	for _, via := range []string{"redirect from trusted host", "instance URL is internal"} {
		for name, call := range entryPoints {
			t.Run(via+"/"+name, func(t *testing.T) {
				var hits int32
				target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					atomic.AddInt32(&hits, 1)
					w.WriteHeader(http.StatusOK)
				}))
				t.Cleanup(target.Close)

				p, _ := setupInstallURLCheckPlugin(t, "localhost")
				instanceURL := target.URL
				if via == "redirect from trusted host" {
					instanceURL = newLocalhostRedirector(t, target.URL)
				}
				si := newServerInstanceForURLTest(t, p, instanceURL)

				err := call(si)

				if via == "instance URL is internal" {
					// Accepted behaviour: the host of an installed instance was
					// chosen by an admin and is trusted even when it is internal.
					assert.Equal(t, int32(1), atomic.LoadInt32(&hits), "request to the instance's own host must be allowed")
					return
				}
				assert.Error(t, err)
				assert.Equal(t, int32(0), atomic.LoadInt32(&hits), "request to an untrusted internal address must be blocked")
			})
		}
	}
}

func TestUserConnectErrorDoesNotLeakInternalTargetDetails(t *testing.T) {
	closedSrv := httptest.NewServer(http.NotFoundHandler())
	closedURL := closedSrv.URL
	closedSrv.Close()
	closedPort := closedURL[strings.LastIndex(closedURL, ":")+1:]

	notFoundSrv := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(notFoundSrv.Close)
	notFoundPort := notFoundSrv.URL[strings.LastIndex(notFoundSrv.URL, ":")+1:]

	for name, tc := range map[string]struct {
		target    string
		forbidden []string
	}{
		"redirect to closed internal port": {
			target:    closedURL,
			forbidden: []string{"refused", "dial", "127.0.0.1", closedPort},
		},
		"redirect to internal port answering 404": {
			target:    notFoundSrv.URL,
			forbidden: []string{"404", "127.0.0.1", notFoundPort},
		},
	} {
		t.Run(name, func(t *testing.T) {
			p, _ := setupInstallURLCheckPlugin(t, "localhost")
			si := newServerInstanceForURLTest(t, p, newLocalhostRedirector(t, tc.target))
			p.instanceStore = &singleInstanceStore{instance: si}
			p.userStore = mockUserStore{}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/instance/x/user/connect", nil)
			r.Header.Set("Mattermost-User-Id", "user1")

			status, err := p.httpUserConnect(w, r, si.GetID())

			require.Error(t, err)
			assert.NotEqual(t, http.StatusFound, status)
			body := strings.ToLower(w.Body.String())
			for _, s := range tc.forbidden {
				assert.NotContains(t, body, strings.ToLower(s))
			}
		})
	}
}

// The cloud-oauth install only accepts *.atlassian.net, so it must not send
// any request to a URL that can never be accepted.
func TestInstallCloudOAuthInstanceDoesNotProbeNonCloudURL(t *testing.T) {
	var hits int32
	srv := newStatusServer(t, http.StatusOK, `{"state":"RUNNING"}`, &hits)
	p, store := setupInstallURLCheckPlugin(t, "127.0.0.0/8")

	_, _, err := p.installCloudOAuthInstance(srv.URL)

	require.Error(t, err)
	assert.Equal(t, int32(0), atomic.LoadInt32(&hits), "no request may be sent to a non-Atlassian host")
	instancesStored, _ := store.writes()
	assert.Equal(t, 0, instancesStored)
}

func TestCheckJiraURLBlocksReservedAddressForms(t *testing.T) {
	for name, rawURL := range map[string]string{ //nolint:gosec // test fixture: fake userinfo checks that credentials in the URL don't bypass the IP filter
		"IPv6 loopback":           "http://[::1]:8065",
		"IPv4-mapped IPv6":        "http://[::ffff:127.0.0.1]:8065",
		"unspecified IPv4":        "http://0.0.0.0:8065",
		"cloud metadata":          "http://169.254.169.254",
		"localhost hostname":      "http://localhost:8065",
		"userinfo before IP":      "http://user:pass@127.0.0.1:8065",
		"private network":         "http://10.0.0.1:8065",
		"no scheme, loopback":     "127.0.0.1/jira",
		"IPv6 unique local":       "http://[fd00::1]:8065",
		"trailing dot localhost.": "http://localhost.:8065",
	} {
		t.Run(name, func(t *testing.T) {
			p, _ := setupInstallURLCheckPlugin(t, "")

			_, err := p.checkJiraURL(rawURL)

			var checkErr *utils.JiraURLCheckError
			require.True(t, errors.As(err, &checkErr), "expected a JiraURLCheckError, got %v", err)
			assert.Contains(t, checkErr.Cause.Error(), "address forbidden")
		})
	}
}

func TestCheckJiraURLWithoutServerSettingsBlocksInternal(t *testing.T) {
	var hits int32
	srv := newStatusServer(t, http.StatusOK, `{"state":"RUNNING"}`, &hits)

	api := &plugintest.API{}
	p := &Plugin{}
	p.SetAPI(api)
	p.client = pluginapi.NewClient(api, p.Driver)
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()

	_, err := p.checkJiraURL(srv.URL)

	assert.Error(t, err)
	assert.Equal(t, int32(0), atomic.LoadInt32(&hits))
}
