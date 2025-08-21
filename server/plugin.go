// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	htmlTemplate "html/template"
	"math"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	textTemplate "text/template"

	"github.com/andygrunwald/go-jira"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/mattermost/mattermost/server/public/pluginapi/experimental/flow"

	"github.com/mattermost-community/mattermost-plugin-autolink/server/autolink"
	"github.com/mattermost-community/mattermost-plugin-autolink/server/autolinkclient"

	"github.com/mattermost/mattermost-plugin-jira/server/enterprise"
	"github.com/mattermost/mattermost-plugin-jira/server/telemetry"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
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

	// The encryption key used to encrypt stored api tokens
	EncryptionKey string

	// API token from Jira
	AdminAPIToken string

	// Email of the admin
	AdminEmail string

	// Number of days Jira comments will be posted as threaded replies instead of a new post
	ThreadedJiraCommentSubscriptionDuration string `json:"threadedjiracommentsubscriptionduration"`

	// Comma separated list of Team IDs and name to be used for filtering subscription on the basis of teams. Ex: [team-1-name](team-1-id),[team-2-name](team-2-id)
	TeamIDs string `json:"teamids"`

	TeamIDList []TeamList `json:"teamidlist"`
}

type TeamList struct {
	Name string
	ID   string
}

const defaultMaxAttachmentSize = types.ByteSize(100 * 1024 * 1024) // 100Mb

