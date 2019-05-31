package main

import (
	"net/url"

	"github.com/andygrunwald/go-jira"
)

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
