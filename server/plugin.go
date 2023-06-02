// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"text/template"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"

	pluginapi "github.com/mattermost/mattermost-plugin-api"
	"github.com/mattermost/mattermost-plugin-api/experimental/flow"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"

	"github.com/mattermost/mattermost-plugin-autolink/server/autolink"
	"github.com/mattermost/mattermost-plugin-autolink/server/autolinkclient"

	root "github.com/mattermost/mattermost-plugin-jira"
	"github.com/mattermost/mattermost-plugin-jira/server/enterprise"
	"github.com/mattermost/mattermost-plugin-jira/server/telemetry"
	"github.com/mattermost/mattermost-plugin-jira/server/utils"
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

var (
	Manifest     model.Manifest = root.Manifest
	isE2eTesting                = ""
)

type externalConfig struct {
	// Setting to turn on/off the webapp components of this plugin
	EnableJiraUI bool `json:"enablejiraui"`

	// Webhook secret
	Secret string `json:"secret"`

	// What MM roles that can create subscriptions
	RolesAllowedToEditJiraSubscriptions string

	// Comma separated list of jira groups with permission. Empty is all.
	GroupsAllowedToEditJiraSubscriptions string

	// Maximum attachment size allowed to be uploaded to Jira, can be a
	// number, optionally followed by one of [b, kb, mb, gb, tb]
	MaxAttachmentSize string

	// Additional Help Text to be shown in the output of '/jira help' command
	JiraAdminAdditionalHelpText string

	// When enabled, a subscription without security level rules will filter out an issue that has a security level assigned
	SecurityLevelEmptyForJiraSubscriptions bool

	// Hide issue descriptions and comments in Webhook and Subscription messages
	HideDecriptionComment bool

	// Enable slash command autocomplete
	EnableAutocomplete bool

	// Enable Webhook Event Logging
	EnableWebhookEventLogging bool

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

	mattermostSiteURL string
	rsaKey            *rsa.PrivateKey
}

type Plugin struct {
	plugin.MattermostPlugin
	client *pluginapi.Client

	// configuration and a muttex to control concurrent access
	conf     config
	confLock sync.RWMutex

	instanceStore InstanceStore
	userStore     UserStore
	otsStore      OTSStore
	secretsStore  SecretsStore

	setupFlow  *flow.Flow
	oauth2Flow *flow.Flow

	router *mux.Router

	// Generated once, then cached in the database, and here deserialized
	RSAKey *rsa.PrivateKey `json:",omitempty"`

	// templates are loaded on startup
	templates map[string]*template.Template

	// channel to distribute work to the webhook processors
	webhookQueue chan *webhookMessage

	// service that determines if this Mattermost instance has access to
	// enterprise features
	enterpriseChecker enterprise.Checker

	// Telemetry package copied inside repository, should be changed
	// to pluginapi's one (0.1.3+) when min_server_version is safe to point at 7.x

	// telemetry client
	telemetryClient telemetry.Client

	// telemetry Tracker
	tracker telemetry.Tracker
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
	if p.client == nil {
		p.client = pluginapi.NewClient(p.API, p.Driver)
	}
	err := p.client.Configuration.LoadPluginConfiguration(&ec)
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

	// create new tracker on each configuration change
	if p.tracker != nil {
		p.tracker.ReloadConfig(telemetry.NewTrackerConfig(p.API.GetConfig()))
	}

	return nil
}

