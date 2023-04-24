// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	jira "github.com/andygrunwald/go-jira"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/pkg/errors"
	ajwt "github.com/rbriski/atlassian-jwt"
	"golang.org/x/oauth2"
	oauth2_jira "golang.org/x/oauth2/jira"

	"github.com/mattermost/mattermost-plugin-jira/server/utils"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

type cloudInstance struct {
	*InstanceCommon

	// Initially a new instance is created with an expiration time. The
	// admin is expected to upload it to the Jira instance, and we will
	// then receive a /installed callback that initializes the instance
	// and makes it permanent. No subsequent /installed will be accepted
	// for the instance.
	Installed bool

	// For cloud instances (atlassian-connect.json install and user auth)
	RawAtlassianSecurityContext string
	*AtlassianSecurityContext   `json:"-"`
}

var _ Instance = (*cloudInstance)(nil)

type AtlassianSecurityContext struct {
	Key            string `json:"key"`
	ClientKey      string `json:"clientKey"`
	SharedSecret   string `json:"sharedSecret"`
	ServerVersion  string `json:"serverVersion"`
	PluginsVersion string `json:"pluginsVersion"`
	BaseURL        string `json:"baseUrl"`
	ProductType    string `json:"productType"`
	Description    string `json:"description"`
	EventType      string `json:"eventType"`
	OAuthClientID  string `json:"oauthClientId"`
}

func newCloudInstance(p *Plugin, key types.ID, installed bool, rawASC string, asc *AtlassianSecurityContext) *cloudInstance {
	return &cloudInstance{
		InstanceCommon:              newInstanceCommon(p, CloudInstanceType, key),
		Installed:                   installed,
		RawAtlassianSecurityContext: rawASC,
		AtlassianSecurityContext:    asc,
	}
}

func (p *Plugin) installInactiveCloudInstance(rawURL string, actingUserID string) (string, error) {
	jiraURL, err := utils.CheckJiraURL(p.GetSiteURL(), rawURL, false)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(jiraURL, "https://") {
		return "", errors.New("a secure HTTPS URL is required")
	}

	instances, _ := p.instanceStore.LoadInstances()
	if !p.enterpriseChecker.HasEnterpriseFeatures() {
		if instances != nil && len(instances.IDs()) > 0 {
			return "", errors.New(licenseErrorString)
		}
	}

	// Create an "uninitialized" instance of Jira Cloud that will
	// receive the /installed callback
	err = p.instanceStore.CreateInactiveCloudInstance(types.ID(jiraURL), actingUserID)
	if err != nil {
		return "", err
	}

	return jiraURL, err
}

func (ci *cloudInstance) GetMattermostKey() string {
	return ci.AtlassianSecurityContext.Key
}

func (ci *cloudInstance) GetDisplayDetails() map[string]string {
	if !ci.Installed {
		return map[string]string{
			"Setup": "In progress",
		}
	}

	return map[string]string{
		"Atlassian Connect Key":        ci.AtlassianSecurityContext.Key,
		"Atlassian Connect Client Key": ci.AtlassianSecurityContext.ClientKey,
		"Jira Cloud Version":           ci.AtlassianSecurityContext.ServerVersion,
		"Jira Cloud Plugins Version":   ci.AtlassianSecurityContext.PluginsVersion,
	}
}

func (ci *cloudInstance) GetUserConnectURL(mattermostUserID string) (string, *http.Cookie, error) {
	// Create JWT secret we use in Jira's connect URL params
	randomBytes1 := make([]byte, 32)
	_, err := rand.Read(randomBytes1)
	if err != nil {
		return "", nil, err
	}
	jwtSecret := strings.ReplaceAll(fmt.Sprintf("%x", randomBytes1), "-", "_")

	randomBytes2 := make([]byte, 32)
	_, err = rand.Read(randomBytes2)
	if err != nil {
		return "", nil, err
	}

	// Create cookie secret to act as a cross-reference of user integrity
	cookieSecret := strings.ReplaceAll(fmt.Sprintf("%x", randomBytes2), "-", "_")
	cookie, err := ci.Plugin.createCookieFromSecret(cookieSecret)
	if err != nil {
		return "", nil, err
	}

	// Store JWT and cookie secret together in KV store
	storedSecret := jwtSecret + "-" + cookieSecret
	err = ci.Plugin.otsStore.StoreOneTimeSecret(mattermostUserID, storedSecret)
	if err != nil {
		return "", nil, err
	}

	token, err := ci.Plugin.NewEncodedAuthToken(mattermostUserID, jwtSecret)
	if err != nil {
		return "", nil, err
	}

	v := url.Values{}
	v.Add(argMMToken, token)
	connectURL := fmt.Sprintf("%s/login?dest-url=%s/plugins/servlet/ac/%s/%s?%v",
		ci.GetURL(), ci.GetURL(), ci.AtlassianSecurityContext.Key, userRedirectPageKey, v.Encode())

	return connectURL, cookie, nil
}

