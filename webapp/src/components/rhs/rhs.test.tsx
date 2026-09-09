// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {
    fireEvent,
    render,
    screen,
    waitFor,
} from '@testing-library/react';

import {Instance, InstanceType} from 'types/model';
import {RHSIssue, RHSTab} from 'types/rhs';

import Rhs, {Props} from './rhs';
import {RHS_STRINGS} from './rhs_strings';

const assignedTab: RHSTab = {kind: 'assigned', name: 'Assigned to me'};
const inProgressTab: RHSTab = {kind: 'category', name: 'In Progress', key: 'indeterminate'};

const cloudOne: Instance = {instance_id: 'https://one.atlassian.net', type: InstanceType.CLOUD};
const cloudTwo: Instance = {instance_id: 'https://two.atlassian.net', type: InstanceType.CLOUD_OAUTH};

const issueOne: RHSIssue = {
    key: 'TES-1',
    summary: 'One',
    browseUrl: 'https://example.atlassian.net/browse/TES-1',
    status: {name: 'In Progress', categoryKey: 'indeterminate'},
    issueType: 'Task',
    project: 'TES',
    updated: '2026-01-02T00:00:00Z',
};

function makeProps(overrides: Partial<Props> = {}): Props {
    return {
        connectedCloud: [cloudOne],
        channelId: 'channel-1',
        instanceID: cloudOne.instance_id,
        tab: assignedTab,
        sort: 'updated',
        issues: [],
        tabs: [],
        isLast: true,
        loading: false,
        error: null,
        getConnected: jest.fn(async () => {
            return {};
        }),
        restoreRHSViewState: jest.fn(() => {
            return {data: null};
        }),
        resolveAndFetchRHSIssues: jest.fn(async () => {
            return {};
        }),
        loadMoreRHSIssues: jest.fn(async () => {
            return {};
        }),
        openCreateModalWithoutPost: jest.fn(),
        handleConnectFlow: jest.fn(),
        ...overrides,
    };
}

function renderRHS(overrides: Partial<Props> = {}) {
    const props = makeProps(overrides);
    const view = render(
        <Rhs
            {...props}
        />,
    );
    return {props, ...view};
}

async function renderSettled(overrides: Partial<Props> = {}) {
    const result = renderRHS(overrides);
    await waitFor(() => {
        expect(result.props.resolveAndFetchRHSIssues).toHaveBeenCalled();
    });
    return result;
}

