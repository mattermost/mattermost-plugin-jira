// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useEffect, useState} from 'react';

import type {ResolveAndFetchArgs} from 'actions';

import {Instance} from 'types/model';
import {
    RHSErrorCode,
    RHSIssue,
    RHSSort,
    RHSTab,
    RHSViewState,
    RHS_DEFAULT_TAB,
} from 'types/rhs';
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

type RHSPanelKind = 'loading' | 'not_connected' | 'rate_limited' | 'error' | 'empty' | 'list';

function rhsPanelKind(args: {
    booting: boolean;
    loading: boolean;
    issuesLength: number;
    error: RHSErrorCode | null;
    connectedCloudLength: number;
}): RHSPanelKind {
    if (args.booting || (args.loading && args.issuesLength === 0 && args.error === null)) {
        return 'loading';
    }
    if (args.connectedCloudLength === 0 || args.error === 'not_connected') {
        return 'not_connected';
    }
    if (args.error === 'rate_limited' && args.issuesLength === 0) {
        return 'rate_limited';
    }
    if (args.error !== null && args.issuesLength === 0) {
        return 'error';
    }
    if (args.issuesLength === 0) {
        return 'empty';
    }
    return 'list';
}

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

    const isPopout = window.location.pathname.indexOf('/_popout/') === 0;
    const [booting, setBooting] = useState(true);

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

    // Boot once on mount. Empty deps are intentional: connect + hydrate + first fetch
    // must not re-run when callback prop identities change.
    // eslint-disable-next-line react-hooks/exhaustive-deps
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

    const panel = rhsPanelKind({
        booting,
        loading,
        issuesLength: issues.length,
        error,
        connectedCloudLength: connectedCloud.length,
    });

    let body: JSX.Element;
    switch (panel) {
    case 'loading':
        body = <RHSLoadingState/>;
        break;
    case 'not_connected':
        body = (
            <RHSNotConnectedState
                onConnect={onConnect}
            />
        );
        break;
    case 'rate_limited':
        body = (
            <RHSRateLimitedState
                onRetry={onRetry}
            />
        );
        break;
    case 'error':
        body = (
            <RHSErrorState
                onRetry={onRetry}
            />
        );
        break;
    case 'empty':
        body = <RHSEmptyState/>;
        break;
    case 'list':
        body = (
            <RHSIssueList
                issues={issues}
                loading={loading}
                isLast={isLast}
                error={error}
                onLoadMore={onLoadMore}
                onRetry={onRetry}
            />
        );
        break;
    default: {
        const exhaustive: never = panel;
        body = exhaustive;
        break;
    }
    }

    const instancePicker = connectedCloud.length > 1 ? (
        <RHSInstancePicker
            instances={connectedCloud}
            selectedInstanceID={instanceID}
            onChange={onInstanceChange}
        />
    ) : null;

    const showChrome = panel !== 'not_connected';
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
