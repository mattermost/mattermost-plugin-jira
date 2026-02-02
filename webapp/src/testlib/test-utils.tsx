// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {RenderOptions, render} from '@testing-library/react';
import {Provider} from 'react-redux';
import {IntlProvider} from 'react-intl';
import configureStore from 'redux-mock-store';
import thunk from 'redux-thunk';

import {InstanceType} from 'types/model';

const mockStore = configureStore([thunk]);

export const mockTheme = {
    centerChannelColor: '#333333',
    centerChannelBg: '#ffffff',
    buttonBg: '#166de0',
    buttonColor: '#ffffff',
    linkColor: '#2389d7',
    errorTextColor: '#fd5960',
};

export const defaultMockState = {
    'plugins-jira': {
        installedInstances: [{instance_id: 'instance1', type: InstanceType.CLOUD}],
        connectedInstances: [{instance_id: 'instance1', type: InstanceType.CLOUD}],
        defaultUserInstanceID: 'instance1',
        jiraProjectMetadata: null,
        jiraIssueMetadata: null,
        channelIdWithSettingsOpen: null,
        channelSubscriptions: {},
    },
    entities: {
        general: {
            config: {
                SiteURL: 'http://localhost:8065',
            },
        },
    },
};

interface CustomRenderOptions extends Omit<RenderOptions, 'wrapper'> {
    initialState?: Record<string, unknown>;
}

export function renderWithRedux(
    ui: React.ReactElement,
    {initialState = defaultMockState, ...renderOptions}: CustomRenderOptions = {},
) {
    const store = mockStore(initialState);

    function Wrapper({children}: {children: React.ReactNode}) {
        return (
            <IntlProvider locale='en'>
                <Provider store={store}>{children}</Provider>
            </IntlProvider>
        );
    }

    return {
        store,
        ...render(ui, {wrapper: Wrapper, ...renderOptions}),
    };
}
