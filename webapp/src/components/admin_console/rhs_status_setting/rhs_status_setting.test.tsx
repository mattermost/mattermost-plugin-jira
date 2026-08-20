// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {fireEvent, screen, waitFor} from '@testing-library/react';

import {Theme} from 'mattermost-redux/selectors/entities/preferences';

import {InstanceType, RHSStatusesResponse} from 'types/model';
import {mockTheme as baseMockTheme, renderWithRedux} from 'testlib/test-utils';

import {RHSFetchError} from '../../../client';

import RHSStatusSetting, {Props} from './rhs_status_setting';
import {
    INSTANCE_LABEL,
    IN_PROGRESS_TAB,
    NOT_CONNECTED_MESSAGE,
    STATUS_TABS_LABEL,
} from './rhs_status_options';

const mockTheme = baseMockTheme as Theme;

const SETTING_ID = 'PluginSettings.Plugins.jira.rhsstatustabs';
const CLOUD_ID = 'https://cloud.example.atlassian.net';
const OAUTH_ID = 'https://oauth.example.atlassian.net';
const SERVER_ID = 'http://jira.example.com';

const cloudInstance = {instance_id: CLOUD_ID, type: InstanceType.CLOUD};
const oauthInstance = {instance_id: OAUTH_ID, type: InstanceType.CLOUD_OAUTH};
const serverInstance = {instance_id: SERVER_ID, type: InstanceType.SERVER};

const poisonConfig = {
    PluginSettings: {
        Plugins: {
            jira: {
                rhsstatustabs: {
                    [CLOUD_ID]: [{kind: 'status', id: '999', name: 'FromConfig'}],
                },
            },
            'com.mattermost.user-survey': {
                systemconsolesetting: {
                    TeamFilter: {filteredTeamIDs: ['nope']},
                },
            },
        },
    },
};

const statusesData: RHSStatusesResponse = {
    categories: [
        {id: 1, key: 'undefined', name: 'No Category'},
        {id: 2, key: 'new', name: 'To Do'},
        {id: 4, key: 'indeterminate', name: 'In Progress'},
        {id: 3, key: 'done', name: 'Done'},
    ],
    statuses: [
        {id: '3', name: 'In Progress', statusCategory: {id: 4, key: 'indeterminate', name: 'In Progress'}},
        {id: '3', name: 'In Progress', statusCategory: {id: 4, key: 'indeterminate', name: 'In Progress'}},
        {id: '10001', name: 'Submitted', statusCategory: {id: 2, key: 'new', name: 'To Do'}},
    ],
};

const baseProps: Props = {
    id: SETTING_ID,
    value: null,
    disabled: false,
    config: poisonConfig,
    onChange: jest.fn(),
    setSaveNeeded: jest.fn(),
    theme: mockTheme as Theme,
    installedInstances: [cloudInstance, oauthInstance, serverInstance],
    fetchRHSStatuses: jest.fn().mockResolvedValue({data: statusesData}),
    getConnected: jest.fn().mockResolvedValue({data: {}}),
};

async function renderSettled(override: Partial<Props> = {}) {
    const props = {
        ...baseProps,
        ...override,
    };

    const view = renderWithRedux(
        <RHSStatusSetting
            {...props}
        />,
    );

    await waitFor(() => {
        expect(props.getConnected).toHaveBeenCalled();
    });

    return {props, ...view};
}

