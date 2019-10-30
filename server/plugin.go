// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"crypto/rsa"
	"fmt"
	"io/ioutil"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"

	jira "github.com/andygrunwald/go-jira"
	"github.com/mattermost/mattermost-server/model"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/plugin"
)

const (
	botUserName    = "jira"
	botDisplayName = "Jira"
	botDescription = "Created by the Jira Plugin."

	// Move these two to the plugin settings if admins need to adjust them.
	WebhookMaxProcsPerServer = 20
	WebhookBufferSize        = 10000
)

var BuildHash = ""
var BuildHashShort = ""
var BuildDate = ""

type externalConfig struct {
	// Setting to turn on/off the webapp components of this plugin
	EnableJiraUI bool `json:"enablejiraui"`

	// Legacy 1.x Webhook secret
	Secret string `json:"secret"`

	// What MM roles that can create subscriptions
	RolesAllowedToEditJiraSubscriptions string

	// Comma separated list of jira groups with permission. Empty is all.
	GroupsAllowedToEditJiraSubscriptions string

	// Maximum attachment size allowed to be uploaded to Jira, can be a
	// number, optionally followed by one of [b, kb, mb, gb, tb]
	MaxAttachmentSize string
}

const currentInstanceTTL = 1 * time.Second
const currentCloudBotClientTTL = 15 * time.Minute

const defaultMaxAttachmentSize = ByteSize(10 * 1024 * 1024) // 10Mb

type config struct {
	// externalConfig caches values from the plugin's settings in the server's config.json
	externalConfig

	// user ID of the bot account
	botUserID string

	// Cached current Jira instance. A non-0 expires indicates the presence
	// of a value. A nil value means there is no instance available.
	currentInstance        Instance
	currentInstanceExpires time.Time

	// Cached non-user Jira cloud client. A non-0 expires indicates the presence
	// of a value. A nil value means there is no client available.
	currentCloudBotClient        *jira.Client
	currentCloudBotClientExpires time.Time

	// Maximum attachment size allowed to be uploaded to Jira
	maxAttachmentSize ByteSize
}

type Plugin struct {
	plugin.MattermostPlugin

	// configuration and a muttex to control concurrent access
	conf     config
	confLock sync.RWMutex

	currentInstanceStore CurrentInstanceStore
	instanceStore        InstanceStore
	userStore            UserStore
	otsStore             OTSStore
	secretsStore         SecretsStore

	// Generated once, then cached in the database, and here deserialized
	RSAKey *rsa.PrivateKey `json:",omitempty"`

	// templates are loaded on startup
	templates map[string]*template.Template

	// channel to distribute work to the webhook processors
	webhookQueue chan []byte
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

	ec.MaxAttachmentSize = strings.TrimSpace(ec.MaxAttachmentSize)
	maxAttachmentSize := defaultMaxAttachmentSize
	if len(ec.MaxAttachmentSize) > 0 {
		maxAttachmentSize, err = ParseByteSize(ec.MaxAttachmentSize)
		if err != nil {
			return errors.WithMessage(err, "failed to load plugin configuration")
		}
	}

	p.updateConfig(func(conf *config) {
		conf.externalConfig = ec
		conf.maxAttachmentSize = maxAttachmentSize
	})

	return nil
}

func (p *Plugin) OnActivate() error {
	botUserID, err := p.Helpers.EnsureBot(&model.Bot{
		Username:    botUserName,
		DisplayName: botDisplayName,
		Description: botDescription,
	})
	if err != nil {
		return errors.Wrap(err, "failed to ensure bot account")
	}

	p.updateConfig(func(conf *config) {
		conf.botUserID = botUserID
	})

	bundlePath, err := p.API.GetBundlePath()
	if err != nil {
		return errors.Wrap(err, "couldn't get bundle path")
	}

	profileImage, err := ioutil.ReadFile(filepath.Join(bundlePath, "assets", "profile.png"))
	if err != nil {
		return errors.Wrap(err, "couldn't read profile image")
	}

	if appErr := p.API.SetProfileImage(botUserID, profileImage); appErr != nil {
		return errors.Wrap(appErr, "couldn't set profile image")
	}

	store := NewStore(p)
	p.currentInstanceStore = store
	p.instanceStore = store
	p.userStore = store
	p.secretsStore = store
	p.otsStore = store

	templates, err := p.loadTemplates(filepath.Join(bundlePath, "assets", "templates"))
	if err != nil {
		return errors.WithMessage(err, "OnActivate: failed to load templates")
	}
	p.templates = templates

	err = p.API.RegisterCommand(getCommand())
	if err != nil {
		return errors.WithMessage(err, "OnActivate: failed to register command")
	}

	// Create our queue of webhook events waiting to be processed.
	p.webhookQueue = make(chan []byte, WebhookBufferSize)

	// Spin up our webhook workers.
	for i := 0; i < WebhookMaxProcsPerServer; i++ {
		go webhookWorker{i, p, p.webhookQueue}.work()
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
	ptr := p.API.GetConfig().ServiceSettings.SiteURL
	if ptr == nil {
		return ""
	}
	return *ptr
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

func (p *Plugin) CheckSiteURL() error {
	ustr := p.GetSiteURL()
	if ustr == "" {
		return errors.Errorf("Mattermost SITEURL must not be empty.")
	}
	u, err := url.Parse(ustr)
	if err != nil {
		return errors.WithMessage(err, "invalid SITEURL")
	}
	if u.Hostname() == "localhost" {
		return errors.Errorf("%s is not a valid Mattermost SITEURL.", ustr)
	}
	return nil
}
