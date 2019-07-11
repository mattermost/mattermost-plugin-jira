// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/model"
)

const (
	keyCurrentJIRAInstance = "current_jira_instance"
	keyKnownJIRAInstances  = "known_jira_instances"
	keyRSAKey              = "rsa_key"
	keyTokenSecret         = "token_secret"
	prefixJIRAInstance     = "jira_instance_"
	prefixOneTimeSecret    = "ots_" // + unique key that will be deleted after the first verification
)

type Store interface {
	CurrentInstanceStore
	InstanceStore
	UserStore
	SecretsStore
	OTSStore
}

type SecretsStore interface {
	EnsureAuthTokenEncryptSecret() ([]byte, error)
	EnsureRSAKey() (rsaKey *rsa.PrivateKey, returnErr error)
}

type InstanceStore interface {
	StoreJIRAInstance(ji Instance) error
	CreateInactiveCloudInstance(jiraURL string) error
	DeleteJiraInstance(key string) error
	LoadJIRAInstance(key string) (Instance, error)
	StoreKnownJIRAInstances(known map[string]string) error
	LoadKnownJIRAInstances() (map[string]string, error)
}

type CurrentInstanceStore interface {
	StoreCurrentJIRAInstance(ji Instance) error
	LoadCurrentJIRAInstance() (Instance, error)
}

type UserStore interface {
	StoreUserInfo(ji Instance, mattermostUserId string, jiraUser JIRAUser) error
	LoadJIRAUser(ji Instance, mattermostUserId string) (JIRAUser, error)
	LoadMattermostUserId(ji Instance, jiraUserName string) (string, error)
	DeleteUserInfo(ji Instance, mattermostUserId string) error
}

type OTSStore interface {
	StoreOneTimeSecret(token, secret string) error
	LoadOneTimeSecret(token string) (string, error)
	StoreOauth1aTemporaryCredentials(mmUserId string, credentials *OAuth1aTemporaryCredentials) error
	OneTimeLoadOauth1aTemporaryCredentials(mmUserId string) (*OAuth1aTemporaryCredentials, error)
}

type store struct {
	plugin *Plugin
}

func NewStore(p *Plugin) Store {
	return &store{plugin: p}
}

func keyWithInstance(ji Instance, key string) string {
	if prefixForInstance {
		h := md5.New()
		fmt.Fprintf(h, "%s/%s", ji.GetURL(), key)
		key = fmt.Sprintf("%x", h.Sum(nil))
	}
	return key
}

func hashkey(prefix, key string) string {
	h := md5.New()
	_, _ = h.Write([]byte(key))
	return fmt.Sprintf("%s%x", prefix, h.Sum(nil))
}

func (store store) get(key string, v interface{}) (returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr, "failed to get from store")
	}()

	data, appErr := store.plugin.API.KVGet(key)
	if appErr != nil {
		return appErr
	}

	if data == nil {
		return nil
	}

	err := json.Unmarshal(data, v)
	if err != nil {
		return err
	}

	return nil
}

func (store store) set(key string, v interface{}) (returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr, "failed to store")
	}()

	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	appErr := store.plugin.API.KVSet(key, data)
	if appErr != nil {
		return appErr
	}
	return nil
}

func (store store) StoreJIRAInstance(ji Instance) (returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr,
			fmt.Sprintf("failed to store Jira instance:%s", ji.GetURL()))
	}()

	err := store.set(hashkey(prefixJIRAInstance, ji.GetURL()), ji)
	if err != nil {
		return err
	}
	store.plugin.debugf("Stored: JIRA instance: %s", ji.GetURL())

	// Update known instances
	known, err := store.LoadKnownJIRAInstances()
	if err != nil {
		return err
	}
	known[ji.GetURL()] = ji.GetType()
	err = store.StoreKnownJIRAInstances(known)
	if err != nil {
		return err
	}
	store.plugin.debugf("Stored: known Jira instances: %+v", known)
	return nil
}

