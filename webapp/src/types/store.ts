// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {GlobalState as ReduxGlobalState} from 'mattermost-redux/types/store';

import type combinedReducers from '../reducers';

export type GlobalState = ReduxGlobalState & {
    'plugins-jira': PluginState
};

export type PluginState = ReturnType<typeof combinedReducers>

export const pluginStateKey = 'plugins-jira' as const;
