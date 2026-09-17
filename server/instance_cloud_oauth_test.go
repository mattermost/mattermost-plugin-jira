// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

// TestSelectResourceID covers routing Jira API calls to the site the instance
// was installed for. A token can grant access to several sites, whose project
// key sequences are independent, so taking the wrong one creates issues in one
// site while every link the plugin renders points at another.
func TestSelectResourceID(t *testing.T) {
	const productionURL = "https://production.example.com"
	const sandboxURL = "https://sandbox.example.com"

	instanceFor := func(baseURL string) *cloudOAuthInstance {
		return &cloudOAuthInstance{
			InstanceCommon: &InstanceCommon{
				InstanceID: types.ID(baseURL),
				Type:       CloudOAuthInstanceType,
			},
			JiraBaseURL: baseURL,
		}
	}

	for name, tc := range map[string]struct {
		baseURL    string
		resources  JiraAccessibleResources
		expectedID string
		expectErr  string
	}{
		"the installed site is picked when the sandbox is listed first": {
			baseURL: productionURL,
			resources: JiraAccessibleResources{
				{ID: "sandbox-resource", URL: sandboxURL},
				{ID: "production-resource", URL: productionURL},
			},
			expectedID: "production-resource",
		},
		"the installed site is picked when it is listed first": {
			baseURL: productionURL,
			resources: JiraAccessibleResources{
				{ID: "production-resource", URL: productionURL},
				{ID: "sandbox-resource", URL: sandboxURL},
			},
			expectedID: "production-resource",
		},
		"a sandbox instance resolves to the sandbox": {
			baseURL: sandboxURL,
			resources: JiraAccessibleResources{
				{ID: "production-resource", URL: productionURL},
				{ID: "sandbox-resource", URL: sandboxURL},
			},
			expectedID: "sandbox-resource",
		},
		"a single granted site still has to match": {
			baseURL:   productionURL,
			resources: JiraAccessibleResources{{ID: "sandbox-resource", URL: sandboxURL}},
			expectErr: "does not have access to",
		},
		"scheme, case and trailing slash do not cause a false mismatch": {
			baseURL:    productionURL,
			resources:  JiraAccessibleResources{{ID: "production-resource", URL: "http://PRODUCTION.example.com/"}},
			expectedID: "production-resource",
		},
		"no granted sites": {
			baseURL:   productionURL,
			resources: JiraAccessibleResources{},
			expectErr: "No resources are available",
		},
		"a resource with no URL never matches": {
			baseURL:   productionURL,
			resources: JiraAccessibleResources{{ID: "someid"}},
			expectErr: "does not have access to",
		},
	} {
		t.Run(name, func(t *testing.T) {
			resourceID, err := instanceFor(tc.baseURL).selectResourceID(tc.resources)

			if tc.expectErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErr)
				assert.Empty(t, resourceID, "a resource ID must not be returned alongside an error, GetURL would use it")
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expectedID, resourceID)
		})
	}
}

// TestResolveJiraCloudResourceIDUsesCache pins that a resolved ID is reused
// rather than re-derived on every Jira request. The zero-value http.Client
// would fail the lookup, so reaching it at all fails the test.
func TestResolveJiraCloudResourceIDUsesCache(t *testing.T) {
	ci := &cloudOAuthInstance{
		InstanceCommon: &InstanceCommon{InstanceID: "https://jira.example.com", Type: CloudOAuthInstanceType},
		JiraBaseURL:    "https://jira.example.com",
		JiraResourceID: "already-resolved",
	}

	resourceID, err := ci.resolveJiraCloudResourceID(http.Client{})
	require.NoError(t, err)
	assert.Equal(t, "already-resolved", resourceID)
}

func TestCloudOAuthInstanceGetURL(t *testing.T) {
	ci := &cloudOAuthInstance{
		InstanceCommon: &InstanceCommon{InstanceID: "https://jira.example.com", Type: CloudOAuthInstanceType},
		JiraBaseURL:    "https://jira.example.com",
		JiraResourceID: "resource-id",
	}

	assert.Equal(t, "https://api.atlassian.com/ex/jira/resource-id", ci.GetURL())
	assert.Equal(t, "https://jira.example.com", ci.GetJiraBaseURL())
}
