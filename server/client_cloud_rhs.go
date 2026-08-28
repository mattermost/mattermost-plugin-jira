// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"crypto/rand"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	jira "github.com/andygrunwald/go-jira"
)

const (
	rateLimitReasonGlobalQuota = "jira-quota-global-based"
	rhsSearchMaxAttempts       = 4
)

type rhsRetry struct {
	base        time.Duration
	cap         time.Duration
	maxAttempts int
	sleep       func(time.Duration)
	jitter      func() float64 // factor in [0.7, 1.3]
}

func defaultRHSRetry() rhsRetry {
	return rhsRetry{
		base:        2 * time.Second,
		cap:         30 * time.Second,
		maxAttempts: rhsSearchMaxAttempts,
		sleep:       time.Sleep,
		jitter:      rhsRandomJitter,
	}
}

func rhsRandomJitter() float64 {
	var b [1]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 1.0
	}
	return 0.7 + float64(b[0])*(1.3-0.7)/255.0
}

func closeJiraResp(resp *jira.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_ = resp.Body.Close()
}

// rhsBackoffDelay sleeps after a retryable 429. Prefer parseable Retry-After
// (seconds, >= 0) over exponential backoff; the cap applies to both exponential
// and Retry-After. Then apply jitter.
func rhsBackoffDelay(retry rhsRetry, failedAttempt int, retryAfterHeader string) time.Duration {
	shift := failedAttempt - 1
	if shift < 0 {
		shift = 0
	}
	delay := retry.base * time.Duration(1<<shift)
	if secs, err := strconv.Atoi(strings.TrimSpace(retryAfterHeader)); err == nil && secs >= 0 {
		delay = time.Duration(secs) * time.Second
	}
	if retry.cap > 0 && delay > retry.cap {
		delay = retry.cap
	}
	factor := 1.0
	if retry.jitter != nil {
		factor = retry.jitter()
	}
	return time.Duration(float64(delay) * factor)
}

func (client jiraCloudClient) ListStatuses() ([]*JiraStatus, error) {
	var result []*JiraStatus
	if err := client.RESTGet("3/status", nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (client jiraCloudClient) ListStatusCategories() ([]*JiraStatusCategory, error) {
	var result []*JiraStatusCategory
	if err := client.RESTGet("3/statuscategory", nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (client jiraCloudClient) SearchJQL(params CloudSearchParams) (*CloudSearchResult, error) {
	return client.searchJQL(params, defaultRHSRetry())
}

func (client jiraCloudClient) searchJQL(params CloudSearchParams, retry rhsRetry) (*CloudSearchResult, error) {
	if retry.maxAttempts < 1 {
		retry.maxAttempts = rhsSearchMaxAttempts
	}
	if retry.sleep == nil {
		retry.sleep = time.Sleep
	}

	endpoint, err := endpointURL("3/search/jql")
	if err != nil {
		return nil, err
	}

	for attempt := 1; attempt <= retry.maxAttempts; attempt++ {
		req, err := client.Jira.NewRequest(http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}
		q := req.URL.Query()
		q.Set("jql", params.JQL)
		if len(params.Fields) > 0 {
			q.Set("fields", strings.Join(params.Fields, ","))
		}
		if params.MaxResults > 0 {
			q.Set("maxResults", strconv.Itoa(params.MaxResults))
		}
		if params.NextPageToken != "" {
			q.Set("nextPageToken", params.NextPageToken)
		}
		req.URL.RawQuery = q.Encode()

		resp, doErr := client.Jira.Do(req, nil)

		if resp != nil && resp.StatusCode == http.StatusTooManyRequests {
			reason := resp.Header.Get("RateLimit-Reason")
			retryAfter := resp.Header.Get("Retry-After")
			closeJiraResp(resp)

			if reason == rateLimitReasonGlobalQuota {
				return nil, ErrRateLimited
			}
			if attempt == retry.maxAttempts {
				return nil, ErrRateLimited
			}
			retry.sleep(rhsBackoffDelay(retry, attempt, retryAfter))
			continue
		}

		if doErr != nil {
			if resp != nil {
				wrapped := userFriendlyJiraError(resp, doErr)
				closeJiraResp(resp)
				return nil, wrapped
			}
			return nil, doErr
		}

		var result CloudSearchResult
		decErr := json.NewDecoder(resp.Body).Decode(&result)
		closeJiraResp(resp)
		if decErr != nil {
			return nil, decErr
		}
		return &result, nil
	}

	return nil, ErrRateLimited
}
