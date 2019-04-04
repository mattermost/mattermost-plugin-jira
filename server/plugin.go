// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"crypto/rsa"
	"fmt"
	"path/filepath"
	"sync"
	"text/template"

	jira "github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

const (
	JIRA_USERNAME = "Jira Plugin"
	JIRA_ICON_URL = "https://s3.amazonaws.com/mattermost-plugin-media/jira.jpg"
)

type Plugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration

	oauth2Config oauth2.Config

	botUserID   string
	rsaKey      *rsa.PrivateKey
	projectKeys []string

	atlassianConnectTemplate *template.Template
	userConfigTemplate       *template.Template
}

func (p *Plugin) OnActivate() error {
	config := p.getConfiguration()
	user, apperr := p.API.GetUserByUsername(config.UserName)
	if apperr != nil {
		return fmt.Errorf("Unable to find user with configured username: %v, error: %v", config.UserName, apperr)
	}

	bpath, err := p.API.GetBundlePath()
	if err != nil {
		return err
	}

	fpath := filepath.Join(bpath, "server", "dist", "templates", "atlassian-connect.json")
	p.atlassianConnectTemplate, err = template.ParseFiles(fpath)
	if err != nil {
		return err
	}

	fpath = filepath.Join(bpath, "server", "dist", "templates", "user-config.html")
	p.userConfigTemplate, err = template.ParseFiles(fpath)
	if err != nil {
		return err
	}

	p.botUserID = user.Id
	p.rsaKey = p.getRSAKey()

	p.oauth2Config = oauth2.Config{
		ClientID:     "LimAAPOhX7ncIN7cPB77tZ1Gwz0r2WmL",
		ClientSecret: "01_Y6g1JRmLnSGcaRU19LzhfnsXHAGwtuQTacQscxR3eCy7tzhLYYbuQHXiVIJq_",
		Scopes:       []string{"read:jira-work", "read:jira-user", "write:jira-work"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://auth.atlassian.com/authorize",
			TokenURL: "https://auth.atlassian.com/oauth/token",
		},
		RedirectURL: fmt.Sprintf("%v/plugins/%v/oauth/complete", p.externalURL(), manifest.Id),
	}

	p.API.RegisterCommand(getCommand())

	return nil
}

func (p *Plugin) MessageHasBeenPosted(c *plugin.Context, post *model.Post) {
	projectKeys, err := p.loadJIRAProjectKeys(false)
	if err != nil {
		return
	}

	issues := parseJIRAIssuesFromText(post.Message, projectKeys)
	if len(issues) == 0 {
		return
	}

	channel, _ := p.API.GetChannel(post.ChannelId)
	if channel == nil {
		return
	}

	if channel.Type != model.CHANNEL_OPEN {
		return
	}

	team, _ := p.API.GetTeam(channel.TeamId)
	if team == nil {
		return
	}

	user, _ := p.API.GetUser(post.UserId)
	if user == nil {
		return
	}

	config := p.API.GetConfig()
	permalink := fmt.Sprintf("%v/%v/pl/%v", *config.ServiceSettings.SiteURL, team.Name, post.Id)

	var jiraClient *jira.Client
	userinfo, err := p.LoadJIRAUserInfo(post.UserId)
	if err == nil {
		jiraClient, _, err = p.getJIRAClientForUser(userinfo.AccountId)
	} else {
		if !team.AllowOpenInvite {
			return
		}

		jiraClient, err = p.getJIRAClientForServer()
	}
	if err != nil {
		return
	}

	for _, issue := range issues {
		comment := &jira.Comment{
			Body: fmt.Sprintf("%s mentioned this ticket in Mattermost:\n{quote}\n%s\n{quote}\n\n[View message in Mattermost|%s]",
				user.Username, post.Message, permalink),
		}

		_, _, err := jiraClient.Issue.AddComment(issue, comment)
		if err != nil {
			p.errorf("%v", errors.WithMessage(err, "failed to add comment"))
		}
	}
}

func (p *Plugin) externalURL() string {
	config := p.getConfiguration()
	if config.ExternalURL != "" {
		return config.ExternalURL
	}
	return *p.API.GetConfig().ServiceSettings.SiteURL
}

func (p *Plugin) debugf(f string, args ...interface{}) {
	p.API.LogDebug(fmt.Sprintf(f, args...))
}

func (p *Plugin) infof(f string, args ...interface{}) {
	p.API.LogInfo(fmt.Sprintf(f, args...))
}

func (p *Plugin) errorf(f string, args ...interface{}) {
	p.API.LogError(fmt.Sprintf(f, args...))
}
