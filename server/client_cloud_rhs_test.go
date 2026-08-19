// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	jira "github.com/andygrunwald/go-jira"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestCloudRHSClient(t *testing.T, handler http.HandlerFunc) *jiraCloudClient {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)
	jc, err := jira.NewClient(ts.Client(), ts.URL)
	require.NoError(t, err)
	client, ok := newCloudClient(jc).(*jiraCloudClient)
	require.True(t, ok)
	return client
}

func loadRHSTestdata(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return b
}

func testRetryNoSleep(sleeps *[]time.Duration) rhsRetry {
	return rhsRetry{
		base:        2 * time.Second,
		cap:         30 * time.Second,
		maxAttempts: rhsSearchMaxAttempts,
		sleep: func(d time.Duration) {
			*sleeps = append(*sleeps, d)
		},
		jitter: func() float64 { return 1.0 },
	}
}

func writeJSON(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func TestCloudRHSSearchJQLHappyPath(t *testing.T) {
	var requests int
	client := newTestCloudRHSClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/rest/api/3/search/jql", r.URL.Path)

		q := r.URL.Query()
		assert.Equal(t, "assignee = currentUser() AND statusCategory != done ORDER BY updated DESC, key ASC", q.Get("jql"))
		assert.Equal(t, "20", q.Get("maxResults"))
		fields := q.Get("fields")
		assert.Contains(t, fields, "summary")
		assert.Contains(t, fields, "status")
		assert.Equal(t, "", q.Get("nextPageToken"))
		_, hasNext := q["nextPageToken"]
		assert.False(t, hasNext)

		writeJSON(w, http.StatusOK, loadRHSTestdata(t, "rhs-search-jql-page1.json"))
	})

	result, err := client.SearchJQL(CloudSearchParams{
		JQL:        "assignee = currentUser() AND statusCategory != done ORDER BY updated DESC, key ASC",
		Fields:     []string{"summary", "status"},
		MaxResults: 20,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Issues, 1)
	assert.Equal(t, "TES-41", result.Issues[0].Key)
	assert.Equal(t, "opaque-token-page-2", result.NextPageToken)
	assert.False(t, result.IsLast)
	assert.Equal(t, 1, requests)
}

func TestCloudRHSSearchJQLCursorPassthrough(t *testing.T) {
	var requests int
	client := newTestCloudRHSClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/rest/api/3/search/jql", r.URL.Path)
		assert.Equal(t, "opaque-token-page-2", r.URL.Query().Get("nextPageToken"))
		writeJSON(w, http.StatusOK, loadRHSTestdata(t, "rhs-search-jql-last.json"))
	})

	result, err := client.SearchJQL(CloudSearchParams{
		JQL:           "assignee = currentUser() ORDER BY updated DESC",
		Fields:        []string{"summary", "status"},
		MaxResults:    20,
		NextPageToken: "opaque-token-page-2",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsLast)
	assert.Equal(t, "", result.NextPageToken)
	assert.Equal(t, 1, requests)
}

func TestCloudRHSSearchJQLRetryAfterHonored(t *testing.T) {
	var requests int
	var sleeps []time.Duration
	client := newTestCloudRHSClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		assert.Equal(t, http.MethodGet, r.Method)
		if requests == 1 {
			w.Header().Set("Retry-After", "7")
			w.Header().Set("RateLimit-Reason", "jira-burst-based")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		writeJSON(w, http.StatusOK, loadRHSTestdata(t, "rhs-search-jql-last.json"))
	})

	result, err := client.searchJQL(CloudSearchParams{
		JQL:        "assignee = currentUser() ORDER BY updated DESC",
		Fields:     []string{"summary"},
		MaxResults: 20,
	}, testRetryNoSleep(&sleeps))
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 2, requests)
	require.Len(t, sleeps, 1)
	assert.Equal(t, 7*time.Second, sleeps[0])
}

