// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import Rhs from 'components/rhs';

import {Instance, InstanceType} from 'types/model';
import {GlobalState, pluginStateKey} from 'types/store';

import {
    JIRA_RHS_TITLE,
    getJiraAppBarIconUrl,
    registerJiraAppBar,
    shouldRegisterJiraRHS,
} from './rhs_register';

const cloudOAuth = {instance_id: 'https://oauth.example.atlassian.net', type: InstanceType.CLOUD_OAUTH};
const cloudJwt = {instance_id: 'https://cloud.example.atlassian.net', type: InstanceType.CLOUD};
const serverDc = {instance_id: 'http://jira.example.com', type: InstanceType.SERVER};

const settingsRhsOff = {
    ui_enabled: false,
    rhs_enabled: false,
    security_level_empty_for_jira_subscriptions: true,
};
const settingsRhsOn = {
    ...settingsRhsOff,
    rhs_enabled: true,
};

function makeState(plugin: {installedInstances?: Instance[]}): GlobalState {
    return {
        entities: {
            general: {config: {SiteURL: 'http://localhost:8065'}},
        },
        [pluginStateKey]: plugin,
    } as unknown as GlobalState;
}

describe('rhs_register', () => {
    test('shouldRegisterJiraRHS is false when rhs_enabled is false even with a Cloud instance', () => {
        expect(shouldRegisterJiraRHS(settingsRhsOff, makeState({installedInstances: [cloudOAuth]}))).toBe(false);
    });

    test('shouldRegisterJiraRHS is false when rhs_enabled is true but only Server/DC is installed', () => {
        expect(shouldRegisterJiraRHS(settingsRhsOn, makeState({installedInstances: [serverDc]}))).toBe(false);
    });

    test('shouldRegisterJiraRHS is true for a cloud-oauth-only installation when rhs_enabled is true', () => {
        expect(shouldRegisterJiraRHS(settingsRhsOn, makeState({installedInstances: [cloudOAuth]}))).toBe(true);
    });

    test('shouldRegisterJiraRHS is true for a cloud-JWT-only installation when rhs_enabled is true', () => {
        expect(shouldRegisterJiraRHS(settingsRhsOn, makeState({installedInstances: [cloudJwt]}))).toBe(true);
    });

    test('shouldRegisterJiraRHS is false when settings is an error object', () => {
        expect(shouldRegisterJiraRHS({error: true}, makeState({installedInstances: [cloudOAuth]}))).toBe(false);
    });

    test('getJiraAppBarIconUrl uses the plugin public path', () => {
        const url = getJiraAppBarIconUrl(makeState({}));
        expect(url).toContain('/plugins/jira/public/icon.svg');
        expect(url).not.toContain('/api/v2');
    });

    test('registerJiraAppBar calls registerAppBarComponent once with omitted action and Rhs', () => {
        const registry = {registerAppBarComponent: jest.fn()};
        registerJiraAppBar(registry, makeState({installedInstances: [cloudOAuth]}));

        expect(registry.registerAppBarComponent).toHaveBeenCalledTimes(1);
        expect(registry.registerAppBarComponent.mock.calls[0][1]).toBeUndefined();
        expect(registry.registerAppBarComponent.mock.calls[0][2]).toBe(JIRA_RHS_TITLE);
        expect(registry.registerAppBarComponent.mock.calls[0][5]).toBe(JIRA_RHS_TITLE);
        expect(registry.registerAppBarComponent.mock.calls[0][4]).toBe(Rhs);
    });
});
