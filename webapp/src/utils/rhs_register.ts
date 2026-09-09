// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {ComponentType} from 'react';

import Rhs from 'components/rhs';

import {getPluginServerRoute, hasCloudInstance} from '../selectors';

import {GlobalState} from 'types/store';

export const JIRA_RHS_TITLE = 'Jira';

type AppBarRegistry = {
    registerAppBarComponent: (
        iconUrl: string,
        action: undefined,
        tooltipText: string,
        dropdown?: undefined,
        rhsComponent?: ComponentType,
        rhsTitle?: string,
    ) => void;
};

export function shouldRegisterJiraRHS(
    settings: unknown,
    state: GlobalState,
): boolean {
    if (!settings || typeof settings !== 'object') {
        return false;
    }
    if (!('rhs_enabled' in settings) || settings.rhs_enabled !== true) {
        return false;
    }
    return hasCloudInstance(state);
}

export function getJiraAppBarIconUrl(state: GlobalState): string {
    return getPluginServerRoute(state) + '/public/icon.svg';
}

export function registerJiraAppBar(registry: AppBarRegistry, state: GlobalState): void {
    // App Bar omits the channel-header action and dropdown; pass undefined positionally.
    /* eslint-disable no-undefined */
    registry.registerAppBarComponent(
        getJiraAppBarIconUrl(state),
        undefined,
        JIRA_RHS_TITLE,
        undefined,
        Rhs,
        JIRA_RHS_TITLE,
    );
    /* eslint-enable no-undefined */
}
