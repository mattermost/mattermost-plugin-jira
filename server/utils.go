package main

import (
	"fmt"
	"net/url"
	"regexp"

	"github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/model"
)

func (p *Plugin) CreateBotDMPost(userID, message, postType string) *model.AppError {
	conf := p.getConfig()
	channel, aerr := p.API.GetDirectChannel(userID, conf.botUserID)
	if aerr != nil {
		p.errorf("Couldn't get bot's DM channel to userId:%v, error:%v", userID, aerr.Error())
		return aerr
	}

	post := &model.Post{
		UserId:    conf.botUserID,
		ChannelId: channel.Id,
		Message:   message,
		Type:      postType,
		Props: map[string]interface{}{
			"from_webhook":      "true",
			"override_username": JIRA_USERNAME,
			"override_icon_url": JIRA_ICON_URL,
		},
	}

	_, aerr = p.API.CreatePost(post)
	if aerr != nil {
		p.errorf("Couldn't create post, error:%v", aerr.Error())
		return aerr
	}

	return nil
}

func (p *Plugin) loadJIRAProjectKeys(jiraClient *jira.Client) ([]string, error) {
	list, _, err := jiraClient.Project.GetList()
	if err != nil {
		return nil, errors.WithMessage(err, "Error requesting list of JIRA projects")
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
