// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package store

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
)

const (
	keyCurrentInstance  = "current_jira_instance"
	keyKnownInstances   = "known_jira_instances"
	keyRSAKey           = "rsa_key"
	keyTokenSecret      = "token_secret"
	prefixInstance      = "jira_instance_"
	prefixOneTimeSecret = "ots_" // + unique key that will be deleted after the first verification
)

type Store interface {
	CurrentInstanceStore
	InstanceStore
	UserStore
	SecretStore
}

type SecretStore interface {
	EnsureAuthTokenEncryptSecret() ([]byte, error)
	EnsureRSAKey() (rsaKey *rsa.PrivateKey, returnErr error)
	StoreOneTimeSecret(token, secret string) error
	LoadOneTimeSecret(token string) (string, error)
	StoreOauth1aTemporaryCredentials(mmUserId string, credentials *OAuth1aTemporaryCredentials) error
	OneTimeLoadOauth1aTemporaryCredentials(mmUserId string) (*OAuth1aTemporaryCredentials, error)
}

type InstanceStore interface {
	StoreInstance(Instance) error
	CreateInactiveCloudInstance(jiraURL string) error
	DeleteJiraInstance(key string) error
	LoadInstance(key string) (Instance, error)
	StoreKnownInstances(known map[string]string) error
	LoadKnownInstances() (map[string]string, error)
}

type CurrentInstanceStore interface {
	StoreCurrentInstance(Instance) error
	LoadCurrentInstance() (Instance, error)
}

type UserStore interface {
	StoreUserInfo(Instance, string, JiraUser) error
	LoadJiraUser(Instance, string) (JiraUser, error)
	LoadMattermostUserId(Instance, string) (string, error)
	DeleteUserInfo(Instance, string) error
}

type store struct {
	plugin *Plugin
}

func NewStore(p *Plugin) Store {
	return &store{plugin: p}
}

func keyWithInstance(instance Instance, key string) string {
	if prefixForInstance {
		h := md5.New()
		fmt.Fprintf(h, "%s/%s", instance.GetURL(), key)
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

func (store store) StoreInstance(instance Instance) (returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr,
			fmt.Sprintf("failed to store Jira instance:%s", instance.GetURL()))
	}()

	err := store.set(hashkey(prefixInstance, instance.GetURL()), instance)
	if err != nil {
		return err
	}

	// Update known instances
	known, err := store.LoadKnownInstances()
	if err != nil {
		return err
	}
	known[instance.GetURL()] = instance.GetType()
	err = store.StoreKnownInstances(known)
	if err != nil {
		return err
	}
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

	instance := NewCloudInstance(jiraURL, false,
		fmt.Sprintf(`{"BaseURL": %s}`, jiraURL),
		&AtlassianSecurityContext{BaseURL: jiraURL})

	data, err := json.Marshal(instance)
	if err != nil {
		return err
	}

	// Expire in 15 minutes
	appErr := store.plugin.API.KVSetWithExpiry(hashkey(prefixInstance,
		instance.GetURL()), data, 15*60)
	if appErr != nil {
		return appErr
	}
	return nil
}

func (store store) StoreCurrentInstance(instance Instance) (returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr,
			fmt.Sprintf("failed to store current Jira instance:%s", instance.GetURL()))
	}()
	err := store.set(keyCurrentInstance, instance)
	if err != nil {
		return err
	}
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
	appErr := store.plugin.API.KVDelete(hashkey(prefixInstance, key))
	if appErr != nil {
		return appErr
	}

	// Update known instances
	known, err := store.LoadKnownInstances()
	if err != nil {
		return err
	}
	for k := range known {
		if k == key {
			delete(known, k)
			break
		}
	}
	err = store.StoreKnownInstances(known)
	if err != nil {
		return err
	}

	// Remove the current instance if it matches the deleted
	current, err := store.LoadCurrentInstance()
	if err != nil {
		return err
	}
	if current.GetURL() == key {
		appErr := store.plugin.API.KVDelete(keyCurrentInstance)
		if appErr != nil {
			return appErr
		}
	}

	return nil
}