describe('components/RHSStatusSetting', () => {
    beforeEach(() => {
        jest.clearAllMocks();
    });

    test.each([
        [null],
        [{}],
        [{[CLOUD_ID]: []}],
    ])('empty value renders Assigned and In Progress selected without calling onChange', async (value) => {
        const onChange = jest.fn();
        const setSaveNeeded = jest.fn();
        const fetchRHSStatuses = jest.fn().mockResolvedValue({data: statusesData});
        const getConnected = jest.fn().mockResolvedValue({data: {}});

        renderWithRedux(
            <RHSStatusSetting
                {...baseProps}
                value={value}
                onChange={onChange}
                setSaveNeeded={setSaveNeeded}
                fetchRHSStatuses={fetchRHSStatuses}
                getConnected={getConnected}
            />,
        );

        await waitFor(() => {
            expect(fetchRHSStatuses).toHaveBeenCalled();
        });

        expect(screen.getByText('Assigned')).toBeInTheDocument();
        expect(screen.getByText('In Progress')).toBeInTheDocument();
        expect(screen.queryByText('FromConfig')).not.toBeInTheDocument();
        expect(onChange).not.toHaveBeenCalled();
        expect(setSaveNeeded).not.toHaveBeenCalled();
    });

    test('not_connected keeps stored chips disables the status control and does not call onChange', async () => {
        const fetchRHSStatuses = jest.fn().mockResolvedValue({
            error: new RHSFetchError('not_connected', 'Jira account is not connected', 401),
        });

        const {props} = await renderSettled({
            value: {[CLOUD_ID]: [{kind: 'category', key: 'new', name: 'To Do'}]},
            fetchRHSStatuses,
        });

        await waitFor(() => {
            expect(fetchRHSStatuses).toHaveBeenCalled();
        });

        expect(screen.getByText('Assigned')).toBeInTheDocument();
        expect(screen.getByText('To Do')).toBeInTheDocument();
        expect(screen.getByText(NOT_CONNECTED_MESSAGE)).toBeInTheDocument();
        expect(screen.queryByText('FromConfig')).not.toBeInTheDocument();
        expect(screen.getByLabelText(STATUS_TABS_LABEL)).toBeDisabled();
        expect(screen.getByLabelText(INSTANCE_LABEL)).not.toBeDisabled();
        expect(props.onChange).not.toHaveBeenCalled();
        expect(props.setSaveNeeded).not.toHaveBeenCalled();
    });

    test('selection follows props.value not a hardcoded config PluginSettings path', async () => {
        const {props} = await renderSettled({
            value: {[CLOUD_ID]: [{kind: 'status', id: '10001', name: 'Submitted'}]},
        });

        await waitFor(() => {
            expect(props.fetchRHSStatuses).toHaveBeenCalled();
        });

        expect(screen.getByText('Assigned')).toBeInTheDocument();
        expect(screen.getByText('Submitted')).toBeInTheDocument();
        expect(screen.queryByText('FromConfig')).not.toBeInTheDocument();
        expect(props.onChange).not.toHaveBeenCalled();
    });

    test('a failed fetch with empty options does not call onChange', async () => {
        const fetchRHSStatuses = jest.fn().mockRejectedValue(new Error('network down'));

        const {props} = await renderSettled({
            value: {[CLOUD_ID]: [{kind: 'category', key: 'done', name: 'Done'}]},
            fetchRHSStatuses,
        });

        await waitFor(() => {
            expect(fetchRHSStatuses).toHaveBeenCalled();
        });

        expect(screen.getByText('Assigned')).toBeInTheDocument();
        expect(screen.getByText('Done')).toBeInTheDocument();
        expect(props.onChange).not.toHaveBeenCalled();
        expect(props.setSaveNeeded).not.toHaveBeenCalled();
    });

    test('an empty 200 fetch does not call onChange', async () => {
        const fetchRHSStatuses = jest.fn().mockResolvedValue({
            data: {statuses: [], categories: []},
        });

        const {props} = await renderSettled({
            value: {[CLOUD_ID]: [{kind: 'category', key: 'done', name: 'Done'}]},
            fetchRHSStatuses,
        });

        await waitFor(() => {
            expect(fetchRHSStatuses).toHaveBeenCalled();
        });

        expect(screen.getByText('Assigned')).toBeInTheDocument();
        expect(screen.getByText('Done')).toBeInTheDocument();
        expect(props.onChange).not.toHaveBeenCalled();
        expect(props.setSaveNeeded).not.toHaveBeenCalled();
    });

    test('Assigned cannot be deselected', async () => {
        const onChange = jest.fn();
        const {props} = await renderSettled({
            onChange,
        });

        await waitFor(() => {
            expect(props.fetchRHSStatuses).toHaveBeenCalled();
        });

        expect(screen.queryByLabelText('Remove Assigned')).not.toBeInTheDocument();

        const removeInProgress = screen.getByLabelText('Remove In Progress');
        fireEvent.click(removeInProgress);

        await waitFor(() => {
            expect(onChange).toHaveBeenCalled();
        });

        expect(screen.queryByLabelText('Remove Assigned')).not.toBeInTheDocument();
    });

    test('selecting a status calls onChange and setSaveNeeded with extras only', async () => {
        const onChange = jest.fn();
        const setSaveNeeded = jest.fn();
        const {props} = await renderSettled({
            onChange,
            setSaveNeeded,
        });

        await waitFor(() => {
            expect(props.fetchRHSStatuses).toHaveBeenCalled();
        });

        fireEvent.keyDown(screen.getByLabelText(STATUS_TABS_LABEL), {key: 'ArrowDown'});

        const doneOption = await screen.findByText('Done');
        fireEvent.click(doneOption);

        await waitFor(() => {
            expect(onChange).toHaveBeenCalledTimes(1);
        });

        expect(onChange).toHaveBeenCalledWith(SETTING_ID, {
            [CLOUD_ID]: [
                IN_PROGRESS_TAB,
                {kind: 'category', key: 'done', name: 'Done'},
            ],
        });
        expect(setSaveNeeded).toHaveBeenCalled();
        const persisted = onChange.mock.calls[0][1][CLOUD_ID];
        expect(persisted.some((tab: {kind: string}) => tab.kind === 'assigned')).toBe(false);
    });

    test('only Cloud instances are offered', async () => {
        const {props} = await renderSettled();

        await waitFor(() => {
            expect(props.fetchRHSStatuses).toHaveBeenCalled();
        });

        fireEvent.keyDown(screen.getByLabelText(INSTANCE_LABEL), {key: 'ArrowDown'});

        const optionLabels = (await screen.findAllByRole('option')).map((option) => option.textContent);
        expect(optionLabels).toEqual([CLOUD_ID, OAUTH_ID]);
        expect(screen.queryByText(SERVER_ID)).not.toBeInTheDocument();
        expect(screen.queryByText('jira.example.com')).not.toBeInTheDocument();
    });

    test('getConnected is called on mount and a getConnected failure does not call onChange', async () => {
        const getConnected = jest.fn().mockRejectedValue(new Error('nope'));

        const {props} = await renderSettled({
            getConnected,
        });

        await waitFor(() => {
            expect(getConnected).toHaveBeenCalled();
        });

        expect(props.onChange).not.toHaveBeenCalled();
    });
});
