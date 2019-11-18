package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUserSettings_String(t *testing.T) {
	tests := map[string]struct {
		settings       UserSettings
		expectedOutput string
	}{
		"notifications on": {
			settings:       UserSettings{Notifications: false},
			expectedOutput: "\tNotifications: off",
		},
		"notifications off": {
			settings:       UserSettings{Notifications: true},
			expectedOutput: "\tNotifications: on",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expectedOutput, tt.settings.String())
		})
	}
}
