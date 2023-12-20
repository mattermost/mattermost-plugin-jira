package main

import (
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/mock"
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

	api.On("KVSetWithOptions", mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("model.PluginKVSetOptions")).Return(true, nil).Run(func(args mock.Arguments) {
		key := args.Get(0).(string)
		value := args.Get(1).([]byte)
		testStore[key] = value
	})

	return testStore
}
