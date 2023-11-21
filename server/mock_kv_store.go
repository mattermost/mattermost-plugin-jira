package main

import (
	"encoding/json"
	"fmt"

	"github.com/mattermost/mattermost-server/v6/plugin/plugintest"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

type testKVStore map[string][]byte

func makeTestKVStore(api *plugintest.API, initialValues testKVStore) testKVStore {
	testStore := initialValues
	if testStore == nil {
		testStore = testKVStore{}
	}

	kvCall := api.On("KVGet", mock.Anything).Maybe().Return(nil, nil)
	kvCall.Run(func(args mock.Arguments) {
		key := args.Get(0).(string)
		val, ok := testStore[key]
		if ok {
			kvCall.Return(val, nil)
		} else {
			kvCall.Return(nil, nil)
		}
	})

	api.On("KVSet", mock.Anything, mock.Anything).Maybe().Return(nil).Run(func(args mock.Arguments) {
		key := args.Get(0).(string)
		value := args.Get(1).([]byte)
		testStore[key] = value
	})

	return testStore
}

type mockInstanceStoreForOauthMigration struct {
	plugin *Plugin
}

func (store mockInstanceStoreForOauthMigration) CreateInactiveCloudInstance(types.ID, string) error {
	return nil
}

func (store mockInstanceStoreForOauthMigration) DeleteInstance(types.ID) error {
	return nil
}

func (store mockInstanceStoreForOauthMigration) LoadInstance(instanceID types.ID) (Instance, error) {
	if instanceID == "" {
		return nil, errors.Wrap(kvstore.ErrNotFound, "no instance specified")
	}

	instance, err := store.LoadInstanceFullKey(hashkey(prefixInstance, instanceID.String()))
	if err != nil {
		return nil, errors.Wrap(err, instanceID.String())
	}
	return instance, nil
}

func (store mockInstanceStoreForOauthMigration) LoadInstanceFullKey(fullkey string) (Instance, error) {
	var data []byte
	data, err := store.plugin.API.KVGet(fullkey)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, errors.Wrap(kvstore.ErrNotFound, fullkey)
	}

	si := serverInstance{}
	if err := json.Unmarshal(data, &si); err != nil {
		return nil, err
	}

	switch si.Type {
	case CloudInstanceType:
		ci := cloudInstance{}
		if err := json.Unmarshal(data, &ci); err != nil {
			return nil, errors.WithMessage(err, fmt.Sprintf("failed to unmarshal stored instance %s", fullkey))
		}

		if len(ci.RawAtlassianSecurityContext) > 0 {
			if err := json.Unmarshal([]byte(ci.RawAtlassianSecurityContext), &ci.AtlassianSecurityContext); err != nil {
				return nil, errors.WithMessage(err, fmt.Sprintf("failed to unmarshal stored instance %s", fullkey))
			}
		}
		ci.Plugin = store.plugin
		return &ci, nil

	case CloudOAuthInstanceType:
		ci := cloudOAuthInstance{}
		if err := json.Unmarshal(data, &ci); err != nil {
			return nil, errors.WithMessage(err, fmt.Sprintf("failed to unmarshal stored instance %s", fullkey))
		}

		if ci.JWTInstance != nil && len(ci.JWTInstance.RawAtlassianSecurityContext) > 0 {
			if err := json.Unmarshal([]byte(ci.JWTInstance.RawAtlassianSecurityContext), &ci.JWTInstance.AtlassianSecurityContext); err != nil {
				return nil, errors.WithMessage(err, fmt.Sprintf("failed to unmarshal stored instance %s", fullkey))
			}
			ci.JWTInstance.Plugin = store.plugin
		}
		ci.Plugin = store.plugin
		return &ci, nil

	case ServerInstanceType:
		si.Plugin = store.plugin
		return &si, nil
	}
	return nil, errors.Errorf("Jira instance %s has unsupported type %s", fullkey, si.Type)
}

func (store mockInstanceStoreForOauthMigration) LoadInstances() (*Instances, error) {
	return NewInstances(), nil
}

func (store mockInstanceStoreForOauthMigration) StoreInstance(instance Instance) error {
	return nil
}

func (store mockInstanceStoreForOauthMigration) StoreInstances(*Instances) error {
	return nil
}

type mockChecker struct{}

func (store *mockChecker) HasEnterpriseFeatures() bool {
	return false
}
