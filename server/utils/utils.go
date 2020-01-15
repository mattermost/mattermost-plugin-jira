// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package utils

import (
	"net/url"
	"path"
	"strings"

	"github.com/pkg/errors"
)

type ReactSelectOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

func NormalizeInstallURL(mattermostSiteURL, jiraURL string) (string, error) {
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
	if jiraURL == strings.TrimSuffix(mattermostSiteURL, "/") {
		return "", errors.Errorf("%s is the Mattermost site URL. Please use your Jira URL with `/jira install`.", jiraURL)
	}

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

func IsJiraCloudURL(jiraURL string) (bool, error) {
	u, err := url.Parse(jiraURL)
	if err != nil {
		return false, err
	}
	return strings.HasSuffix(u.Hostname(), ".atlassian.net"), nil
}
