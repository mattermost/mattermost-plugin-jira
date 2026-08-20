// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {applyMiddleware, createStore} from 'redux';
import thunk from 'redux-thunk';

import ConnectModal from 'components/modals/connect_modal';
import Rhs from 'components/rhs';
import {Instance, InstanceType} from 'types/model';
import {pluginStateKey} from 'types/store';
import {JIRA_RHS_TITLE} from 'utils/rhs_register';

import {setupUILater} from './plugin';
import reducer from './reducers';

const cloudOAuth = {instance_id: 'https://oauth.example.atlassian.net', type: InstanceType.CLOUD_OAUTH};
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

function mockPluginFetches(opts: {settings: object; userinfo: object}) {
    const fetchMock = global.fetch as jest.Mock;
    fetchMock.mockImplementation((url: string) => {
        const href = String(url);
        if (href.indexOf('/api/v2/settingsinfo') !== -1) {
            return Promise.resolve({
                ok: true,
                json: () => Promise.resolve(opts.settings),
            });
        }
        if (href.indexOf('/api/v2/userinfo') !== -1) {
            return Promise.resolve({
                ok: true,
                json: () => Promise.resolve(opts.userinfo),
            });
        }
        return Promise.resolve({
            ok: true,
            json: () => Promise.resolve({}),
        });
    });
    return fetchMock;
}

function userinfoWithInstances(instances: Instance[]) {
    return {
        can_connect: true,
        is_connected: instances.length > 0,
        instances,
        user_info: {
            connected_instances: instances,
            default_instance_id: instances[0] ? instances[0].instance_id : '',
        },
    };
}

function makeRegistry() {
    return {
        registerReducer: jest.fn(),
        registerRootComponent: jest.fn(),
        registerPostDropdownMenuAction: jest.fn(),
        registerLinkTooltipComponent: jest.fn(),
        registerAdminConsoleCustomSetting: jest.fn(),
        registerSlashCommandWillBePostedHook: jest.fn(),
        registerWebSocketEventHandler: jest.fn(),
        registerAppBarComponent: jest.fn(),
    };
}

function makeRHSStore() {
    const pluginState = reducer({} as any, {type: '@@INIT'} as any);
    const initialState = {
        entities: {
            general: {
                config: {
                    SiteURL: 'http://localhost:8065',
                },
            },
            users: {
                currentUserId: 'user-1',
            },
        },
        [pluginStateKey]: pluginState,
    };
    const root = (state = initialState, action: any) => {
        return {
            entities: state.entities,
            [pluginStateKey]: reducer(state[pluginStateKey], action),
        };
    };
    return createStore(root, applyMiddleware(thunk));
}

async function runSetup(registry: ReturnType<typeof makeRegistry>, store: ReturnType<typeof makeRHSStore>) {
    await setupUILater(registry, store as any)();
}

function expectAdminSettingRegistered(registry: ReturnType<typeof makeRegistry>) {
    expect(registry.registerAdminConsoleCustomSetting).toHaveBeenCalledWith('RHSStatusTabs', expect.anything(), {showTitle: true});
}

describe('plugin setupUILater', () => {
    beforeEach(() => {
        (global.fetch as jest.Mock).mockReset();
        localStorage.clear();
    });

    afterEach(() => {
        (global.fetch as jest.Mock).mockReset();
        localStorage.clear();
    });

    test('setupUILater does not call registerAppBarComponent when rhs_enabled is false', async () => {
        mockPluginFetches({
            settings: settingsRhsOff,
            userinfo: userinfoWithInstances([cloudOAuth]),
        });
        const registry = makeRegistry();
        const store = makeRHSStore();

        await runSetup(registry, store);

        expect(registry.registerAppBarComponent).not.toHaveBeenCalled();
        expectAdminSettingRegistered(registry);
    });

    test('setupUILater does not call registerAppBarComponent when only Server/DC is installed', async () => {
        mockPluginFetches({
            settings: settingsRhsOn,
            userinfo: userinfoWithInstances([serverDc]),
        });
        const registry = makeRegistry();
        const store = makeRHSStore();

        await runSetup(registry, store);

        expect(registry.registerAppBarComponent).not.toHaveBeenCalled();
        expectAdminSettingRegistered(registry);
    });

    test('setupUILater calls registerAppBarComponent for a cloud-oauth-only installation when rhs_enabled is true', async () => {
        mockPluginFetches({
            settings: settingsRhsOn,
            userinfo: userinfoWithInstances([cloudOAuth]),
        });
        const registry = makeRegistry();
        const store = makeRHSStore();

        await runSetup(registry, store);

        expect(registry.registerAppBarComponent).toHaveBeenCalledTimes(1);
        expect(registry.registerAppBarComponent.mock.calls[0][1]).toBeUndefined();
        expect(registry.registerAppBarComponent.mock.calls[0][4]).toBe(Rhs);
        expect(registry.registerAppBarComponent.mock.calls[0][5]).toBe(JIRA_RHS_TITLE);
        expect(String(registry.registerAppBarComponent.mock.calls[0][0])).toContain('/public/icon.svg');
        expectAdminSettingRegistered(registry);
    });

    test('setupUILater does not nest App Bar registration inside ui_enabled', async () => {
        mockPluginFetches({
            settings: {
                ui_enabled: false,
                rhs_enabled: true,
                security_level_empty_for_jira_subscriptions: true,
            },
            userinfo: userinfoWithInstances([cloudOAuth]),
        });
        const registry = makeRegistry();
        const store = makeRHSStore();

        await runSetup(registry, store);

        expect(registry.registerAppBarComponent).toHaveBeenCalled();
        expect(registry.registerRootComponent).not.toHaveBeenCalledWith(ConnectModal);
        expect(registry.registerPostDropdownMenuAction).not.toHaveBeenCalled();
    });
});
