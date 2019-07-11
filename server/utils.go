// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"net/url"
	"path"
	"strings"

	"github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"
)

func normalizeInstallURL(jiraURL string) (string, error) {
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
	return strings.TrimSuffix(u.String(), "/"), nil
}

func getIssueURL(issue *jira.Issue) string {
	if issue == nil {
		return ""
	}
	u, _ := url.Parse(issue.Self)
	return u.Scheme + "://" + u.Host + "/browse/" + issue.Key
}

func getUserURL(user *jira.User) string {
	// TODO is this right?
	return user.Self
}

func CloseJiraResponse(resp *jira.Response) {
	if resp != nil && resp.Response != nil {
		resp.Body.Close()
	}
}
