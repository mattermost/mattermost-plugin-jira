package main

import (
	"fmt"
	"net/url"
	"regexp"
)

func parseJiraUsernamesFromText(text string) []string {
	usernameMap := map[string]bool{}
	usernames := []string{}

	var re = regexp.MustCompile(`(?m)\[~([a-zA-Z0-9-_.\+]+)\]`)
	for _, match := range re.FindAllString(text, -1) {
		name := match[:len(match)-1]
		name = name[2:]
		if !usernameMap[name] {
			usernames = append(usernames, name)
			usernameMap[name] = true
		}
	}

	return usernames
}

func parseJiraIssueFromText(text string, keys []string) []string {
	issueMap := map[string]bool{}
	issues := []string{}

	for _, key := range keys {
		var re = regexp.MustCompile(fmt.Sprintf(`(?m)%s-[0-9]+`, key))
		for _, match := range re.FindAllString(text, -1) {
			if !issueMap[match] {
				issues = append(issues, match)
				issueMap[match] = true
			}
		}
	}

	return issues
}

func getIssueURL(i *JIRAWebhookIssue) string {
	u, _ := url.Parse(i.Self)
	return u.Scheme + "://" + u.Host + "/browse/" + i.Key
}

func getUserURL(issue *JIRAWebhookIssue, user *JIRAWebhookUser) string {
	u, _ := url.Parse(issue.Self)
	return u.Scheme + "://" + u.Host + "/people/" + user.AccountId
}
