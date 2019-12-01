package main

import (
	"fmt"
	"testing"

	jira "github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

const (
	nonExistantIssueKey   = "FAKE-1"
	noPermissionsIssueKey = "SUDO-1"
	existingIssueKey      = "REAL-1"
	noIssueFoundError     = "We couldn't find the issue key. Please confirm the issue key and try again. You may not have permissions to access this issue."
	noPermissionsError    = "You do not have the appropriate permissions to perform this action. Please contact your Jira administrator."
)

type testClient struct {
	RESTService
	UserService
	ProjectService
	SearchService
	IssueService
}

func (client testClient) GetTransitions(issueKey string) ([]jira.Transition, error) {
	if issueKey == nonExistantIssueKey {
		return []jira.Transition{}, errors.New(noIssueFoundError)
	} else if issueKey == noPermissionsIssueKey {
		return []jira.Transition{}, nil
	}

	return []jira.Transition{
		jira.Transition{To: jira.Status{Name: "To Do"}},
		jira.Transition{To: jira.Status{Name: "In Progress"}},
		jira.Transition{To: jira.Status{Name: "In Testing"}},
	}, nil
}

func (client testClient) DoTransition(issueKey string, transitionID string) error {
	return nil
}

func TestTransitionJiraIssue(t *testing.T) {
	p := Plugin{currentInstanceStore: mockCurrentInstanceStore{}}
	tests := []struct {
		name        string
		issueKey    string
		toState     string
		expectedMsg string
		expectedErr error
	}{
		{"Transitioning a non existant issue", nonExistantIssueKey, "To Do", "", errors.Errorf(noIssueFoundError)},
		{"Transitioning an issue where user does not have access", noPermissionsIssueKey, "To Do", "", errors.Errorf(noPermissionsError)},
		{"Looking for an invalid state", existingIssueKey, "tofu", "", errors.Errorf("\"tofu\" is not a valid state. Please use one of: \"To Do, In Progress, In Testing\"")},
		{"Matching multiple available states", existingIssueKey, "in", "", errors.Errorf("please be more specific, \"in\" matched several states: \"In Progress, In Testing\"")},
		{"Successfully transitioning to new state", existingIssueKey, "inprog", fmt.Sprintf("[%s](%s/browse/%s) transitioned to `In Progress`", existingIssueKey, mockCurrentInstanceURL, existingIssueKey), nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := p.transitionJiraIssue("user", tt.issueKey, tt.toState)
			assert.Equal(t, tt.expectedMsg, actual)
			if tt.expectedErr != nil {
				assert.Error(t, tt.expectedErr, err)
			}
		})
	}
}
