package main

import (
	"fmt"
	"regexp"

	"github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

func CreateBotDMPost(conf Config, api plugin.API, userStore UserStore, instance Instance, userId, message,
	postType string) (post *model.Post, returnErr error) {
	defer func() {
		if returnErr != nil {
			returnErr = errors.WithMessage(returnErr,
				fmt.Sprintf("failed to create direct post to user %v: ", userId))
		}
	}()

	// Don't send DMs to users who have turned off notifications
	jiraUser, err := userStore.LoadJiraUser(instance, userId)
	if err != nil {
		// not connected to Jira, so no need to send a DM, and no need to report an error
		return nil, nil
	}
	if jiraUser.Settings == nil || !jiraUser.Settings.Notifications {
		return nil, nil
	}

	channel, appErr := api.GetDirectChannel(userId, conf.BotUserID)
	if appErr != nil {
		return nil, appErr
	}

	post = &model.Post{
		UserId:    conf.BotUserID,
		ChannelId: channel.Id,
		Message:   message,
		Type:      postType,
		Props: map[string]interface{}{
			"from_webhook":      "true",
			"override_username": PluginMattermostUsername,
			"override_icon_url": PluginIconURL,
		},
	}

	_, appErr = api.CreatePost(post)
	if appErr != nil {
		return nil, appErr
	}

	return post, nil
}

func StoreCurrentInstanceAndNotify(api plugin.API, currentInstanceStore CurrentInstanceStore,
	instance Instance) error {

	appErr := currentInstanceStore.StoreCurrentInstance(instance)
	if appErr != nil {
		return appErr
	}
	// Notify users we have installed an instance
	api.PublishWebSocketEvent(
		wSEventInstanceStatus,
		map[string]interface{}{
			"instance_installed": true,
		},
		&model.WebsocketBroadcast{},
	)
	return nil
}

func (p *Plugin) loadProjectKeys(jiraClient *jira.Client) ([]string, error) {
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

func parseJiraIssuesFromText(text string, keys []string) []string {
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
