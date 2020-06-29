// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"crypto/rsa"
	"fmt"
	"io/ioutil"
	"math"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"

	"github.com/mattermost/mattermost-plugin-autolink/server/autolink"
	"github.com/mattermost/mattermost-plugin-autolink/server/autolinkclient"
	"github.com/mattermost/mattermost-plugin-jira/server/enterprise"
	"github.com/mattermost/mattermost-plugin-jira/server/expvar"
	"github.com/mattermost/mattermost-plugin-jira/server/utils"
)

const (
	botUserName    = "jira"
	botDisplayName = "Jira"
	botDescription = "Created by the Jira Plugin."

	autolinkPluginId = "mattermost-autolink"

	// Move these two to the plugin settings if admins need to adjust them.
	WebhookMaxProcsPerServer = 20
	WebhookBufferSize        = 10000
	PluginRepo               = "https://github.com/mattermost/mattermost-plugin-jira"
)

var BuildHash = ""
var BuildHashShort = ""
var BuildDate = ""

type externalConfig struct {
	// Setting to turn on/off the webapp components of this plugin
	EnableJiraUI bool `json:"enablejiraui"`

	// Legacy 1.x Webhook secret
	Secret string `json:"secret"`

	// Stats API secret
	StatsSecret string `json:"stats_secret"`

	// What MM roles that can create subscriptions
	RolesAllowedToEditJiraSubscriptions string

	// Comma separated list of jira groups with permission. Empty is all.
	GroupsAllowedToEditJiraSubscriptions string

	// Maximum attachment size allowed to be uploaded to Jira, can be a
	// number, optionally followed by one of [b, kb, mb, gb, tb]
	MaxAttachmentSize string

	// Disable statistics gathering
	DisableStats bool `json:"disable_stats"`

	// Additional Help Text to be shown in the output of '/jira help' command
	JiraAdminAdditionalHelpText string

	// Hide issue descriptions and comments in Webhook and Subscription messages
	HideDecriptionComment bool
}

const currentInstanceTTL = 1 * time.Second

const defaultMaxAttachmentSize = utils.ByteSize(10 * 1024 * 1024) // 10Mb

type config struct {
	// externalConfig caches values from the plugin's settings in the server's config.json
	externalConfig

	// user ID of the bot account
	botUserID string

	// Maximum attachment size allowed to be uploaded to Jira
	maxAttachmentSize utils.ByteSize

	stats             *expvar.Stats
	statsStopAutosave chan bool

	mattermostSiteURL string
	rsaKey            *rsa.PrivateKey
}