func (store store) CreateInactiveCloudInstance(jiraURL string) (returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessagef(returnErr,
			"failed to store new Jira Cloud instance:%s", jiraURL)
	}()

	ji := NewJIRACloudInstance(store.plugin, jiraURL, false,
		fmt.Sprintf(`{"BaseURL": "%s"}`, jiraURL),
		&AtlassianSecurityContext{BaseURL: jiraURL})

	data, err := json.Marshal(ji)
	if err != nil {
		return err
	}

	// Expire in 15 minutes
	appErr := store.plugin.API.KVSetWithExpiry(hashkey(prefixJIRAInstance,
		ji.GetURL()), data, 15*60)
	if appErr != nil {
		return appErr
	}
	store.plugin.debugf("Stored: new Jira Cloud instance: %s", ji.GetURL())
	return nil
}

func (store store) StoreCurrentJIRAInstance(ji Instance) (returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr,
			fmt.Sprintf("failed to store current Jira instance:%s", ji.GetURL()))
	}()
	err := store.set(keyCurrentJIRAInstance, ji)
	if err != nil {
		return err
	}
	store.plugin.updateConfig(func(conf *config) {
		conf.currentInstance = ji
		conf.currentInstanceExpires = time.Now().Add(currentInstanceTTL)
	})
	store.plugin.debugf("Stored: current Jira instance: %s", ji.GetURL())

	// Notify users we have installed an instance
	store.plugin.API.PublishWebSocketEvent(
		wSEventInstanceStatus,
		map[string]interface{}{
			"instance_installed": true,
		},
		&model.WebsocketBroadcast{},
	)

	return nil
}

func (store store) DeleteJiraInstance(key string) (returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr,
			fmt.Sprintf("failed to delete Jira instance:%v", key))
	}()

	// Delete the instance.
	appErr := store.plugin.API.KVDelete(hashkey(prefixJIRAInstance, key))
	if appErr != nil {
		return appErr
	}
	store.plugin.debugf("Deleted: Jira instance: %s", key)

	// Update known instances
	known, err := store.LoadKnownJIRAInstances()
	if err != nil {
		return err
	}
	for k := range known {
		if k == key {
			delete(known, k)
			break
		}
	}
	err = store.StoreKnownJIRAInstances(known)
	if err != nil {
		return err
	}
	store.plugin.debugf("Deleted: from known Jira instances: %s", key)

	// Remove the current instance if it matches the deleted
	current, err := store.LoadCurrentJIRAInstance()
	if err != nil {
		return err
	}
	if current.GetURL() == key {
		appErr := store.plugin.API.KVDelete(keyCurrentJIRAInstance)
		if appErr != nil {
			return appErr
		}
		store.plugin.updateConfig(func(conf *config) {
			// Reset, will get re-initialized as needed
			conf.currentInstance = nil
			conf.currentInstanceExpires = time.Time{}
		})
		store.plugin.debugf("Deleted: current Jira instance")
	}

	return nil
}

func (store store) LoadCurrentJIRAInstance() (Instance, error) {
	conf := store.plugin.getConfig()
	now := time.Now()

	if now.Before(conf.currentInstanceExpires) {
		// if conf.currentInstanceExpires is set and there is no current
		// instance, it's a cached "Not found"
		if conf.currentInstance == nil {
			return nil, errors.New("failed to load current Jira instance: not found")
		}
		return conf.currentInstance, nil
	}

	ji, err := store.loadJIRAInstance(keyCurrentJIRAInstance)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to load current Jira instance")
	}
	store.plugin.updateConfig(func(conf *config) {
		conf.currentInstance = ji
		conf.currentInstanceExpires = now.Add(currentInstanceTTL)
	})

	return ji, nil
}

func (store store) LoadJIRAInstance(key string) (Instance, error) {
	ji, err := store.loadJIRAInstance(hashkey(prefixJIRAInstance, key))
	if err != nil {
		return nil, errors.WithMessage(err, "failed to load Jira instance "+key)
	}

	return ji, nil
}

