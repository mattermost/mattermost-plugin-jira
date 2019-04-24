// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"crypto/rsa"
	"fmt"
	"path/filepath"
	"sync"
	"text/template"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/plugin"
)

const (
	PluginMattermostUsername = "Jira Plugin"
	PluginIconURL            = "https://s3.amazonaws.com/mattermost-plugin-media/jira.jpg"
)

type externalConfig struct {
	// Bot username
	UserName string

	// Legacy 1.x Webhook secret
	Secret string

	// TODO: support mutiple server instances, how? Seems like the config
	// UI needs to be rethought for things like multiple instances. Or
	// maybe we always check JIRAServerURL on config change (startup) and
	// add/select it.
	// JIRAServerURL needs to be configured to run in the JIRA Server
	// mode
	JIRAServerURL string
}

// config captures all cached values that need to be synchronized
type config struct {
	externalConfig

	// Cached actual bot user ID (derived from c.UserName)
	botUserID string
}

type Plugin struct {
	plugin.MattermostPlugin

	// configuration and a muttex to control concurrent access
	conf     config
	confLock sync.RWMutex

	// Generated once, then cached in the database, and here deserialized
	RSAKey *rsa.PrivateKey `json:",omitempty"`

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
		return errors.WithMessage(err, "failed to load plugin configuration")
	}

	p.updateConfig(func(conf *config) {
		conf.externalConfig = ec
	})

	return nil
}

func (p *Plugin) OnActivate() error {
	conf := p.getConfig()
	user, appErr := p.API.GetUserByUsername(conf.UserName)
	if appErr != nil {
		return errors.WithMessage(appErr, fmt.Sprintf("OnActivate: unable to find user: %s", conf.UserName))
	}

	tpath := filepath.Join(*(p.API.GetConfig().PluginSettings.Directory), manifest.Id, "server", "dist", "templates")

	var err error
	fpath := filepath.Join(tpath, "atlassian-connect.json")
	p.atlassianConnectTemplate, err = template.ParseFiles(fpath)
	if err != nil {
		return errors.WithMessage(err, fmt.Sprintf("OnActivate: failed to parse template: %s", fpath))
	}
	fpath = filepath.Join(tpath, "user-config.html")
	p.userConfigTemplate, err = template.ParseFiles(fpath)
	if err != nil {
		return errors.WithMessage(err, fmt.Sprintf("OnActivate: failed to parse template: %s", fpath))
	}

	conf = p.updateConfig(func(conf *config) {
		conf.botUserID = user.Id
	})

	err = p.API.RegisterCommand(getCommand())
	if err != nil {
		return errors.WithMessage(err, "OnActivate: failed to register command")
	}

	return nil
}

func (p *Plugin) GetPluginURLPath() string {
	return "/plugins/" + manifest.Id
}

func (p *Plugin) GetPluginURL() string {
	return p.GetSiteURL() + p.GetPluginURLPath()
}

func (p *Plugin) GetSiteURL() string {
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
