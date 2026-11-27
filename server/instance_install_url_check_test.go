// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-jira/server/enterprise"
	"github.com/mattermost/mattermost-plugin-jira/server/utils"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

type recordingInstanceStore struct {
	mu             sync.Mutex
	storedInstance []Instance
	storedLists    int
}

var _ InstanceStore = (*recordingInstanceStore)(nil)

func (s *recordingInstanceStore) DeleteInstance(types.ID) error { return nil }
func (s *recordingInstanceStore) LoadInstance(types.ID) (Instance, error) {
	return nil, nil
}
func (s *recordingInstanceStore) LoadInstanceFullKey(string) (Instance, error) {
	return nil, nil
}
func (s *recordingInstanceStore) LoadInstances() (*Instances, error) {
	return NewInstances(), nil
}
func (s *recordingInstanceStore) StoreInstance(instance Instance) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.storedInstance = append(s.storedInstance, instance)
	return nil
}
func (s *recordingInstanceStore) StoreInstances(*Instances) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.storedLists++
	return nil
}

func (s *recordingInstanceStore) writes() (int, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.storedInstance), s.storedLists
}

func setupInstallURLCheckPlugin(t *testing.T, allowedInternal string) (*Plugin, *recordingInstanceStore) {
	t.Helper()

	api := &plugintest.API{}
	p := &Plugin{}
	p.SetAPI(api)
	p.client = pluginapi.NewClient(api, p.Driver)
	p.enterpriseChecker = enterprise.NewEnterpriseChecker(api)
	p.updateConfig(func(conf *config) {
		conf.mattermostSiteURL = "https://mattermost.example.com"
	})
	store := &recordingInstanceStore{}
	p.instanceStore = store

	mmConfig := &model.Config{}
	mmConfig.SetDefaults()
	mmConfig.ServiceSettings.AllowedUntrustedInternalConnections = model.NewPointer(allowedInternal)

	bundlePath, err := filepath.Abs("..")
	require.NoError(t, err)

	api.On("GetConfig").Return(mmConfig).Maybe()
	api.On("GetLicense").Return(nil).Maybe()
	api.On("GetBundlePath").Return(bundlePath, nil).Maybe()
	api.On("UnregisterCommand", mock.Anything, mock.Anything).Return(nil).Maybe()
	api.On("RegisterCommand", mock.Anything).Return(nil).Maybe()
	api.On("PublishWebSocketEvent", mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("LogError", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()

	return p, store
}

// newStatusServer returns a loopback server whose /status endpoint replies
// with the given status code and body, counting requests to /status.
func newStatusServer(t *testing.T, statusCode int, body string, hits *int32) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/status" {
			atomic.AddInt32(hits, 1)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

type installFunc func(p *Plugin, rawURL string) error

var installEntryPoints = map[string]installFunc{
	"server": func(p *Plugin, rawURL string) error {
		_, _, err := p.installServerInstance(rawURL)
		return err
	},
	"cloud-oauth": func(p *Plugin, rawURL string) error {
		_, _, err := p.installCloudOAuthInstance(rawURL)
		return err
	},
}

func TestInstallInstanceBlocksInternalJiraURL(t *testing.T) {
	for name, install := range installEntryPoints {
		t.Run(name, func(t *testing.T) {
			var hits int32
			srv := newStatusServer(t, http.StatusOK, `{"state":"RUNNING"}`, &hits)
			p, store := setupInstallURLCheckPlugin(t, "")

			err := install(p, srv.URL)

			assert.Error(t, err)
			assert.Equal(t, int32(0), atomic.LoadInt32(&hits), "request to an internal address must be blocked before reaching the target")
			instancesStored, listsStored := store.writes()
			assert.Equal(t, 0, instancesStored, "no instance may be stored for an internal address")
			assert.Equal(t, 0, listsStored, "the instance list must not be updated for an internal address")
		})
	}
}

func TestInstallInstanceErrorDoesNotLeakTargetDetails(t *testing.T) {
	closedSrv := httptest.NewServer(http.NotFoundHandler())
	closedURL := closedSrv.URL
	closedSrv.Close()

	for name, tc := range map[string]struct {
		targetURL       func(t *testing.T) string
		allowedInternal string
		forbidden       []string
	}{
		"upstream 404 status": {
			targetURL: func(t *testing.T) string {
				var hits int32
				return newStatusServer(t, http.StatusNotFound, "not here", &hits).URL
			},
			forbidden: []string{"404", "Not Found"},
		},
		"upstream 404 status, internal connections allowed": {
			targetURL: func(t *testing.T) string {
				var hits int32
				return newStatusServer(t, http.StatusNotFound, "not here", &hits).URL
			},
			allowedInternal: "127.0.0.0/8",
			forbidden:       []string{"404", "Not Found"},
		},
		"closed port": {
			targetURL:       func(*testing.T) string { return closedURL },
			allowedInternal: "127.0.0.0/8",
			forbidden:       []string{"connection refused", "refused", "dial"},
		},
		"closed port, internal connections not allowed": {
			targetURL: func(*testing.T) string { return closedURL },
			forbidden: []string{"connection refused", "refused", "dial"},
		},
	} {
		for entryName, install := range installEntryPoints {
			t.Run(name+"/"+entryName, func(t *testing.T) {
				p, store := setupInstallURLCheckPlugin(t, tc.allowedInternal)

				err := install(p, tc.targetURL(t))

				require.Error(t, err)
				for _, s := range tc.forbidden {
					assert.NotContains(t, strings.ToLower(err.Error()), strings.ToLower(s))
				}
				instancesStored, _ := store.writes()
				assert.Equal(t, 0, instancesStored)
			})
		}
	}
}

func TestInstallServerInstanceDoesNotFollowRedirectToInternalAddress(t *testing.T) {
	var targetHits int32
	target := newStatusServer(t, http.StatusOK, `{"state":"RUNNING"}`, &targetHits)

	var firstHopHits int32
	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&firstHopHits, 1)
		http.Redirect(w, r, target.URL+"/status", http.StatusFound)
	}))
	t.Cleanup(redirector.Close)

	// Only the "localhost" hostname of the first hop is trusted; the redirect
	// target is a raw loopback IP that is not in the allow list.
	redirectorURL := strings.Replace(redirector.URL, "127.0.0.1", "localhost", 1)
	p, store := setupInstallURLCheckPlugin(t, "localhost")

	_, _, err := p.installServerInstance(redirectorURL)

	assert.Error(t, err)
	assert.Positive(t, atomic.LoadInt32(&firstHopHits), "the first hop must be reached for this test to be meaningful")
	assert.Equal(t, int32(0), atomic.LoadInt32(&targetHits), "redirect to an untrusted internal address must not be followed")
	instancesStored, listsStored := store.writes()
	assert.Equal(t, 0, instancesStored)
	assert.Equal(t, 0, listsStored)
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

