// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"crypto/rsa"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

	// JIRAServerURL needs to be configured to run in the JIRA Server
	// mode.
	// TODO If JIRAServerURL is configured, add/select the instance as needed
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

	// templates are loaded on startup
	templates map[string]*template.Template
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

	templates := make(map[string]*template.Template)
	dir := filepath.Join(*(p.API.GetConfig().PluginSettings.Directory), manifest.Id, "server", "dist", "templates")

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		template, err := template.ParseFiles(path)
		if err != nil {
			p.errorf("OnActivate: failed to parse template %s: %v", path, err)
			return nil
		}
		key := path[len(dir):]
		templates[key] = template
		p.debugf("loaded template %s", key)
		return nil
	})
	if err != nil {
		return errors.WithMessage(err, "OnActivate: failed to load templates")
	}
	p.templates = templates

	conf = p.updateConfig(func(conf *config) {
		conf.botUserID = user.Id
	})

	err = p.API.RegisterCommand(getCommand())
	if err != nil {
		return errors.WithMessage(err, "OnActivate: failed to register command")
	}

	return nil
}

func (p Plugin) GetPluginKey() string {
	return "mattermost_" + regexpNonAlnum.ReplaceAllString(p.GetSiteURL(), "_")
}
func (p Plugin) GetPluginURLPath() string {
	return "/plugins/" + manifest.Id
}

func (p Plugin) GetPluginURL() string {
	return strings.TrimRight(p.GetSiteURL(), "/") + p.GetPluginURLPath()
}

func (p Plugin) GetSiteURL() string {
	return *p.API.GetConfig().ServiceSettings.SiteURL
}

func (p Plugin) debugf(f string, args ...interface{}) {
	p.API.LogDebug(fmt.Sprintf(f, args...))
}

func (p Plugin) infof(f string, args ...interface{}) {
	p.API.LogInfo(fmt.Sprintf(f, args...))
}

func (p Plugin) errorf(f string, args ...interface{}) {
	p.API.LogError(fmt.Sprintf(f, args...))
}
