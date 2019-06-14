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

type Plugin struct {
	plugin.MattermostPlugin

	CurrentInstanceStore CurrentInstanceStore
	InstanceStore        InstanceStore
	UserStore            UserStore
	SecretStore          SecretStore

	// configuration and a muttex to control concurrent access
	Config   Config
	confLock sync.RWMutex
}

// OnConfigurationChange is invoked when configuration changes may have been made.
func (p *Plugin) OnConfigurationChange() error {
	// Load the public configuration fields from the Mattermost server configuration.
	ec := ExternalConfig{}
	err := p.API.LoadPluginConfiguration(&ec)
	if err != nil {
		return errors.WithMessage(err, "failed to load plugin configuration")
	}

	p.UpdateConfig(func(conf *Config) {
		conf.ExternalConfig = ec
	})

	return nil
}

func (p *Plugin) OnActivate() error {
	conf := p.GetConfig()
	user, appErr := p.API.GetUserByUsername(conf.UserName)
	if appErr != nil {
		return errors.WithMessage(appErr, fmt.Sprintf("OnActivate: unable to find user: %s", conf.UserName))
	}

	store := NewStore(p)
	p.CurrentInstanceStore = store
	p.InstanceStore = store
	p.UserStore = store
	p.SecretStore = store

	dir := filepath.Join(*(p.API.GetConfig().PluginSettings.Directory), manifest.Id, "server", "dist", "templates")
	templates, err := p.loadTemplates(dir)
	if err != nil {
		return errors.WithMessage(err, "OnActivate: failed to load templates")
	}

	conf = p.UpdateConfig(func(conf *Config) {
		conf.BotUserID = user.Id
		conf.SiteURL = *p.API.GetConfig().ServiceSettings.SiteURL
		conf.PluginKey = "mattermost_" + regexpNonAlnum.ReplaceAllString(conf.SiteURL, "_")
		conf.PluginURLPath = "/plugins/" + manifest.Id
		conf.PluginURL = strings.TrimRight(conf.SiteURL, "/") + conf.PluginURLPath
		conf.Templates = templates
	})

	err = p.API.RegisterCommand(getCommand())
	if err != nil {
		return errors.WithMessage(err, "OnActivate: failed to register command")
	}

	return nil
}

func (p *Plugin) GetConfig() Config {
	p.confLock.RLock()
	defer p.confLock.RUnlock()
	return p.Config
}

func (p *Plugin) UpdateConfig(f func(conf *Config)) Config {
	p.confLock.Lock()
	defer p.confLock.Unlock()

	f(&p.Config)
	return p.Config
}

func (p *Plugin) loadTemplates(dir string) (map[string]*template.Template, error) {
	templates := make(map[string]*template.Template)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		template, err := template.ParseFiles(path)
		if err != nil {
			p.API.LogError(fmt.Sprintf("OnActivate: failed to parse template %s: %v", path, err))
			return nil
		}
		key := path[len(dir):]
		templates[key] = template
		return nil
	})
	if err != nil {
		return nil, errors.WithMessage(err, "OnActivate: failed to load templates")
	}
	return templates, nil
}