func (store store) loadJIRAInstance(fullkey string) (Instance, error) {
	data, appErr := store.plugin.API.KVGet(fullkey)
	if appErr != nil {
		return nil, appErr
	}
	if data == nil {
		return nil, errors.New("not found: " + fullkey)
	}

	// Unmarshal into any of the types just so that we can get the common data
	jsi := jiraServerInstance{}
	err := json.Unmarshal(data, &jsi)
	if err != nil {
		return nil, err
	}

	switch jsi.Type {
	case JIRATypeCloud:
		jci := jiraCloudInstance{}
		err = json.Unmarshal(data, &jci)
		if err != nil {
			return nil, errors.WithMessage(err, "failed to unmarshal stored Instance "+fullkey)
		}
		if len(jci.RawAtlassianSecurityContext) > 0 {
			err = json.Unmarshal([]byte(jci.RawAtlassianSecurityContext), &jci.AtlassianSecurityContext)
			if err != nil {
				return nil, errors.WithMessage(err, "failed to unmarshal stored Instance "+fullkey)
			}
		}
		jci.Init(store.plugin)
		return &jci, nil

	case JIRATypeServer:
		jsi.Init(store.plugin)
		return &jsi, nil
	}

	return nil, errors.New(fmt.Sprintf("Jira instance %s has unsupported type: %s", fullkey, jsi.Type))
}

func (store store) StoreKnownJIRAInstances(known map[string]string) (returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr,
			fmt.Sprintf("failed to store known Jira instances %+v", known))
	}()

	return store.set(keyKnownJIRAInstances, known)
}

func (store store) LoadKnownJIRAInstances() (map[string]string, error) {
	known := map[string]string{}
	err := store.get(keyKnownJIRAInstances, &known)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to load known Jira instances")
	}
	return known, nil
}

func (store store) StoreUserInfo(ji Instance, mattermostUserId string, jiraUser JIRAUser) (returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr,
			fmt.Sprintf("failed to store user, mattermostUserId:%s, Jira user:%s", mattermostUserId, jiraUser.Name))
	}()

	err := store.set(keyWithInstance(ji, mattermostUserId), jiraUser)
	if err != nil {
		return err
	}

	err = store.set(keyWithInstance(ji, jiraUser.Name), mattermostUserId)
	if err != nil {
		return err
	}

	// Also store AccountID -> mattermostUserID because Jira Cloud is deprecating the name field
	// https://developer.atlassian.com/cloud/jira/platform/api-changes-for-user-privacy-announcement/
	err = store.set(keyWithInstance(ji, jiraUser.AccountID), mattermostUserId)
	if err != nil {
		return err
	}

	store.plugin.debugf("Stored: Jira user, keys:\n\t%s (%s): %+v\n\t%s (%s): %s",
		keyWithInstance(ji, mattermostUserId), mattermostUserId, jiraUser,
		keyWithInstance(ji, jiraUser.Name), jiraUser.Name, mattermostUserId)

	return nil
}

var ErrUserNotFound = errors.New("user not found")

func (store store) LoadJIRAUser(ji Instance, mattermostUserId string) (JIRAUser, error) {
	jiraUser := JIRAUser{}
	err := store.get(keyWithInstance(ji, mattermostUserId), &jiraUser)
	if err != nil {
		return JIRAUser{}, errors.WithMessage(err,
			fmt.Sprintf("failed to load Jira user for mattermostUserId:%s", mattermostUserId))
	}
	if len(jiraUser.Key) == 0 {
		return JIRAUser{}, ErrUserNotFound
	}
	return jiraUser, nil
}

func (store store) LoadMattermostUserId(ji Instance, jiraUserNameOrID string) (string, error) {
	mattermostUserId := ""
	err := store.get(keyWithInstance(ji, jiraUserNameOrID), &mattermostUserId)
	if err != nil {
		return "", errors.WithMessage(err,
			"failed to load Mattermost user ID for Jira user/ID: "+jiraUserNameOrID)
	}
	if len(mattermostUserId) == 0 {
		return "", ErrUserNotFound
	}
	return mattermostUserId, nil
}

func (store store) DeleteUserInfo(ji Instance, mattermostUserId string) (returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr,
			fmt.Sprintf("failed to delete user, mattermostUserId:%s", mattermostUserId))
	}()

	jiraUser, err := store.LoadJIRAUser(ji, mattermostUserId)
	if err != nil {
		return err
	}

	appErr := store.plugin.API.KVDelete(keyWithInstance(ji, mattermostUserId))
	if appErr != nil {
		return appErr
	}

	appErr = store.plugin.API.KVDelete(keyWithInstance(ji, jiraUser.Name))
	if appErr != nil {
		return appErr
	}

	store.plugin.debugf("Deleted: user, keys: %s(%s), %s(%s)",
		mattermostUserId, keyWithInstance(ji, mattermostUserId),
		jiraUser.Name, keyWithInstance(ji, jiraUser.Name))
	return nil
}

