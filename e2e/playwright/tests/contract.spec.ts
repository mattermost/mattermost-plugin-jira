// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {expect, test} from '@playwright/test';

import {ContractValidator} from '../fake-jira/contract/validator';
import {cloudUser} from '../fake-jira/state';
import {fakeCloudSite} from '../support/constants';

const validator = new ContractValidator();

test('fake Cloud user matches the published User schema', () => {
    expect(validator.validateResponse('GET', '/rest/api/2/myself', 200, cloudUser(fakeCloudSite))).toEqual([]);
    expect(validator.validateResponse('GET', '/rest/api/3/myself', 200, cloudUser(fakeCloudSite))).toEqual([]);
});

test('validator rejects fields Jira Cloud does not return', () => {
    const errors = validator.validateResponse('GET', '/rest/api/2/myself', 200, {...cloudUser(fakeCloudSite), username: 'not-a-cloud-field'});

    expect(errors.join('\n')).toContain('additionalProperty');
});

test('validator flags routes missing from the contract', () => {
    expect(validator.validateRequest('GET', '/rest/api/2/not-a-real-endpoint', undefined)).toEqual([
        'GET /rest/api/2/not-a-real-endpoint is not covered by fake-jira/contract/operations.json',
    ]);
});
