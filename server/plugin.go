// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
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
	UserName string `json:"username"`

	// Legacy 1.x Webhook secret
	Secret string `json:"secret"`
}

type config struct {
	// externalConfig caches values from the plugin's settings in the server's config.json
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

// saveConfigToServer persists the externalConfig portion of the plugin config to the server's config.json
func (p *Plugin) saveConfigToServer() error {
	b, err := json.Marshal(p.getConfig().externalConfig)
	if err != nil {
		return errors.Errorf("failed to Marshal externalConfig to bytes: %v", err)
	}

	mapString := make(map[string]interface{})
	if err = json.Unmarshal(b, &mapString); err != nil {
		return errors.Errorf("failed to Unmarshal bytes to a map[string]interface{}: %v", err)
	}

	if err = p.API.SavePluginConfig(mapString); err != nil {
		return errors.Errorf("failed to savePluginConfig: %v", err)
	}

	return nil
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

	dir := filepath.Join(*(p.API.GetConfig().PluginSettings.Directory), manifest.Id, "server", "dist", "templates")
	templates, err := p.loadTemplates(dir)
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

func (p *Plugin) GetPluginKey() string {
	return "mattermost_" + regexpNonAlnum.ReplaceAllString(p.GetSiteURL(), "_")
}
func (p *Plugin) GetPluginURLPath() string {
	return "/plugins/" + manifest.Id
}

func (p *Plugin) GetPluginURL() string {
	return strings.TrimRight(p.GetSiteURL(), "/") + p.GetPluginURLPath()
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