func (store store) LoadCurrentInstance() (Instance, error) {
	instance, err := store.loadInstance(keyCurrentInstance)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to load current Jira instance")
	}

	return instance, nil
}

func (store store) LoadInstance(key string) (Instance, error) {
	instance, err := store.loadInstance(hashkey(prefixInstance, key))
	if err != nil {
		return nil, errors.WithMessage(err, "failed to load Jira instance "+key)
	}

	return instance, nil
}

func (store store) loadInstance(fullkey string) (Instance, error) {
	data, appErr := store.plugin.API.KVGet(fullkey)
	if appErr != nil {
		return nil, appErr
	}
	if data == nil {
		return nil, errors.New("not found: " + fullkey)
	}

	// Unmarshal into any of the types just so that we can get the common data
	serverInstance := jiraServerInstance{}
	err := json.Unmarshal(data, &serverInstance)
	if err != nil {
		return nil, err
	}

	switch serverInstance.Type {
	case InstanceTypeServer:
		return &serverInstance, nil

	case InstanceTypeCloud:
		jci := jiraCloudInstance{}
		err = json.Unmarshal(data, &jci)
		if err != nil {
			return nil, errors.WithMessage(err, "failed to unmarshal stored Instance "+fullkey)
		}
		return &jci, nil
	}

	return nil, errors.New(fmt.Sprintf("Jira instance %s has unsupported type: %s", fullkey, serverInstance.Type))
}

func (store store) StoreKnownInstances(known map[string]string) (returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr,
			fmt.Sprintf("failed to store known Jira instances %+v", known))
	}()

	return store.set(keyKnownInstances, known)
}

func (store store) LoadKnownInstances() (map[string]string, error) {
	known := map[string]string{}
	err := store.get(keyKnownInstances, &known)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to load known Jira instances")
	}
	return known, nil
}

func (store store) StoreUserInfo(instance Instance, mattermostUserId string, jiraUser JiraUser) (returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr,
			fmt.Sprintf("failed to store user, mattermostUserId:%s, Jira user:%s", mattermostUserId, jiraUser.Name))
	}()

	err := store.set(keyWithInstance(instance, mattermostUserId), jiraUser)
	if err != nil {
		return err
	}

	err = store.set(keyWithInstance(instance, jiraUser.Name), mattermostUserId)
	if err != nil {
		return err
	}

	return nil
}

var ErrUserNotFound = errors.New("user not found")

func (store store) LoadJiraUser(instance Instance, mattermostUserId string) (JiraUser, error) {
	jiraUser := JiraUser{}
	err := store.get(keyWithInstance(instance, mattermostUserId), &jiraUser)
	if err != nil {
		return JiraUser{}, errors.WithMessage(err,
			fmt.Sprintf("failed to load Jira user for user ID: %q", mattermostUserId))
	}
	if len(jiraUser.Key) == 0 {
		return JiraUser{}, ErrUserNotFound
	}
	return jiraUser, nil
}

func (store store) LoadMattermostUserId(instance Instance, jiraUserName string) (string, error) {
	mattermostUserId := ""
	err := store.get(keyWithInstance(instance, jiraUserName), &mattermostUserId)
	if err != nil {
		return "", errors.WithMessage(err,
			"failed to load Mattermost user ID for Jira user: "+jiraUserName)
	}
	if len(mattermostUserId) == 0 {
		return "", ErrUserNotFound
	}
	return mattermostUserId, nil
}

func (store store) DeleteUserInfo(instance Instance, mattermostUserId string) (returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr,
			fmt.Sprintf("failed to delete user, user ID: %q", mattermostUserId))
	}()

	jiraUser, err := store.LoadJiraUser(instance, mattermostUserId)
	if err != nil {
		return err
	}

	appErr := store.plugin.API.KVDelete(keyWithInstance(instance, mattermostUserId))
	if appErr != nil {
		return appErr
	}

	appErr = store.plugin.API.KVDelete(keyWithInstance(instance, jiraUser.Name))
	if appErr != nil {
		return appErr
	}

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
