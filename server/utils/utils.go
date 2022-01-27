// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package utils

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/pkg/errors"
)

const NotAvailable = "n/a"

type ReactSelectOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

func NormalizeJiraURL(jiraURL string) (string, error) {
	u, err := url.Parse(jiraURL)
	if err != nil {
		return "", err
	}
	if u.Host == "" {
		ss := strings.Split(u.Path, "/")
		if len(ss) > 0 && ss[0] != "" {
			u.Host = ss[0]
			u.Path = path.Join(ss[1:]...)
		}
		u, err = url.Parse(u.String())
		if err != nil {
			return "", err
		}
	}
	if u.Host == "" {
		return "", errors.Errorf("Invalid URL, no hostname: %q", jiraURL)
	}
	if u.Scheme == "" {
		u.Scheme = "https"
	}

	jiraURL = strings.TrimSuffix(u.String(), "/")
	return jiraURL, nil
}

// Reference: https://gobyexample.com/collection-functions
func Map(vs []string, f func(string) string) []string {
	vsm := make([]string, len(vs))
	for i, v := range vs {
		vsm[i] = f(v)
	}
	return vsm
}

func IsJiraCloudURL(jiraURL string) bool {
	u, err := url.Parse(jiraURL)
	if err != nil {
		return false
	}
	return strings.HasSuffix(u.Hostname(), ".atlassian.net")
}

type JiraStatus struct {
	State string `json:"state"`
}

// CheckJiraURL checks if `/status` endpoint of the Jira URL is accessible
// and responding with the correct state which is "RUNNING"
func CheckJiraURL(mattermostSiteURL, jiraURL string, requireHTTPS bool) (string, error) {
	jiraURL, err := NormalizeJiraURL(jiraURL)
	if err != nil {
		return "", err
	}
	if jiraURL == strings.TrimSuffix(mattermostSiteURL, "/") {
		return "", errors.Errorf("%s is the Mattermost site URL. Please use your Jira URL", jiraURL)
	}
	if !strings.HasPrefix(jiraURL, "https://") {
		return "", errors.New("a secure https URL is required")
	}

	resp, err := http.Get(jiraURL + "/status")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("Jira server returned http status code %q when checking for availability: %q", resp.Status, jiraURL)
	}

	resBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var status JiraStatus
	err = json.Unmarshal(resBody, &status)
	if err != nil {
		return "", err
	}
	if status.State != "RUNNING" {
		return "", errors.Errorf("Jira server is not in correct state, it should be up and running: %q", jiraURL)
	}
	return jiraURL, nil
}
