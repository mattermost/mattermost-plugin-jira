// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/pkg/errors"
)

const (
	// Key to migrate the V2 installed instance
	keyRSAKey           = "rsa_key"
	keyTokenSecret      = "token_secret"
	prefixOneTimeSecret = "ots_" // + unique key that will be deleted after the first verification
	prefixStats         = "stats_"
	prefixUser          = "user_"
)

type Store interface {
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
	CreateInactiveCloudInstance(jiraURL string) error
	DeleteInstance(key string) error
	LoadInstance(key string) (Instance, error)
	LoadInstances() (Instances, error)
	StoreInstance(instance Instance) error
	StoreInstances(Instances) error
}

type UserStore interface {
	LoadUser(mattermostUserId string) (*User, error)
	StoreUser(*User) error
	StoreConnection(instance Instance, mattermostUserId string, c *Connection) error
	LoadConnection(instance Instance, mattermostUserId string) (*Connection, error)
	LoadMattermostUserId(instance Instance, jiraUsername string) (string, error)
	DeleteConnection(instance Instance, mattermostUserId string) error
	CountUsers() (int, error)
}

type OTSStore interface {
	StoreOneTimeSecret(token, secret string) error
	LoadOneTimeSecret(token string) (string, error)
	StoreOauth1aTemporaryCredentials(mmUserId string, credentials *OAuth1aTemporaryCredentials) error
	OneTimeLoadOauth1aTemporaryCredentials(mmUserId string) (*OAuth1aTemporaryCredentials, error)
}

// Number of items to retrieve in KVList operations, made a variable so
// that tests can manipulate
var listPerPage = 100

type store struct {
	plugin *Plugin
}

func NewStore(p *Plugin) *store {
	return &store{plugin: p}
}

func keyWithInstance(instance Instance, key string) string {
	h := md5.New()
	fmt.Fprintf(h, "%s/%s", instance.GetURL(), key)
	key = fmt.Sprintf("%x", h.Sum(nil))
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

func (store store) StoreConnection(instance Instance, mattermostUserId string, c *Connection) (returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr,
			fmt.Sprintf("failed to store connection, mattermostUserId:%s, Jira user:%s", mattermostUserId, c.DisplayName))
	}()

	err := store.set(keyWithInstance(instance, mattermostUserId), c)
	if err != nil {
		return err
	}

	err = store.set(keyWithInstance(instance, c.JiraAccountID()), mattermostUserId)
	if err != nil {
		return err
	}

	// Also store AccountID -> mattermostUserID because Jira Cloud is deprecating the name field
	// https://developer.atlassian.com/cloud/jira/platform/api-changes-for-user-privacy-announcement/
	err = store.set(keyWithInstance(instance, c.JiraAccountID()), mattermostUserId)
	if err != nil {
		return err
	}

	store.plugin.debugf("Stored: connection, keys:\n\t%s (%s): %+v\n\t%s (%s): %s",
		keyWithInstance(instance, mattermostUserId), mattermostUserId, c,
		keyWithInstance(instance, c.JiraAccountID()), c.JiraAccountID(), mattermostUserId)

	return nil
}

var ErrConnectionNotFound = errors.New("connection not found")

func (store store) LoadConnection(instance Instance, mattermostUserId string) (*Connection, error) {
	c := &Connection{}
	err := store.get(keyWithInstance(instance, mattermostUserId), c)
	if err != nil {
		return nil, errors.WithMessage(err,
			fmt.Sprintf("failed to load connection for mattermostUserId:%s", mattermostUserId))
	}
	if len(c.JiraAccountID()) == 0 {
		return nil, ErrUserNotFound
	}
	c.PluginVersion = manifest.Version
	return c, nil
}

func (store store) LoadMattermostUserId(instance Instance, jiraUserNameOrID string) (string, error) {
	mattermostUserId := ""
	err := store.get(keyWithInstance(instance, jiraUserNameOrID), &mattermostUserId)
	if err != nil {
		return "", errors.WithMessage(err,
			"failed to load Mattermost user ID for Jira user/ID: "+jiraUserNameOrID)
	}
	if len(mattermostUserId) == 0 {
		return "", ErrUserNotFound
	}
	return mattermostUserId, nil
}

func (store store) DeleteConnection(instance Instance, mattermostUserId string) (returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr,
			fmt.Sprintf("failed to delete user, mattermostUserId:%s", mattermostUserId))
	}()

	c, err := store.LoadConnection(instance, mattermostUserId)
	if err != nil {
		return err
	}

	appErr := store.plugin.API.KVDelete(keyWithInstance(instance, mattermostUserId))
	if appErr != nil {
		return appErr
	}

	appErr = store.plugin.API.KVDelete(keyWithInstance(instance, c.JiraAccountID()))
	if appErr != nil {
		return appErr
	}

	store.plugin.debugf("Deleted: user, keys: %s(%s), %s(%s)",
		mattermostUserId, keyWithInstance(instance, mattermostUserId),
		c.JiraAccountID(), keyWithInstance(instance, c.JiraAccountID()))
	return nil
}

func (store store) StoreUser(user *User) (returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr,
			fmt.Sprintf("failed to store user, mattermostUserId:%s", user.MattermostUserID))
	}()

	key := hashkey(prefixUser, user.MattermostUserID)
	err := store.set(key, user)
	if err != nil {
		return err
	}

	store.plugin.debugf("Stored: user key:%s: %+v", key, user)
	return nil
}

var ErrUserNotFound = errors.New("user not found")

func (store store) LoadUser(mattermostUserId string) (*User, error) {
	user := &User{}
	key := hashkey(prefixUser, user.MattermostUserID)
	err := store.get(key, user)
	if err != nil {
		return nil, errors.WithMessage(err,
			fmt.Sprintf("failed to load Jira user for mattermostUserId:%s", mattermostUserId))
	}
	return user, nil
}

var reHexKeyFormat = regexp.MustCompile("^[[:xdigit:]]{32}$")

func (store store) CountUsers() (int, error) {
	count := 0
	for i := 0; ; i++ {
		keys, appErr := store.plugin.API.KVList(i, listPerPage)
		if appErr != nil {
			return 0, appErr
		}

		for _, key := range keys {
			// User records are not currently prefixed. Consider any 32-hex key.
			if !reHexKeyFormat.MatchString(key) {
				continue
			}

			var data []byte
			data, appErr = store.plugin.API.KVGet(key)
			if appErr != nil {
				return 0, appErr
			}
			v := map[string]interface{}{}
			err := json.Unmarshal(data, &v)
			if err != nil {
				// Skip non-JSON values.
				continue
			}

			// A valid user record?
			if v["Settings"] != nil && (v["accountId"] != nil || v["name"] != nil && v["key"] != nil) {
				count++
			}
		}

		if len(keys) < listPerPage {
			break
		}
	}
	return count, nil
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