func (p *Plugin) OnDeactivate() error {
	// close the tracker on plugin deactivation
	if p.telemetryClient != nil {
		err := p.telemetryClient.Close()
		if err != nil {
			return errors.Wrap(err, "OnDeactivate: Failed to close telemetryClient")
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
	p.client = pluginapi.NewClient(p.API, p.Driver)

	p.initializeRouter()

	bundlePath, err := p.client.System.GetBundlePath()
	if err != nil {
		return errors.Wrap(err, "couldn't get bundle path")
	}

	botUserID, err := p.client.Bot.EnsureBot(&model.Bot{
		OwnerId:     Manifest.Id, // Workaround to support older server version affected by https://github.com/mattermost/mattermost-server/pull/21560
		Username:    botUserName,
		DisplayName: botDisplayName,
		Description: botDescription,
	}, pluginapi.ProfileImagePath(filepath.Join("assets", "profile.png")))
	if err != nil {
		return errors.Wrap(err, "failed to ensure bot account")
	}

	mattermostSiteURL := ""
	ptr := p.client.Configuration.GetConfig().ServiceSettings.SiteURL
	if ptr != nil {
		mattermostSiteURL = *ptr
	}

	err = p.setDefaultConfiguration()
	if err != nil {
		return errors.Wrap(err, "failed to set default configuration")
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

	instances, err := MigrateV2Instances(p)
	if err != nil {
		return errors.WithMessage(err, "OnActivate: failed to migrate from previous version of the Jira plugin")
	}

	templates, err := p.loadTemplates(filepath.Join(bundlePath, "assets", "templates"))
	if err != nil {
		return err
	}
	p.templates = templates

	p.setupFlow = p.NewSetupFlow()
	p.oauth2Flow = p.NewOAuth2Flow()

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

	go func() {
		for _, url := range instances.IDs() {
			var instance Instance
			instance, err = p.instanceStore.LoadInstance(url)
			if err != nil {
				continue
			}

			ci, ok := instance.(*cloudInstance)
			if !ok {
				p.client.Log.Info("only cloud instances supported for autolink", "err", err)
				continue
			}
			var status *model.PluginStatus
			status, err = p.client.Plugin.GetPluginStatus(autolinkPluginID)
			if err != nil {
				p.client.Log.Warn("OnActivate: Autolink plugin unavailable. API returned error", "error", err.Error())
				continue
			}
			if status.State != model.PluginStateRunning {
				p.client.Log.Warn("OnActivate: Autolink plugin unavailable. Plugin is not running", "status", status)
				continue
			}

			if err = p.AddAutolinksForCloudInstance(ci); err != nil {
				p.client.Log.Info("could not install autolinks for cloud instance", "instance", ci.BaseURL, "err", err)
				continue
			}
		}
	}()

	p.initializeTelemetry()

	return nil
}

func (p *Plugin) AddAutolinksForCloudInstance(ci *cloudInstance) error {
	client, err := ci.getClientForBot()
	if err != nil {
		return fmt.Errorf("unable to get jira client for server: %w", err)
	}

	plist, err := jiraCloudClient{JiraClient{Jira: client}}.ListProjects("", -1, false)
	if err != nil {
		return fmt.Errorf("unable to get project keys: %w", err)
	}

	for _, proj := range plist {
		key := proj.Key
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
	return "/plugins/" + Manifest.Id
}

func (p *Plugin) GetPluginURL() string {
	return strings.TrimRight(p.GetSiteURL(), "/") + p.GetPluginURLPath()
}

func (p *Plugin) GetSiteURL() string {
	return p.getConfig().mattermostSiteURL
}

func (p *Plugin) debugf(f string, args ...interface{}) {
	p.client.Log.Debug(fmt.Sprintf(f, args...))
}

func (p *Plugin) infof(f string, args ...interface{}) {
	p.client.Log.Info(fmt.Sprintf(f, args...))
}

func (p *Plugin) errorf(f string, args ...interface{}) {
	p.client.Log.Error(fmt.Sprintf(f, args...))
}

func (p *Plugin) CheckSiteURL() error {
	ustr := p.GetSiteURL()
	if ustr == "" {
		return errors.New("Mattermost SITEURL must not be empty.")
	}
	u, err := url.Parse(ustr)
	if err != nil {
		return errors.WithMessage(err, "invalid SITEURL")
	}
	if isE2eTesting != "true" && u.Hostname() == "localhost" {
		return errors.Errorf("Using %s as your Mattermost SiteURL is not permitted, as the URL is not reachable from Jira. If you are using Jira Cloud, please make sure your URL is reachable from the public internet.", ustr)
	}
	return nil
}

func (p *Plugin) storeConfig(ec externalConfig) error {
	var out map[string]interface{}
	data, err := json.Marshal(ec)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, &out)
	if err != nil {
		return err
	}

	return p.client.Configuration.SavePluginConfig(out)
}

func generateSecret() (string, error) {
	b := make([]byte, 256)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	s := base64.RawStdEncoding.EncodeToString(b)

	s = s[:32]

	return s, nil
}

func (c *externalConfig) setDefaults() (bool, error) {
	changed := false

	if c.Secret == "" {
		secret, err := generateSecret()
		if err != nil {
			return false, err
		}
		c.Secret = secret
		changed = true
	}

	return changed, nil
}

func (p *Plugin) setDefaultConfiguration() error {
	ec := p.getConfig().externalConfig
	changed, err := ec.setDefaults()
	if err != nil {
		return err
	}

	if changed {
		err := p.storeConfig(ec)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Plugin) OnInstall(c *plugin.Context, event model.OnInstallEvent) error {
	instances, err := p.instanceStore.LoadInstances()
	if err != nil {
		return err
	}

	if instances.Len() == 0 {
		return p.setupFlow.ForUser(event.UserId).Start(nil)
	}

	return nil
}