func TestCloudRHSSearchJQLGlobalQuotaNoRetry(t *testing.T) {
	var requests int
	var sleeps []time.Duration
	client := newTestCloudRHSClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		assert.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("RateLimit-Reason", "jira-quota-global-based")
		w.Header().Set("Retry-After", "1847")
		w.WriteHeader(http.StatusTooManyRequests)
	})

	result, err := client.searchJQL(CloudSearchParams{
		JQL:        "assignee = currentUser() ORDER BY updated DESC",
		Fields:     []string{"summary"},
		MaxResults: 20,
	}, testRetryNoSleep(&sleeps))
	assert.Nil(t, result)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrRateLimited))
	assert.Equal(t, 1, requests)
	assert.Empty(t, sleeps)
}

func TestCloudRHSSearchJQLAttemptsCapAt4(t *testing.T) {
	var requests int
	var sleeps []time.Duration
	client := newTestCloudRHSClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		assert.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("RateLimit-Reason", "jira-burst-based")
		w.WriteHeader(http.StatusTooManyRequests)
	})

	result, err := client.searchJQL(CloudSearchParams{
		JQL:        "assignee = currentUser() ORDER BY updated DESC",
		Fields:     []string{"summary"},
		MaxResults: 20,
	}, testRetryNoSleep(&sleeps))
	assert.Nil(t, result)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrRateLimited))
	assert.Equal(t, 4, requests)
	require.Len(t, sleeps, 3)
	assert.Equal(t, 2*time.Second, sleeps[0])
	assert.Equal(t, 4*time.Second, sleeps[1])
	assert.Equal(t, 8*time.Second, sleeps[2])
}

func TestCloudRHSListStatuses(t *testing.T) {
	var requests int
	client := newTestCloudRHSClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/rest/api/3/status", r.URL.Path)
		writeJSON(w, http.StatusOK, loadRHSTestdata(t, "rhs-status.json"))
	})

	got, err := client.ListStatuses()
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "3", got[0].ID)
	assert.Equal(t, "In Progress", got[0].Name)
	assert.Equal(t, "indeterminate", got[0].StatusCategory.Key)
	assert.Equal(t, 4, got[0].StatusCategory.ID)
	assert.Equal(t, "10001", got[1].ID)
	assert.Equal(t, "new", got[1].StatusCategory.Key)
	assert.Equal(t, 1, requests)
}

func TestCloudRHSListStatusCategories(t *testing.T) {
	var requests int
	client := newTestCloudRHSClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/rest/api/3/statuscategory", r.URL.Path)
		writeJSON(w, http.StatusOK, loadRHSTestdata(t, "rhs-statuscategory.json"))
	})

	got, err := client.ListStatusCategories()
	require.NoError(t, err)
	require.Len(t, got, 4)
	assert.Equal(t, "undefined", got[0].Key)
	assert.Equal(t, 1, got[0].ID)
	assert.Equal(t, "No Category", got[0].Name)
	assert.Equal(t, "new", got[1].Key)
	assert.Equal(t, 2, got[1].ID)
	assert.Equal(t, "To Do", got[1].Name)
	assert.Equal(t, "indeterminate", got[2].Key)
	assert.Equal(t, 4, got[2].ID)
	assert.Equal(t, "In Progress", got[2].Name)
	assert.Equal(t, "done", got[3].Key)
	assert.Equal(t, 3, got[3].ID)
	assert.Equal(t, "Done", got[3].Name)
	assert.Equal(t, 1, requests)
}

func TestCloudRHSEndpointURL(t *testing.T) {
	status, err := endpointURL("3/status")
	require.NoError(t, err)
	assert.Equal(t, "/rest/api/3/status", status)

	cats, err := endpointURL("3/statuscategory")
	require.NoError(t, err)
	assert.Equal(t, "/rest/api/3/statuscategory", cats)

	search, err := endpointURL("3/search/jql")
	require.NoError(t, err)
	assert.Equal(t, "/rest/api/3/search/jql", search)
}
