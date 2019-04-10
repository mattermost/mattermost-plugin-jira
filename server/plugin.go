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

type externalConfig struct {
	// Bot username
	UserName string

	// Legacy 1.x Webhook secret
	Secret string

	// TODO remove
	ExternalURL string
}

// config captures all cached values that need to be synchronized
type config struct {
	externalConfig

	// Cached actual bot user ID (derived from c.UserName)
	botUserID string

	// Generated once, then cached in the database, and here deserialized
	rsaKey *rsa.PrivateKey

	// secret used to generate auth tokens in the Atlassian connect
	// user mapping flow
	tokenSecret []byte

	// Fetched from JIRA once, then cached
	projectKeys []string
}

type Plugin struct {
	plugin.MattermostPlugin
	JIRAInstance

	// configuration and a muttex to control concurrent access
	config
	configLock sync.RWMutex

	oauth2Config oauth2.Config

	atlassianConnectTemplate *template.Template
	userConfigTemplate       *template.Template
}

func (p *Plugin) getConfig() config {
	p.configLock.RLock()
	defer p.configLock.RUnlock()
	return p.config
}

func (p *Plugin) updateConfig(f func(c *config)) config {
	p.configLock.Lock()
	defer p.configLock.Unlock()

	f(&p.config)
	return p.config
}

// OnConfigurationChange is invoked when configuration changes may have been made.
func (p *Plugin) OnConfigurationChange() error {

	// Load the public configuration fields from the Mattermost server configuration.
	ec := externalConfig{}
	err := p.API.LoadPluginConfiguration(&ec)
	if err != nil {
		return errors.Wrap(err, "failed to load plugin configuration")
	}

	p.updateConfig(func(c *config) {
		c.externalConfig = ec
	})

	return nil
}
func (p *Plugin) OnActivate() error {
	config := p.getConfig()
	user, apperr := p.API.GetUserByUsername(config.UserName)
	if apperr != nil {
		return fmt.Errorf("Unable to find user with configured username: %v, error: %v", config.UserName, apperr)
	}

	tpath := filepath.Join(*(p.API.GetConfig().PluginSettings.Directory), manifest.Id, "server", "dist", "templates")

	var err error
	fpath := filepath.Join(tpath, "atlassian-connect.json")
	p.atlassianConnectTemplate, err = template.ParseFiles(fpath)
	if err != nil {
		return err
	}
	fpath = filepath.Join(tpath, "user-config.html")
	p.userConfigTemplate, err = template.ParseFiles(fpath)
	if err != nil {
		return err
	}

	p.botUserID = user.Id

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
		p.errorf("MessageHasBeenPosted: failed to load project keys from JIRA: %v", err)
		return
	}

	issues := parseJIRAIssuesFromText(post.Message, projectKeys)
	if len(issues) == 0 {
		return
	}

	channel, aerr := p.API.GetChannel(post.ChannelId)
	if aerr != nil {
		p.errorf("MessageHasBeenPosted: failed to load channel ID: %v, error: %v.", post.ChannelId, aerr)
		return
	}

	if channel.Type != model.CHANNEL_OPEN {
		p.infof("MessageHasBeenPosted: ignoring JIRA comment in %v: not a public channel.", channel.Name)
		return
	}

	team, aerr := p.API.GetTeam(channel.TeamId)
	if aerr != nil {
		p.errorf("MessageHasBeenPosted: failed to load team ID: %v, error: %v.", channel.TeamId, aerr)
		return
	}

	user, aerr := p.API.GetUser(post.UserId)
	if aerr != nil {
		p.errorf("MessageHasBeenPosted: failed to load user ID: %v, error: %v.", post.UserId, aerr)
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
			p.errorf("User %v is not connected and team %v does not allow open invites",
				user.GetDisplayName(model.SHOW_NICKNAME_FULLNAME), team.DisplayName)
			return
		}

		jiraClient, err = p.getJIRAClientForServer()
	}
	if err != nil {
		p.errorf("MessageHasBeenPosted: failed to obtain an authenticated client, error: %v.", err)
		return
	}

	for _, issue := range issues {
		comment := &jira.Comment{
			Body: fmt.Sprintf("%s mentioned this ticket in Mattermost:\n{quote}\n%s\n{quote}\n\n[View message in Mattermost|%s]",
				user.Username, post.Message, permalink),
		}

		_, _, err := jiraClient.Issue.AddComment(issue, comment)
		if err != nil {
			p.errorf("MessageHasBeenPosted: failed to add the comment to JIRA, error: %v", err)
		}
	}
}

func (p *Plugin) externalURL() string {
	config := p.getConfig()
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
