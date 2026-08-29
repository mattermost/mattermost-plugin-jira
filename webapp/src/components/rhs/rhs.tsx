// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useEffect, useState} from 'react';

import type {ResolveAndFetchArgs} from 'actions';

import {
    Instance,
    RHSErrorCode,
    RHSIssue,
    RHSSort,
    RHSTab,
    RHSViewState,
    RHS_DEFAULT_TAB,
} from 'types/model';

import {isRHSPopoutPathname} from 'utils/rhs_popout';
import {rhsTabsEqual} from 'utils/rhs_resolve';

import RHSHeader from './rhs_header';
import RHSInstancePicker from './rhs_instance_picker';
import RHSIssueList from './rhs_issue_list';
import {
    RHSEmptyState,
    RHSErrorState,
    RHSLoadingState,
    RHSNotConnectedState,
    RHSRateLimitedState,
} from './rhs_states';
import RHSTabStrip, {RHS_TAB_PANEL_ID, rhsTabDomId} from './rhs_tab_strip';

import './rhs.scss';

export type Props = {
    connectedCloud: Instance[];
    channelId: string;
    instanceID: string;
    tab: RHSTab;
    sort: RHSSort;
    issues: RHSIssue[];
    tabs: RHSTab[];
    isLast: boolean;
    loading: boolean;
    error: RHSErrorCode | null;
    getConnected: () => Promise<{data?: unknown; error?: unknown}>;
    restoreRHSViewState: () => {data: RHSViewState | null};
    resolveAndFetchRHSIssues: (overrides?: ResolveAndFetchArgs) => Promise<{data?: unknown; error?: unknown}>;
    loadMoreRHSIssues: () => Promise<unknown>;
    openCreateModalWithoutPost: (description: string, channelId: string) => void;
    handleConnectFlow: () => void;
};

export default function Rhs(props: Props): JSX.Element {
    const {
        connectedCloud,
        channelId,
        instanceID,
        tab,
        sort,
        issues,
        tabs,
        isLast,
        loading,
        error,
        getConnected,
        restoreRHSViewState,
        resolveAndFetchRHSIssues,
        loadMoreRHSIssues,
        openCreateModalWithoutPost,
        handleConnectFlow,
    } = props;

    const isPopout = isRHSPopoutPathname(window.location.pathname);
    const [booting, setBooting] = useState(true);
    const nowMs = Date.now();

    useEffect(() => {
        let cancelled = false;

        const boot = async () => {
            await getConnected();
            if (cancelled) {
                return;
            }
            restoreRHSViewState();
            if (cancelled) {
                return;
            }
            await resolveAndFetchRHSIssues();
            if (!cancelled) {
                setBooting(false);
            }
        };

        boot();

        return () => {
            cancelled = true;
        };
    }, []);

    const onSelectTab = (next: RHSTab) => {
        if (rhsTabsEqual(next, tab)) {
            return;
        }
        resolveAndFetchRHSIssues({tab: next});
    };

    const onSortChange = (next: RHSSort) => {
        resolveAndFetchRHSIssues({sort: next});
    };

    const onInstanceChange = (next: string) => {
        resolveAndFetchRHSIssues({instanceID: next, tab: RHS_DEFAULT_TAB});
    };

    const onRetry = () => {
        resolveAndFetchRHSIssues();
    };

    const onLoadMore = () => {
        if (!loading && !isLast) {
            loadMoreRHSIssues();
        }
    };

    const onRefresh = () => {
        if (!loading) {
            resolveAndFetchRHSIssues();
        }
    };

    const onNewTicket = () => {
        openCreateModalWithoutPost('', channelId);
    };

    const onConnect = () => {
        handleConnectFlow();
    };

    const showPage1Loading = booting || (loading && issues.length === 0 && error === null);
    const showNotConnected = !showPage1Loading && (connectedCloud.length === 0 || error === 'not_connected');
    const showRateLimited = !showPage1Loading && !showNotConnected && error === 'rate_limited' && issues.length === 0;
    const showError = !showPage1Loading && !showNotConnected && !showRateLimited && error !== null && issues.length === 0;
    const showEmpty = !showPage1Loading && !showNotConnected && !showError && !showRateLimited && issues.length === 0;

    let body: JSX.Element;
    if (showPage1Loading) {
        body = <RHSLoadingState/>;
    } else if (showNotConnected) {
        body = (
            <RHSNotConnectedState
                onConnect={onConnect}
            />
        );
    } else if (showRateLimited) {
        body = (
            <RHSRateLimitedState
                onRetry={onRetry}
            />
        );
    } else if (showError) {
        body = (
            <RHSErrorState
                onRetry={onRetry}
            />
        );
    } else if (showEmpty) {
        body = <RHSEmptyState/>;
    } else {
        body = (
            <RHSIssueList
                issues={issues}
                loading={loading}
                isLast={isLast}
                error={error}
                nowMs={nowMs}
                onLoadMore={onLoadMore}
                onRetry={onRetry}
            />
        );
    }

    const instancePicker = connectedCloud.length > 1 ? (
        <RHSInstancePicker
            instances={connectedCloud}
            selectedInstanceID={instanceID}
            onChange={onInstanceChange}
        />
    ) : null;

    const showChrome = !showNotConnected;
    const showTabs = showChrome && tabs.length > 0;

    return (
        <div
            className='jira-rhs'
            data-testid='jira-rhs'
            data-rhs-popout={isPopout ? 'true' : 'false'}
        >
            {showChrome && (
                <RHSHeader
                    sort={sort}
                    loading={booting || loading}
                    onSortChange={onSortChange}
                    onRefresh={onRefresh}
                    onNewTicket={onNewTicket}
                    instancePicker={instancePicker}
                />
            )}
            {showTabs && (
                <RHSTabStrip
                    tabs={tabs}
                    selectedTab={tab}
                    onSelect={onSelectTab}
                />
            )}
            <div
                className='jira-rhs-body'
                {...(showTabs ? {
                    role: 'tabpanel',
                    id: RHS_TAB_PANEL_ID,
                    'aria-labelledby': rhsTabDomId(tab),
                } : {})}
            >
                {body}
            </div>
        </div>
    );
}
