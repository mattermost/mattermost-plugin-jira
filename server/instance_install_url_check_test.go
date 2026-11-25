// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
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
	assert.Equal(t, int32(0), atomic.LoadInt32(&targetHits), "redirect to an untrusted internal address must not be followed")
	instancesStored, listsStored := store.writes()
	assert.Equal(t, 0, instancesStored)
	assert.Equal(t, 0, listsStored)
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
