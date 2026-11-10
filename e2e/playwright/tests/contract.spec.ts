// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {expect, test} from '@playwright/test';

import {ContractValidator} from '../fake-jira/contract/validator';
import {cloudUser} from '../fake-jira/state';
import {fakeCloudSite} from '../support/constants';

const validator = new ContractValidator();

/**
 * @objective Verify fake-jira's Cloud user matches Atlassian's published User schema.
 */
test('validates fake Cloud user against the published User schema', {tag: '@jira_plugin'}, () => {
    // # Validate the fake user for both REST versions
    const v2 = validator.validateResponse('GET', '/rest/api/2/myself', 200, cloudUser(fakeCloudSite));
    const v3 = validator.validateResponse('GET', '/rest/api/3/myself', 200, cloudUser(fakeCloudSite));

    // * Verify there are no violations
    expect(v2).toEqual([]);
    expect(v3).toEqual([]);
});

/**
 * @objective Verify the contract validator catches fields Jira Cloud does not return.
 */
test('rejects fields Jira Cloud does not return', {tag: '@jira_plugin'}, () => {
    // # Validate a user with a Server-only field
    const errors = validator.validateResponse('GET', '/rest/api/2/myself', 200, {...cloudUser(fakeCloudSite), username: 'not-a-cloud-field'});

    // * Verify the extra field is reported
    expect(errors.join('\n')).toContain('additionalProperty');
});

/**
 * @objective Verify the contract validator flags routes missing from operations.json.
 */
test('flags routes missing from the contract', {tag: '@jira_plugin'}, () => {
    // # Validate a request to an unknown route
    const errors = validator.validateRequest('GET', '/rest/api/2/not-a-real-endpoint', undefined);

    // * Verify the route is reported as uncovered
    expect(errors).toEqual(['GET /rest/api/2/not-a-real-endpoint is not covered by fake-jira/contract/operations.json']);
});
