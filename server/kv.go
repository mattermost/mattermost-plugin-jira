// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
)

const (
	keyCurrentJIRAInstance = "current_jira_instance"
	keyKnownJIRAInstances  = "known_jira_instances"
	keyRSAKey              = "rsa_key"
	keyTokenSecret         = "token_secret"
	prefixJIRAInstance     = "jira_instance_"
	prefixOneTimeSecret    = "ots_" // + unique key that will be deleted after the first verification
)

type KV interface {
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
	DeleteOneTimeSecret(token string) error
}

type kv struct {
	plugin *Plugin
}

func NewKV(p *Plugin) KV {
	return &kv{plugin: p}
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

func (kv kv) get(key string, v interface{}) (returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr, "failed to get from KV store")
	}()

	data, appErr := kv.plugin.API.KVGet(key)
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

func (kv kv) set(key string, v interface{}) (returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr, "kvSet")
	}()

	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	appErr := kv.plugin.API.KVSet(key, data)
	if appErr != nil {
		return appErr
	}
	return nil
}

func (kv kv) StoreJIRAInstance(ji Instance) (returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr,
			fmt.Sprintf("failed to store Jira instance:%s", ji.GetURL()))
	}()

	err := kv.set(hashkey(prefixJIRAInstance, ji.GetURL()), ji)
	if err != nil {
		return err
	}
	kv.plugin.debugf("Stored: JIRA instance: %s", ji.GetURL())

	// Update known instances
	known, err := kv.LoadKnownJIRAInstances()
	if err != nil {
		return err
	}
	known[ji.GetURL()] = ji.GetType()
	err = kv.StoreKnownJIRAInstances(known)
	if err != nil {
		return err
	}
	kv.plugin.debugf("Stored: known Jira instances: %+v", known)
	return nil
}

func (kv kv) CreateInactiveCloudInstance(jiraURL string) (returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessagef(returnErr,
			"failed to store new Jira Cloud instance:%s", jiraURL)
	}()

	ji := NewJIRACloudInstance(kv.plugin, jiraURL, false,
		fmt.Sprintf(`{"BaseURL": %s}`, jiraURL),
		&AtlassianSecurityContext{BaseURL: jiraURL})

	data, err := json.Marshal(ji)
	if err != nil {
		return err
	}

	// Expire in 15 minutes
	appErr := kv.plugin.API.KVSetWithExpiry(hashkey(prefixJIRAInstance,
		ji.GetURL()), data, 15*60)
	if appErr != nil {
		return appErr
	}
	kv.plugin.debugf("Stored: new Jira Cloud instance: %s", ji.GetURL())
	return nil
}

func (kv kv) StoreCurrentJIRAInstance(ji Instance) (returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr,
			fmt.Sprintf("failed to store current Jira instance:%s", ji.GetURL()))
	}()
	err := kv.set(keyCurrentJIRAInstance, ji)
	if err != nil {
		return err
	}
	kv.plugin.debugf("Stored: current Jira instance: %s", ji.GetURL())
	return nil
}

func (kv kv) DeleteJiraInstance(key string) (returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr,
			fmt.Sprintf("failed to delete Jira instance:%v", key))
	}()

	// Delete the instance.
	appErr := kv.plugin.API.KVDelete(hashkey(prefixJIRAInstance, key))
	if appErr != nil {
		return appErr
	}
	kv.plugin.debugf("Deleted: Jira instance: %s", key)

	// Update known instances
	known, err := kv.LoadKnownJIRAInstances()
	if err != nil {
		return err
	}
	for k := range known {
		if k == key {
			delete(known, k)
			break
		}
	}
	err = kv.StoreKnownJIRAInstances(known)
	if err != nil {
		return err
	}
	kv.plugin.debugf("Deleted: from known Jira instances: %s", key)

	// Remove the current instance if it matches the deleted
	current, err := kv.LoadCurrentJIRAInstance()
	if err != nil {
		return err
	}
	if current.GetURL() == key {
		appErr := kv.plugin.API.KVDelete(keyCurrentJIRAInstance)
		if appErr != nil {
			return appErr
		}
		kv.plugin.debugf("Deleted: current Jira instance")
	}

	return nil
}

func (kv kv) LoadCurrentJIRAInstance() (Instance, error) {
	ji, err := kv.loadJIRAInstance(keyCurrentJIRAInstance)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to load current Jira instance")
	}

	return ji, nil
}

func (kv kv) LoadJIRAInstance(key string) (Instance, error) {
	ji, err := kv.loadJIRAInstance(hashkey(prefixJIRAInstance, key))
	if err != nil {
		return nil, errors.WithMessage(err, "failed to load Jira instance "+key)
	}

	return ji, nil
}

func (kv kv) loadJIRAInstance(fullkey string) (Instance, error) {
	data, appErr := kv.plugin.API.KVGet(fullkey)
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
		jci.Init(kv.plugin)
		return &jci, nil

	case JIRATypeServer:
		jsi.Init(kv.plugin)
		return &jsi, nil
	}

	return nil, errors.New(fmt.Sprintf("Jira instance %s has unsupported type: %s", fullkey, jsi.Type))
}

func (kv kv) StoreKnownJIRAInstances(known map[string]string) (returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr,
			fmt.Sprintf("failed to store known Jira instances %+v", known))
	}()

	return kv.set(keyKnownJIRAInstances, known)
}

func (kv kv) LoadKnownJIRAInstances() (map[string]string, error) {
	known := map[string]string{}
	err := kv.get(keyKnownJIRAInstances, &known)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to load known Jira instances")
	}
	return known, nil
}

