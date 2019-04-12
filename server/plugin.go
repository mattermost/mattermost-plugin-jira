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

	// TODO: support mutiple instances, how? Seems like the config UI needs to be rethought
	// for things like multiple instances.
	// JiraServerURL needs to be configured to run in the JIRA Server mode
	// JiraServerURL string
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

	// configuration and a muttex to control concurrent access
	conf     config
	confLock sync.RWMutex

	atlassianConnectTemplate *template.Template
	userConfigTemplate       *template.Template
}

func (p *Plugin) getConfig() config {
	p.confLock.RLock()
	defer p.confLock.RUnlock()
	return p.conf
}

func (p *Plugin) updateConfig(f func(conf *config)) config {
	p.confLock.Lock()
	defer p.confLock.Unlock()

	f(&p.conf)
	return p.conf
}

// OnConfigurationChange is invoked when configuration changes may have been made.
func (p *Plugin) OnConfigurationChange() error {

	// Load the public configuration fields from the Mattermost server configuration.
	ec := externalConfig{}
	err := p.API.LoadPluginConfiguration(&ec)
	if err != nil {
		return errors.Wrap(err, "failed to load plugin configuration")
	}

	p.updateConfig(func(conf *config) {
		conf.externalConfig = ec
	})

	return nil
}
func (p *Plugin) OnActivate() error {
	conf := p.getConfig()
	user, aerr := p.API.GetUserByUsername(conf.UserName)
	if aerr != nil {
		return fmt.Errorf("Unable to find user with configured username: %v, error: %v", conf.UserName, aerr)
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

	conf = p.updateConfig(func(conf *config) {
		conf.botUserID = user.Id

	})

	p.API.RegisterCommand(getCommand())

	return nil
}

func (p *Plugin) MessageHasBeenPosted(c *plugin.Context, post *model.Post) {
	var err error
	defer func() {
		if err != nil {
			p.errorf("MessageHasBeenPosted: %s", err.Error())
		}
	}()

	ji, err := p.LoadCurrentJIRAInstance()
	if err != nil {
		err = errors.WithMessage(err, "failed to load current JIRA instance")
		return
	}

	projectKeys, err := p.loadJIRAProjectKeys(ji, false)
	if err != nil {
		err = errors.WithMessage(err, "failed to load project keys from JIRA")
		return
	}

	issues := parseJIRAIssuesFromText(post.Message, projectKeys)
	if len(issues) == 0 {
		return
	}

	channel, aerr := p.API.GetChannel(post.ChannelId)
	if aerr != nil {
		err = errors.WithMessagef(aerr, "failed to load channel ID: %s", post.ChannelId)
		return
	}

	if channel.Type != model.CHANNEL_OPEN {
		err = errors.New("ignoring JIRA comment in " + channel.Name)
		return
	}

	team, aerr := p.API.GetTeam(channel.TeamId)
	if aerr != nil {
		err = errors.WithMessagef(aerr, "failed to load team ID: %v", channel.TeamId)
		return
	}

	user, aerr := p.API.GetUser(post.UserId)
	if aerr != nil {
		err = errors.WithMessagef(aerr, "failed to load user ID: %v", post.UserId)
		return
	}

	conf := p.API.GetConfig()
	permalink := fmt.Sprintf("%v/%v/pl/%v", *conf.ServiceSettings.SiteURL, team.Name, post.Id)

	var jiraClient *jira.Client
	userinfo, err := p.LoadJIRAUserInfo(ji, post.UserId)
	if err == nil {
		jiraClient, _, err = ji.GetJIRAClientForUser(userinfo)
	} else {
		if !team.AllowOpenInvite {
			p.errorf("User %v is not connected and team %v does not allow open invites",
				user.GetDisplayName(model.SHOW_NICKNAME_FULLNAME), team.DisplayName)
			return
		}

		// TODO reconsider enabling posting comments anonymously if the author
		// has not connected his account
		// jiraClient, err = p.GetJIRAClientForServer()
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
	conf := p.getConfig()
	if conf.ExternalURL != "" {
		return conf.ExternalURL
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
