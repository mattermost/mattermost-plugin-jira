// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import Rhs from 'components/rhs';

import {getPluginServerRoute, hasCloudInstance} from '../selectors';

import {GlobalState} from 'types/store';

export const JIRA_RHS_TITLE = 'Jira';

const omittedAppBarArg = [][0];

export function shouldRegisterJiraRHS(
    settings: {rhs_enabled?: boolean} | null | undefined,
    state: GlobalState,
): boolean {
    if (!settings || settings.rhs_enabled !== true) {
        return false;
    }
    return hasCloudInstance(state);
}

export function getJiraAppBarIconUrl(state: GlobalState): string {
    return getPluginServerRoute(state) + '/public/icon.svg';
}

export function registerJiraAppBar(registry: any, state: GlobalState): void {
    registry.registerAppBarComponent(
        getJiraAppBarIconUrl(state),
        omittedAppBarArg,
        JIRA_RHS_TITLE,
        omittedAppBarArg,
        Rhs,
        JIRA_RHS_TITLE,
    );
}
