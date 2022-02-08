package main

import "github.com/mattermost/mattermost-server/v6/plugin/plugintest/mock"

func mockAnythingOfTypeBatch(argType string, numCalls int) []interface{} {
	args := make([]interface{}, numCalls)

	for i := 0; i < numCalls; i++ {
		args[i] = mock.AnythingOfType(argType)
	}

	return args
}