type Plugin struct {
	plugin.MattermostPlugin

	// configuration and a muttex to control concurrent access
	conf     config
	confLock sync.RWMutex

	instanceStore InstanceStore
	userStore     UserStore
	otsStore      OTSStore
	secretsStore  SecretsStore

	// Active workflows store
	workflowTriggerStore *TriggerStore

	// Generated once, then cached in the database, and here deserialized
	RSAKey *rsa.PrivateKey `json:",omitempty"`

	// templates are loaded on startup
	templates map[string]*template.Template

	// channel to distribute work to the webhook processors
	webhookQueue chan *webhookMessage

	// service that determiines if this Mattermost instance has access to
	// enterprise features
	enterpriseChecker enterprise.EnterpriseChecker
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
		maxAttachmentSize, err = utils.ParseByteSize(ec.MaxAttachmentSize)
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
	store := NewStore(p)
	p.instanceStore = store
	p.userStore = store
	p.secretsStore = store
	p.otsStore = store

	botUserID, err := p.Helpers.EnsureBot(&model.Bot{
		Username:    botUserName,
		DisplayName: botDisplayName,
		Description: botDescription,
	})
	if err != nil {
		return errors.Wrap(err, "failed to ensure bot account")
	}

	mattermostSiteURL := ""
	ptr := p.API.GetConfig().ServiceSettings.SiteURL
	if ptr != nil {
		mattermostSiteURL = *ptr
	}

	rsaKey, err := p.secretsStore.EnsureRSAKey()
	if err != nil {
		return errors.WithMessage(err, "OnActivate: failed to make RSA public key")
	}

	p.updateConfig(func(conf *config) {
		conf.botUserID = botUserID
		conf.mattermostSiteURL = mattermostSiteURL
		conf.rsaKey = rsaKey
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

	err = store.MigrateV2Instances()
	if err != nil {
		return errors.WithMessage(err, "OnActivate: failed to migrate from previous version of the Jira plugin")
	}

	templates, err := p.loadTemplates(filepath.Join(bundlePath, "assets", "templates"))
	if err != nil {
		return errors.WithMessage(err, "OnActivate: failed to load templates")
	}
	p.templates = templates

	// Register /jira command and stash the loaded list of known instances for
	// later (autolink registration).
	instances, err := p.registerJiraCommand()
	if err != nil {
		return errors.Wrap(err, "OnActivate")
	}

	// Create our queue of webhook events waiting to be processed.
	p.webhookQueue = make(chan *webhookMessage, WebhookBufferSize)

	// Spin up our webhook workers.
	for i := 0; i < WebhookMaxProcsPerServer; i++ {
		go webhookWorker{i, p, p.webhookQueue}.work()
	}

	p.workflowTriggerStore = NewTriggerStore()

	p.enterpriseChecker = enterprise.NewEnterpriseChecker(p.API)

	go p.initStats()

	go func() {
		for _, url := range instances.IDs() {
			instance, err := p.instanceStore.LoadInstance(url)
			if err != nil {
				continue
			}

			ci, ok := instance.(*cloudInstance)
			if !ok {
				p.API.LogWarn("only cloud instances supported for autolink", "err", err)
				continue
			}

			if err := p.AddAutolinksForCloudInstance(ci); err != nil {
				p.API.LogWarn("could not install autolinks for cloud instance", "instance", ci.BaseURL, "err", err)
				continue
			}
		}
	}()

	return nil
}

func (p *Plugin) AddAutolinksForCloudInstance(ci *cloudInstance) error {
	client, err := ci.getClientForBot()
	if err != nil {
		return fmt.Errorf("unable to get jira client for server: %w", err)
	}

	keys, err := JiraClient{Jira: client}.GetAllProjectKeys()
	if err != nil {
		return fmt.Errorf("unable to get project keys: %w", err)
	}

	for _, key := range keys {
		err = p.AddAutolinks(key, ci.BaseURL)
	}
	if err != nil {
		return fmt.Errorf("some keys were not installed: %w", err)
	}

	return nil
}

func (p *Plugin) AddAutolinks(key, baseURL string) error {
	baseURL = strings.TrimRight(baseURL, "/")
	installList := []autolink.Autolink{
		{
			Name:     key + " key to link for " + baseURL,
			Pattern:  `(` + key + `)(-)(?P<jira_id>\d+)`,
			Template: `[` + key + `-${jira_id}](` + baseURL + `/browse/` + key + `-${jira_id})`,
		},
		{
			Name:     key + " link to key for " + baseURL,
			Pattern:  `(` + strings.ReplaceAll(baseURL, ".", `\.`) + `/browse/)(` + key + `)(-)(?P<jira_id>\d+)`,
			Template: `[` + key + `-${jira_id}](` + baseURL + `/browse/` + key + `-${jira_id})`,
		},
	}

	client := autolinkclient.NewClientPlugin(p.API)
	if err := client.Add(installList...); err != nil {
		return fmt.Errorf("Unable to add autolinks: %w", err)
	}

	return nil
}

var regexpNonAlnum = regexp.MustCompile("[^a-zA-Z0-9]+")

func (p *Plugin) GetPluginKey() string {
	sURL := p.GetSiteURL()
	prefix := "mattermost_"
	escaped := regexpNonAlnum.ReplaceAllString(sURL, "_")

	start := len(escaped) - int(math.Min(float64(len(escaped)), 32))
	return prefix + escaped[start:]
}

func (p *Plugin) GetPluginURLPath() string {
	return "/plugins/" + manifest.Id
}

func (p *Plugin) GetPluginURL() string {
	return strings.TrimRight(p.GetSiteURL(), "/") + p.GetPluginURLPath()
}

func (p *Plugin) GetSiteURL() string {
	return p.getConfig().mattermostSiteURL
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