type config struct {
	// externalConfig caches values from the plugin's settings in the server's config.json
	externalConfig

	// user ID of the bot account
	botUserID string

	// Maximum attachment size allowed to be uploaded to Jira
	maxAttachmentSize types.ByteSize

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
	htmlTemplates map[string]*htmlTemplate.Template
	textTemplates map[string]*textTemplate.Template

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

func isValidUUIDv4(uuid string) bool {
	// UUIDv4 format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
	re := regexp.MustCompile(`^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[89abAB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$`)
	return re.MatchString(uuid)
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
	mattermostMaxAttachmentSize := p.API.GetConfig().FileSettings.MaxFileSize
	if mattermostMaxAttachmentSize != nil {
		maxAttachmentSize = types.ByteSize(*mattermostMaxAttachmentSize)
	}
	if len(ec.MaxAttachmentSize) > 0 {
		maxAttachmentSize, err = types.ParseByteSize(ec.MaxAttachmentSize)
		if err != nil {
			return errors.WithMessage(err, "failed to load plugin configuration")
		}
	}

	jsonBytes, err := json.Marshal(ec.AdminAPIToken)
	if err != nil {
		p.client.Log.Warn("Error marshaling the admin API token", "error", err.Error())
		return err
	}

	encryptionKey := ec.EncryptionKey
	if ec.AdminAPIToken != "" && encryptionKey == "" {
		p.client.Log.Warn("Encryption key required to encrypt admin API token")
		return errors.New("failed to encrypt admin token. Encryption key not generated")
	}

	encryptedAdminAPIToken, err := encrypt(jsonBytes, []byte(encryptionKey))
	if err != nil {
		p.client.Log.Warn("Error encrypting the admin API token", "error", err.Error())
		return err
	}
	ec.AdminAPIToken = string(encryptedAdminAPIToken)

	duration, err := strconv.Atoi(ec.ThreadedJiraCommentSubscriptionDuration)
	if err != nil {
		return errors.New("error converting comment post reply duration to integer")
	}
	if duration < 0 {
		return errors.New("comment post reply duration cannot be negative")
	}

	if ec.TeamIDs != "" {
		teamListData := strings.Split(ec.TeamIDs, ",")
		re := regexp.MustCompile(`^\[(.*?)\]\((.*?)\)$`)

		var teamIDList []TeamList
		var errorPrinted bool
		var acceptedLength int

		for _, item := range teamListData {
			item = strings.TrimSpace(item)
			match := re.FindStringSubmatch(item)

			if len(match) != 3 {
				if !errorPrinted {
					p.client.Log.Warn("Please provide a valid list of team name and ID")
					errorPrinted = true
				}
				continue
			}

			teamName := strings.TrimSpace(match[1])
			teamID := strings.TrimSpace(match[2])

			if teamName == "" || !isValidUUIDv4(teamID) {
				if !errorPrinted {
					p.client.Log.Warn("Please provide a valid list of team name and ID")
					errorPrinted = true
				}

				continue
			}

			teamIDList = append(teamIDList, TeamList{
				Name: teamName,
				ID:   teamID,
			})

			// Add length for accepted entry:
			acceptedLength += len(teamName) + len(teamID) + 4 // +4 for [ ] ( )
		}

		// Add length for commas (only between items)
		if len(teamIDList) > 0 {
			acceptedLength += len(teamIDList) - 1
		}

		if acceptedLength != len(ec.TeamIDs) {
			p.client.Log.Warn("Some team entries were invalid and ignored")
		}

		ec.TeamIDList = teamIDList
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
		OwnerId:     manifest.Id, // Workaround to support older server version affected by https://github.com/mattermost/mattermost-server/pull/21560
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
	} else {
		return errors.New("please configure the Mattermost server's SiteURL, then restart the plugin.")
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

	htmlTemplates, textTemplates, err := p.loadTemplates(filepath.Join(bundlePath, "assets", "templates"))
	if err != nil {
		return err
	}
	p.htmlTemplates = htmlTemplates
	p.textTemplates = textTemplates

	setupFlow, err := p.NewSetupFlow()
	if err != nil {
		return err
	}
	p.setupFlow = setupFlow

	oauth2Flow, err := p.NewOAuth2Flow()
	if err != nil {
		return err
	}
	p.oauth2Flow = oauth2Flow

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
		p.SetupAutolink(instances)
	}()

	p.initializeTelemetry()

	return nil
}

func (p *Plugin) SetupAutolink(instances *Instances) {
	for _, url := range instances.IDs() {
		var instance Instance
		instance, err := p.instanceStore.LoadInstance(url)
		if err != nil {
			continue
		}

		if p.getConfig().AdminAPIToken == "" || p.getConfig().AdminEmail == "" {
			p.client.Log.Info("unable to setup autolink due to missing API Token or Admin Email")
			continue
		}

		switch instance.(type) {
		case *cloudInstance, *cloudOAuthInstance:
		default:
			p.client.Log.Info("only cloud and cloud-oauth instances supported for autolink")
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

		switch instance := instance.(type) {
		case *cloudInstance:
			if err = p.AddAutolinksForCloudInstance(instance); err != nil {
				p.client.Log.Info("could not install autolinks for cloud instance", "instance", instance.BaseURL, "error", err.Error())
			} else {
				p.client.Log.Info("successfully installed autolinks for cloud instance", "instance", instance.BaseURL)
			}
		case *cloudOAuthInstance:
			if err = p.AddAutolinksForCloudOAuthInstance(instance); err != nil {
				p.client.Log.Info("could not install autolinks for cloud-oauth instance", "instance", instance.JiraBaseURL, "error", err.Error())
			} else {
				p.client.Log.Info("successfully installed autolinks for cloud-oauth instance", "instance", instance.JiraBaseURL)
			}
		}
	}
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

	return p.AddAutoLinkForProjects(plist, ci.BaseURL)
}

func (p *Plugin) AddAutolinksForCloudOAuthInstance(coi *cloudOAuthInstance) error {
	plist, err := p.GetProjectListWithAPIToken(string(coi.InstanceID))
	if err != nil {
		return fmt.Errorf("error getting project list: %w", err)
	}

	return p.AddAutoLinkForProjects(*plist, coi.JiraBaseURL)
}

func (p *Plugin) AddAutoLinkForProjects(plist jira.ProjectList, baseURL string) error {
	var err error
	for _, proj := range plist {
		key := proj.Key
		err = p.AddAutolinks(key, baseURL)
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
		// Do not return an error if the status code is 304 (indicating that the autolink for this project is already installed).
		if !strings.Contains(err.Error(), `Error: 304, {"status": "OK"}`) {
			return fmt.Errorf("unable to add autolinks: %w", err)
		}
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

func (p *Plugin) CreateFullURLPath(extensionPath string) string {
	return fmt.Sprintf("%s%s%s", p.GetSiteURL(), p.GetPluginURLPath(), extensionPath)
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
	if u.Hostname() == "localhost" {
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

	if c.EncryptionKey == "" {
		encryptionKey, err := generateSecret()
		if err != nil {
			return false, err
		}
		c.EncryptionKey = encryptionKey
		changed = true
	}

	return changed, nil
}

func (p *Plugin) setDefaultConfiguration() error {
	ec := externalConfig{}
	err := p.client.Configuration.LoadPluginConfiguration(&ec)
	if err != nil {
		return errors.WithMessage(err, "failed to load plugin configuration")
	}

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