func TestInstallServerInstanceAllowedInternalJiraURL(t *testing.T) {
	for name, allowed := range map[string]string{
		"CIDR":                   "127.0.0.0/8",
		"single IP CIDR":         "127.0.0.1/32",
		"CIDR among other hosts": "jira.internal, 127.0.0.0/8 10.0.0.0/8",
	} {
		t.Run(name, func(t *testing.T) {
			var hits int32
			srv := newStatusServer(t, http.StatusOK, `{"state":"RUNNING"}`, &hits)
			p, store := setupInstallURLCheckPlugin(t, allowed)

			jiraURL, instance, err := p.installServerInstance(srv.URL)

			require.NoError(t, err)
			require.NotNil(t, instance)
			assert.Equal(t, srv.URL, jiraURL)
			assert.Equal(t, int32(1), atomic.LoadInt32(&hits))
			instancesStored, listsStored := store.writes()
			assert.Equal(t, 1, instancesStored)
			assert.Equal(t, 1, listsStored)
		})
	}
}

type singleInstanceStore struct {
	recordingInstanceStore
	instance Instance
}

func (s *singleInstanceStore) LoadInstance(types.ID) (Instance, error) {
	return s.instance, nil
}

// newLocalhostRedirector stands in for an installed host that passed the
// install check: "localhost" is allowed by name, and every request is
// redirected to target, keeping the method and path.
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