func (store store) EnsureAuthTokenEncryptSecret() (secret []byte, returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr, "failed to ensure auth token secret")
	}()

	// nil, nil == NOT_FOUND, if we don't already have a key, try to generate one.
	secret, appErr := store.plugin.API.KVGet(keyTokenSecret)
	if appErr != nil {
		return nil, appErr
	}

	if len(secret) == 0 {
		newSecret := make([]byte, 32)
		_, err := rand.Reader.Read(newSecret)
		if err != nil {
			return nil, err
		}

		appErr = store.plugin.API.KVSet(keyTokenSecret, newSecret)
		if appErr != nil {
			return nil, appErr
		}
		secret = newSecret
		store.plugin.debugf("Stored: auth token secret")
	}

	// If we weren't able to save a new key above, another server must have beat us to it. Get the
	// key from the database, and if that fails, error out.
	if secret == nil {
		secret, appErr = store.plugin.API.KVGet(keyTokenSecret)
		if appErr != nil {
			return nil, appErr
		}
	}

	return secret, nil
}

func (store store) EnsureRSAKey() (rsaKey *rsa.PrivateKey, returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr, "failed to ensure RSA key")
	}()

	appErr := store.get(keyRSAKey, &rsaKey)
	if appErr != nil {
		return nil, appErr
	}

	if rsaKey == nil {
		newRSAKey, err := rsa.GenerateKey(rand.Reader, 1024)
		if err != nil {
			return nil, err
		}

		appErr = store.set(keyRSAKey, newRSAKey)
		if appErr != nil {
			return nil, appErr
		}
		rsaKey = newRSAKey
		store.plugin.debugf("Stored: RSA key")
	}

	// If we weren't able to save a new key above, another server must have beat us to it. Get the
	// key from the database, and if that fails, error out.
	if rsaKey == nil {
		appErr = store.get(keyRSAKey, &rsaKey)
		if appErr != nil {
			return nil, appErr
		}
	}

	return rsaKey, nil
}

func (store store) StoreOneTimeSecret(token, secret string) error {
	// Expire in 15 minutes
	appErr := store.plugin.API.KVSetWithExpiry(
		hashkey(prefixOneTimeSecret, token), []byte(secret), 15*60)
	if appErr != nil {
		return errors.WithMessage(appErr, "failed to store one-ttime secret "+token)
	}
	return nil
}

func (store store) LoadOneTimeSecret(key string) (string, error) {
	b, appErr := store.plugin.API.KVGet(hashkey(prefixOneTimeSecret, key))
	if appErr != nil {
		return "", errors.WithMessage(appErr, "failed to load one-time secret "+key)
	}

	appErr = store.plugin.API.KVDelete(hashkey(prefixOneTimeSecret, key))
	if appErr != nil {
		return "", errors.WithMessage(appErr, "failed to delete one-time secret "+key)
	}
	return string(b), nil
}

func (store store) StoreOauth1aTemporaryCredentials(mmUserId string, credentials *OAuth1aTemporaryCredentials) error {
	data, err := json.Marshal(&credentials)
	if err != nil {
		return err
	}
	// Expire in 15 minutes
	appErr := store.plugin.API.KVSetWithExpiry(hashkey(prefixOneTimeSecret, mmUserId), data, 15*60)
	if appErr != nil {
		return errors.WithMessage(appErr, "failed to store oauth temporary credentials for "+mmUserId)
	}
	return nil
}

func (store store) OneTimeLoadOauth1aTemporaryCredentials(mmUserId string) (*OAuth1aTemporaryCredentials, error) {
	b, appErr := store.plugin.API.KVGet(hashkey(prefixOneTimeSecret, mmUserId))
	if appErr != nil {
		return nil, errors.WithMessage(appErr, "failed to load temporary credentials for "+mmUserId)
	}
	var credentials OAuth1aTemporaryCredentials
	err := json.Unmarshal(b, &credentials)
	if err != nil {
		return nil, err
	}
	appErr = store.plugin.API.KVDelete(hashkey(prefixOneTimeSecret, mmUserId))
	if appErr != nil {
		return nil, errors.WithMessage(appErr, "failed to delete temporary credentials for "+mmUserId)
	}
	return &credentials, nil
}