func (p *Plugin) createCookieFromSecret(secret string) (*http.Cookie, error) {
	siteURL := p.GetSiteURL()
	u, err := url.Parse(siteURL)
	if err != nil {
		return nil, err
	}

	domain := u.Hostname()
	path := u.Path
	if path == "" {
		path = "/"
	}

	maxAge := 15 * 60 // 15 minutes
	expiresAt := time.Unix(model.GetMillis()/1000+int64(maxAge), 0)

	cookie := &http.Cookie{
		Name:     cookieSecretName,
		Value:    secret,
		Path:     path,
		MaxAge:   maxAge,
		Expires:  expiresAt,
		HttpOnly: true,
		Domain:   domain,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
	}

	return cookie, nil
}

func (ci *cloudInstance) GetURL() string {
	return ci.AtlassianSecurityContext.BaseURL
}

func (ci *cloudInstance) GetJiraBaseURL() string {
	return ci.GetURL()
}

func (ci *cloudInstance) GetManageAppsURL() string {
	return fmt.Sprintf("%s/plugins/servlet/upm", ci.GetURL())
}

func (ci *cloudInstance) GetManageWebhooksURL() string {
	return cloudManageWebhooksURL(ci.GetURL())
}

func cloudManageWebhooksURL(jiraURL string) string {
	return fmt.Sprintf("%s/plugins/servlet/webhooks", jiraURL)
}

func (ci *cloudInstance) GetClient(connection *Connection) (Client, error) {
	client, _, err := ci.getClientForConnection(connection)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get Jira client for user "+connection.DisplayName)
	}
	return newCloudClient(client), nil
}

// Creates a client for acting on behalf of a user
func (ci *cloudInstance) getClientForConnection(connection *Connection) (*jira.Client, *http.Client, error) {
	oauth2Conf := oauth2_jira.Config{
		BaseURL: ci.GetURL(),
		Subject: connection.AccountID,
		Config: oauth2.Config{
			ClientID:     ci.AtlassianSecurityContext.OAuthClientID,
			ClientSecret: ci.AtlassianSecurityContext.SharedSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://auth.atlassian.io",
				TokenURL: "https://auth.atlassian.io/oauth2/token",
			},
		},
	}

	conf := ci.getConfig()
	httpClient := oauth2Conf.Client(context.Background())
	httpClient = utils.WrapHTTPClient(httpClient,
		utils.WithRequestSizeLimit(conf.maxAttachmentSize),
		utils.WithResponseSizeLimit(conf.maxAttachmentSize))

	jiraClient, err := jira.NewClient(httpClient, oauth2Conf.BaseURL)
	return jiraClient, httpClient, err
}

// Creates a "bot" client with a JWT
func (ci *cloudInstance) getClientForBot() (*jira.Client, error) {
	conf := ci.getConfig()
	jwtConf := &ajwt.Config{
		Key:          ci.AtlassianSecurityContext.Key,
		ClientKey:    ci.AtlassianSecurityContext.ClientKey,
		SharedSecret: ci.AtlassianSecurityContext.SharedSecret,
		BaseURL:      ci.AtlassianSecurityContext.BaseURL,
	}

	httpClient := jwtConf.Client()
	httpClient = utils.WrapHTTPClient(httpClient,
		utils.WithRequestSizeLimit(conf.maxAttachmentSize),
		utils.WithResponseSizeLimit(conf.maxAttachmentSize))

	return jira.NewClient(httpClient, jwtConf.BaseURL)
}

func (ci *cloudInstance) parseHTTPRequestJWT(r *http.Request) (*jwt.Token, string, error) {
	err := r.ParseForm()
	if err != nil {
		return nil, "", errors.WithMessage(err, "failed to parse request")
	}
	tokenString := r.FormValue("jwt")
	if tokenString == "" {
		return nil, "", errors.New("no jwt in the request")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.Errorf("unsupported signing method: %v", token.Header["alg"])
		}
		// HMAC secret is a []byte
		return []byte(ci.AtlassianSecurityContext.SharedSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, "", errors.WithMessage(err, "failed to validate JWT")
	}

	return token, tokenString, nil
}