describe('components/rhs', () => {
    test('loading state is not the empty state', async () => {
        await renderSettled({
            loading: true,
            issues: [],
            error: null,
            connectedCloud: [cloudOne],
        });

        expect(screen.getByTestId('rhs-state-loading')).toBeInTheDocument();
        expect(screen.queryByTestId('rhs-state-empty')).toBeNull();
        expect(screen.queryByText(RHS_STRINGS.empty)).toBeNull();
        expect(screen.getByText('Loading')).toBeInTheDocument();
    });

    test('empty state renders after a successful zero-row fetch', async () => {
        await renderSettled({
            loading: false,
            issues: [],
            error: null,
            connectedCloud: [cloudOne],
        });

        expect(screen.getByTestId('rhs-state-empty')).toBeInTheDocument();
        expect(screen.queryByTestId('rhs-state-loading')).toBeNull();
        expect(screen.getByText(RHS_STRINGS.empty)).toBeInTheDocument();
    });

    test('error state renders for internal_error', async () => {
        const {props} = await renderSettled({
            error: 'internal_error',
            issues: [],
            loading: false,
        });

        expect(screen.getByTestId('rhs-state-error')).toBeInTheDocument();
        const retry = screen.getByText(RHS_STRINGS.retry);
        expect(retry).toHaveClass('btn', 'btn-secondary', 'btn-sm');
        (props.resolveAndFetchRHSIssues as jest.Mock).mockClear();
        fireEvent.click(retry);
        expect(props.resolveAndFetchRHSIssues).toHaveBeenCalled();
    });

    test('not_connected error renders the connect state', async () => {
        const {props} = await renderSettled({
            error: 'not_connected',
        });

        expect(screen.getByTestId('rhs-state-not-connected')).toBeInTheDocument();
        expect(screen.queryByTestId('rhs-sort')).toBeNull();
        expect(screen.queryByTestId('rhs-refresh')).toBeNull();
        expect(screen.queryByTestId('rhs-new-ticket')).toBeNull();
        expect(screen.queryByTestId('rhs-tab-strip')).toBeNull();
        fireEvent.click(screen.getByTestId('rhs-connect'));
        expect(props.handleConnectFlow).toHaveBeenCalled();
        expect(props.handleConnectFlow).toHaveBeenCalledTimes(1);
    });

    test('zero connected Cloud instances renders not-connected without fetching', async () => {
        await renderSettled({
            connectedCloud: [],
            loading: false,
            error: null,
            issues: [],
        });

        expect(screen.getByTestId('rhs-state-not-connected')).toBeInTheDocument();
    });

    test('rate_limited renders the distinct rate-limit state', async () => {
        await renderSettled({
            error: 'rate_limited',
            issues: [],
            loading: false,
        });

        expect(screen.getByTestId('rhs-state-rate-limited')).toBeInTheDocument();
        expect(screen.getByText(RHS_STRINGS.rateLimited)).toBeInTheDocument();
        expect(screen.queryByTestId('rhs-state-error')).toBeNull();
    });

    test('instance picker is absent with one connected Cloud instance', async () => {
        await renderSettled({
            connectedCloud: [cloudOne],
        });

        expect(screen.queryByTestId('rhs-instance-picker')).toBeNull();
    });

    test('instance picker is present with two connected Cloud instances', async () => {
        await renderSettled({
            connectedCloud: [cloudOne, cloudTwo],
        });

        expect(screen.getByTestId('rhs-instance-picker')).toBeInTheDocument();
    });

    test('changing sort dispatches resolveAndFetchRHSIssues with created', async () => {
        const {props} = await renderSettled();
        (props.resolveAndFetchRHSIssues as jest.Mock).mockClear();

        fireEvent.change(screen.getByTestId('rhs-sort'), {target: {value: 'created'}});

        expect(props.resolveAndFetchRHSIssues).toHaveBeenCalledWith({sort: 'created'});
    });

    test('selecting a tab dispatches resolveAndFetchRHSIssues with that tab', async () => {
        const {props} = await renderSettled({
            tabs: [assignedTab, inProgressTab],
            tab: assignedTab,
        });
        (props.resolveAndFetchRHSIssues as jest.Mock).mockClear();

        fireEvent.click(screen.getByText(inProgressTab.name));

        expect(props.resolveAndFetchRHSIssues).toHaveBeenCalledWith({tab: inProgressTab});
    });

    test('arrow keys move between tabs', async () => {
        const {props} = await renderSettled({
            tabs: [assignedTab, inProgressTab],
            tab: assignedTab,
        });
        (props.resolveAndFetchRHSIssues as jest.Mock).mockClear();

        const assigned = screen.getByRole('tab', {name: assignedTab.name});
        assigned.focus();
        fireEvent.keyDown(assigned, {key: 'ArrowRight'});

        expect(props.resolveAndFetchRHSIssues).toHaveBeenCalledWith({tab: inProgressTab});
    });

    test('load more is shown when not last and click calls loadMoreRHSIssues', async () => {
        const {props} = await renderSettled({
            issues: [issueOne],
            isLast: false,
            loading: false,
        });

        fireEvent.click(screen.getByTestId('rhs-load-more'));
        expect(props.loadMoreRHSIssues).toHaveBeenCalled();
    });

    test('refresh is disabled while loading so a click is not treated as a page-1 replace', async () => {
        await renderSettled({
            issues: [issueOne],
            loading: true,
        });

        expect(screen.getByTestId('rhs-refresh')).toBeDisabled();
        expect(screen.queryByTestId('rhs-state-loading')).toBeNull();
        expect(screen.getByTestId('rhs-issue-list')).toBeInTheDocument();
        expect(screen.getByTestId('rhs-list-loading')).toBeInTheDocument();
        expect(screen.queryByTestId('rhs-load-more')).toBeNull();
    });

    test('new ticket dispatches openCreateModalWithoutPost with empty description and the channel id', async () => {
        const {props} = await renderSettled({
            channelId: 'channel-1',
        });

        fireEvent.click(screen.getByTestId('rhs-new-ticket'));
        expect(props.openCreateModalWithoutPost).toHaveBeenCalledWith('', 'channel-1');
    });

    test('on mount getConnected runs before restoreRHSViewState before resolveAndFetchRHSIssues', async () => {
        const order: string[] = [];
        const getConnected = jest.fn(async () => {
            order.push('connected');
            return {};
        });
        const restoreRHSViewState = jest.fn(() => {
            order.push('restore');
            return {data: null};
        });
        const resolveAndFetchRHSIssues = jest.fn(async () => {
            order.push('fetch');
            return {};
        });

        renderRHS({
            getConnected,
            restoreRHSViewState,
            resolveAndFetchRHSIssues,
        });

        await waitFor(() => {
            expect(order).toEqual(['connected', 'restore', 'fetch']);
        });
    });

    test('booting shows loading not empty', () => {
        renderRHS({
            getConnected: jest.fn(() => new Promise(() => {
                // never resolves; first paint must not be empty
            })),
            loading: false,
            issues: [],
            error: null,
            connectedCloud: [cloudOne],
        });

        expect(screen.getByTestId('rhs-state-loading')).toBeInTheDocument();
        expect(screen.queryByTestId('rhs-state-empty')).toBeNull();
    });
});