func TestServerInstanceRequestsBlockInternalAddresses(t *testing.T) {
	entryPoints := map[string]func(si *serverInstance) error{
		"OAuth1 request token": func(si *serverInstance) error {
			_, _, err := si.GetUserConnectURL("user1")
			return err
		},
		"OAuth1 access token": func(si *serverInstance) error {
			_, _, err := si.getOAuth1Config().AccessToken("token", "secret", "verifier")
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

	for _, via := range []string{"redirect from allowed host", "instance host resolves to an internal address"} {
		for name, call := range entryPoints {
			t.Run(via+"/"+name, func(t *testing.T) {
				var hits int32
				target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					atomic.AddInt32(&hits, 1)
					w.WriteHeader(http.StatusOK)
				}))
				t.Cleanup(target.Close)

				p, _ := setupInstallURLCheckPlugin(t, "localhost")
				markAdminNoticeSent(p)
				instanceURL := target.URL
				if via == "redirect from allowed host" {
					instanceURL = newLocalhostRedirector(t, target.URL)
				}
				si := newServerInstanceForURLTest(t, p, instanceURL)

				err := call(si)

				assert.Error(t, err)
				assert.Equal(t, int32(0), atomic.LoadInt32(&hits), "request to an internal address must be blocked")
			})
		}
	}
}

func TestServerInstanceAllowedInternalAddress(t *testing.T) {
	var hits int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"jira-user"}`))
	}))
	t.Cleanup(target.Close)

	p, _ := setupInstallURLCheckPlugin(t, "127.0.0.1/32")
	si := newServerInstanceForURLTest(t, p, target.URL)

	client, err := si.GetClient(&Connection{Oauth1AccessToken: "token", Oauth1AccessSecret: "secret"})
	require.NoError(t, err)
	_, err = client.GetSelf()

	require.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&hits))
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
		target          string
		allowedInternal string
		forbidden       []string
	}{
		"redirect to closed internal port": {
			target:          closedURL,
			allowedInternal: "localhost",
			forbidden:       []string{"refused", "dial", "forbidden", "127.0.0.1", closedPort},
		},
		"redirect to allowed internal port answering 404": {
			target:          notFoundSrv.URL,
			allowedInternal: "localhost 127.0.0.1/32",
			forbidden:       []string{"404", "127.0.0.1", notFoundPort},
		},
	} {
		t.Run(name, func(t *testing.T) {
			p, _ := setupInstallURLCheckPlugin(t, tc.allowedInternal)
			markAdminNoticeSent(p)
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
			assert.Contains(t, body, strings.ToLower(errJiraServerRequestFailed.Error()))
			for _, s := range tc.forbidden {
				assert.NotContains(t, body, strings.ToLower(s))
			}
		})
	}
}

// markAdminNoticeSent makes one-time admin notices look already sent.
func markAdminNoticeSent(p *Plugin) {
	p.API.(*plugintest.API).On("KVSetWithOptions", mock.Anything, mock.Anything, mock.Anything).Return(false, (*model.AppError)(nil)).Maybe()
}

func TestBlockedInstanceNotifiesAdminsWithoutTargetDetails(t *testing.T) {
	target := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(target.Close)

	p, _ := setupInstallURLCheckPlugin(t, "localhost")
	p.updateConfig(func(conf *config) {
		conf.botUserID = connectTestBotUserID
	})
	instanceURL := newLocalhostRedirector(t, target.URL)
	si := newServerInstanceForURLTest(t, p, instanceURL)

	api := p.API.(*plugintest.API)
	api.On("KVSetWithOptions", mock.Anything, mock.Anything, mock.Anything).Return(true, (*model.AppError)(nil)).Once()
	// The exact match also checks that the redirect target's address isn't named.
	expectAdminDM(api, fmt.Sprintf(blockedInstanceNotice, instanceURL, "localhost"))

	_, _, err := si.GetUserConnectURL("user1")

	require.Error(t, err)
	api.AssertExpectations(t)
}

func TestInstanceRequestErrorsDoNotNameResolvedAddress(t *testing.T) {
	target := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(target.Close)
	// "localhost." is not in the allow list, so its resolved address is checked and blocked.
	targetByName := strings.Replace(target.URL, "127.0.0.1", "localhost.", 1)

	p, _ := setupInstallURLCheckPlugin(t, "localhost")
	markAdminNoticeSent(p)
	si := newServerInstanceForURLTest(t, p, newLocalhostRedirector(t, targetByName))

	client, err := si.GetClient(&Connection{Oauth1AccessToken: "token", Oauth1AccessSecret: "secret"})
	require.NoError(t, err)
	_, err = client.GetSelf()

	require.Error(t, err)
	for _, s := range []string{"127.0.0.1", "reserved", "forbidden"} {
		assert.NotContains(t, strings.ToLower(err.Error()), s)
	}
}
