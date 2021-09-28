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

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"

	"github.com/mattermost/mattermost-plugin-autolink/server/autolink"
	"github.com/mattermost/mattermost-plugin-autolink/server/autolinkclient"

	"github.com/mattermost/mattermost-plugin-jira/server/enterprise"
	"github.com/mattermost/mattermost-plugin-jira/server/expvar"
	"github.com/mattermost/mattermost-plugin-jira/server/tracker"
	"github.com/mattermost/mattermost-plugin-jira/server/utils"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/telemetry"
)

const (
	botUserName    = "jira"
	botDisplayName = "Jira"
	botDescription = "Created by the Jira Plugin."

	autolinkPluginID = "mattermost-autolink"

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

	// Enable slash command autocomplete
	EnableAutocomplete bool

	// Display subscription name in notifications
	DisplaySubscriptionNameInNotifications bool
}

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

	// Generated once, then cached in the database, and here deserialized
	RSAKey *rsa.PrivateKey `json:",omitempty"`

	// templates are loaded on startup
	templates map[string]*template.Template

	// channel to distribute work to the webhook processors
	webhookQueue chan *webhookMessage

	// telemetry client
	telemetryClient telemetry.Client

	// telemetry Tracker
	Tracker tracker.Tracker

	// service that determines if this Mattermost instance has access to
	// enterprise features
	enterpriseChecker enterprise.Checker
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

	prev := p.getConfig()
	p.updateConfig(func(conf *config) {
		conf.externalConfig = ec
		conf.maxAttachmentSize = maxAttachmentSize
	})

	// OnConfigurationChanged is first called before the plugin is activated,
	// in this case don't register the command, let Activate do it, it has the instanceStore.
	// TODO: consider moving (some? stores? all?) initialization into the first OnConfig instead of OnActivate.
	if prev.EnableAutocomplete != ec.EnableAutocomplete && p.instanceStore != nil {
		instances, err := p.instanceStore.LoadInstances()
		if err != nil {
			return err
		}
		err = p.registerJiraCommand(ec.EnableAutocomplete, instances.Len() > 1)
		if err != nil {
			return err
		}
	}

	diagnostics := false
	if p.API.GetConfig().LogSettings.EnableDiagnostics != nil {
		diagnostics = *p.API.GetConfig().LogSettings.EnableDiagnostics
	}

	// create new tracker on each configuration change
	p.Tracker = tracker.New(telemetry.NewTracker(
		p.telemetryClient,
		p.API.GetDiagnosticId(),
		p.API.GetServerVersion(),
		manifest.ID,
		manifest.Version,
		diagnostics,
	))

	return nil
}

func (p *Plugin) OnDeactivate() error {
	// close the tracker on plugin deactivation
	if p.telemetryClient != nil {
		err := p.telemetryClient.Close()
		if err != nil {
			return errors.Wrap(err, "OnDeactivate: Failed to close telemetryClient.")
		}
	}
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

	instances, err := MigrateV2Instances(p)
	if err != nil {
		return errors.WithMessage(err, "OnActivate: failed to migrate from previous version of the Jira plugin")
	}

	templates, err := p.loadTemplates(filepath.Join(bundlePath, "assets", "templates"))
	if err != nil {
		return err
	}
	p.templates = templates

	// Register /jira command and stash the loaded list of known instances for
	// later (autolink registration).
	err = p.registerJiraCommand(p.getConfig().EnableAutocomplete, instances.Len() > 1)
	if err != nil {
		return errors.Wrap(err, "OnActivate")
	}

	// Create our queue of webhook events waiting to be processed.
	p.webhookQueue = make(chan *webhookMessage, WebhookBufferSize)

	// Spin up our webhook workers.
	for i := 0; i < WebhookMaxProcsPerServer; i++ {
		go webhookWorker{i, p, p.webhookQueue}.work()
	}

	p.enterpriseChecker = enterprise.NewEnterpriseChecker(p.API)

	go p.initStats()

	go func() {
		for _, url := range instances.IDs() {
			var instance Instance
			instance, err = p.instanceStore.LoadInstance(url)
			if err != nil {
				continue
			}

			ci, ok := instance.(*cloudInstance)
			if !ok {
				p.API.LogInfo("only cloud instances supported for autolink", "err", err)
				continue
			}

			status, apiErr := p.API.GetPluginStatus(autolinkPluginID)
			if apiErr != nil {
				p.API.LogWarn("OnActivate: Autolink plugin unavailable. API returned error", "error", apiErr.Error())
				continue
			}
			if status.State != model.PluginStateRunning {
				p.API.LogWarn("OnActivate: Autolink plugin unavailable. Plugin is not running", "status", status)
				continue
			}

			if err = p.AddAutolinksForCloudInstance(ci); err != nil {
				p.API.LogInfo("could not install autolinks for cloud instance", "instance", ci.BaseURL, "err", err)
				continue
			}
		}
	}()

	// initialize the rudder client once on activation
	p.telemetryClient, err = telemetry.NewRudderClient()
	if err != nil {
		p.API.LogError("Cannot create telemetry client. err=%v", err)
	}

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
		return fmt.Errorf("unable to add autolinks: %w", err)
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
	return "/plugins/" + manifest.ID
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
		return errors.Errorf("Using %s as your Mattermost SiteURL is not permitted, as the URL is not reachable from Jira. If you are using Jira Cloud, please make sure your URL is reachable from the public internet.", ustr)
	}
	return nil
}