func (kv kv) StoreUserInfo(ji Instance, mattermostUserId string, jiraUser JIRAUser) (returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr,
			fmt.Sprintf("failed to store user, mattermostUserId:%s, Jira user:%s", mattermostUserId, jiraUser.Name))
	}()

	err := kv.set(keyWithInstance(ji, mattermostUserId), jiraUser)
	if err != nil {
		return err
	}

	err = kv.set(keyWithInstance(ji, jiraUser.Name), mattermostUserId)
	if err != nil {
		return err
	}

	kv.plugin.debugf("Stored: Jira user, keys:\n\t%s (%s): %+v\n\t%s (%s): %s",
		keyWithInstance(ji, mattermostUserId), mattermostUserId, jiraUser,
		keyWithInstance(ji, jiraUser.Name), jiraUser.Name, mattermostUserId)

	return nil
}

var ErrUserNotFound = errors.New("user not found")

func (kv kv) LoadJIRAUser(ji Instance, mattermostUserId string) (JIRAUser, error) {
	jiraUser := JIRAUser{}
	err := kv.get(keyWithInstance(ji, mattermostUserId), &jiraUser)
	if err != nil {
		return JIRAUser{}, errors.WithMessage(err,
			fmt.Sprintf("failed to load Jira user for mattermostUserId:%s", mattermostUserId))
	}
	if len(jiraUser.Key) == 0 {
		return JIRAUser{}, ErrUserNotFound
	}
	return jiraUser, nil
}

func (kv kv) LoadMattermostUserId(ji Instance, jiraUserName string) (string, error) {
	mattermostUserId := ""
	err := kv.get(keyWithInstance(ji, jiraUserName), &mattermostUserId)
	if err != nil {
		return "", errors.WithMessage(err,
			"failed to load Mattermost user ID for Jira user: "+jiraUserName)
	}
	if len(mattermostUserId) == 0 {
		return "", ErrUserNotFound
	}
	return mattermostUserId, nil
}

func (kv kv) DeleteUserInfo(ji Instance, mattermostUserId string) (returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr,
			fmt.Sprintf("failed to delete user, mattermostUserId:%s", mattermostUserId))
	}()

	jiraUser, err := kv.LoadJIRAUser(ji, mattermostUserId)
	if err != nil {
		return err
	}

	appErr := kv.plugin.API.KVDelete(keyWithInstance(ji, mattermostUserId))
	if appErr != nil {
		return appErr
	}

	appErr = kv.plugin.API.KVDelete(keyWithInstance(ji, jiraUser.Name))
	if appErr != nil {
		return appErr
	}

	kv.plugin.debugf("Deleted: user, keys: %s(%s), %s(%s)",
		mattermostUserId, keyWithInstance(ji, mattermostUserId),
		jiraUser.Name, keyWithInstance(ji, jiraUser.Name))
	return nil
}

func (kv kv) EnsureAuthTokenEncryptSecret() (secret []byte, returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr, "failed to ensure auth token secret")
	}()

	// nil, nil == NOT_FOUND, if we don't already have a key, try to generate one.
	secret, appErr := kv.plugin.API.KVGet(keyTokenSecret)
	if appErr != nil {
		return nil, appErr
	}

	if len(secret) == 0 {
		newSecret := make([]byte, 32)
		_, err := rand.Reader.Read(newSecret)
		if err != nil {
			return nil, err
		}

		appErr = kv.plugin.API.KVSet(keyTokenSecret, newSecret)
		if appErr != nil {
			return nil, appErr
		}
		secret = newSecret
		kv.plugin.debugf("Stored: auth token secret")
	}

	// If we weren't able to save a new key above, another server must have beat us to it. Get the
	// key from the database, and if that fails, error out.
	if secret == nil {
		secret, appErr = kv.plugin.API.KVGet(keyTokenSecret)
		if appErr != nil {
			return nil, appErr
		}
	}

	return secret, nil
}

func (kv kv) EnsureRSAKey() (rsaKey *rsa.PrivateKey, returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr, "failed to ensure RSA key")
	}()

	appErr := kv.get(keyRSAKey, &rsaKey)
	if appErr != nil {
		return nil, appErr
	}

	if rsaKey == nil {
		newRSAKey, err := rsa.GenerateKey(rand.Reader, 1024)
		if err != nil {
			return nil, err
		}

		appErr = kv.set(keyRSAKey, newRSAKey)
		if appErr != nil {
			return nil, appErr
		}
		rsaKey = newRSAKey
		kv.plugin.debugf("Stored: RSA key")
	}

	// If we weren't able to save a new key above, another server must have beat us to it. Get the
	// key from the database, and if that fails, error out.
	if rsaKey == nil {
		appErr = kv.get(keyRSAKey, &rsaKey)
		if appErr != nil {
			return nil, appErr
		}
	}

	return rsaKey, nil
}

func (kv kv) StoreOneTimeSecret(token, secret string) error {
	// Expire in 15 minutes
	appErr := kv.plugin.API.KVSetWithExpiry(
		hashkey(prefixOneTimeSecret, token), []byte(secret), 15*60)
	if appErr != nil {
		return errors.WithMessage(appErr, "failed to store one-ttime secret "+token)
	}
	return nil
}

func (kv kv) LoadOneTimeSecret(token string) (string, error) {
	b, appErr := kv.plugin.API.KVGet(hashkey(prefixOneTimeSecret, token))
	if appErr != nil {
		return "", errors.WithMessage(appErr, "failed to load one-time secret "+token)
	}
	return string(b), nil
}

func (kv kv) DeleteOneTimeSecret(token string) error {
	appErr := kv.plugin.API.KVDelete(hashkey(prefixOneTimeSecret, token))
	if appErr != nil {
		return errors.WithMessage(appErr, "failed to delete one-time secret "+token)
	}
	return nil
}
