// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {RenderOptions, render} from '@testing-library/react';
import {Provider} from 'react-redux';
import configureStore from 'redux-mock-store';
import thunk from 'redux-thunk';

import {InstanceType} from 'types/model';

const mockStore = configureStore([thunk]);

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
        return <Provider store={store}>{children}</Provider>;
    }

    return {
        store,
        ...render(ui, {wrapper: Wrapper, ...renderOptions}),
    };
}
