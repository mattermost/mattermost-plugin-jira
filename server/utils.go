package main

import (
	"fmt"
	"net/url"
	"regexp"

	"github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/model"
)

func (p *Plugin) CreateBotDMPost(ji Instance, userId, message, postType string) (returnErr error) {
	defer func() {
		if returnErr != nil {
			returnErr = errors.WithMessage(returnErr,
				fmt.Sprintf("failed to create direct post to user %v: ", userId))
		}
	}()

	// Don't send DMs to users who have turned off notifications
	jiraUser, err := p.LoadJIRAUser(ji, userId)
	if err != nil {
		// not connected to Jira, so no need to send a DM, and no need to report an error
		return
	}
	if !jiraUser.Settings.Notifications {
		return
	}

	conf := p.getConfig()
	channel, appErr := p.API.GetDirectChannel(userId, conf.botUserID)
	if appErr != nil {
		return appErr
	}

	post := &model.Post{
		UserId:    conf.botUserID,
		ChannelId: channel.Id,
		Message:   message,
		Type:      postType,
		Props: map[string]interface{}{
			"from_webhook":      "true",
			"override_username": PluginMattermostUsername,
			"override_icon_url": PluginIconURL,
		},
	}

	_, appErr = p.API.CreatePost(post)
	if appErr != nil {
		return appErr
	}

	return nil
}

func (p *Plugin) loadJIRAProjectKeys(jiraClient *jira.Client) ([]string, error) {
	list, _, err := jiraClient.Project.GetList()
	if err != nil {
		return nil, errors.WithMessage(err, "Error requesting list of Jira projects")
	}

	projectKeys := []string{}
	for _, proj := range *list {
		projectKeys = append(projectKeys, proj.Key)
	}
	return projectKeys, nil
}

func parseJIRAUsernamesFromText(text string) []string {
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

func parseJIRAIssuesFromText(text string, keys []string) []string {
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

func getUserURL(issue *JIRAWebhookIssue, user *jira.User) string {
	return user.Self
}
