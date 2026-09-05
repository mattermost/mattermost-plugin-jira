// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {fireEvent, screen, waitFor} from '@testing-library/react';

import {Theme} from 'mattermost-redux/selectors/entities/preferences';

import {InstanceType} from 'types/model';
import {RHSStatusesResponse} from 'types/rhs';
import {mockTheme as baseMockTheme, renderWithRedux} from 'testlib/test-utils';

import {RHSFetchError, getRHSStatuses} from '../../../client/rhs';

import RHSStatusSetting, {Props} from './rhs_status_setting';
import {
    ASSIGNED_TAB,
    INSTANCE_LABEL,
    IN_PROGRESS_TAB,
    NOT_CONNECTED_MESSAGE,
    STATUS_TABS_LABEL,
} from './rhs_status_options';

jest.mock('../../../client/rhs', () => {
    const actual = jest.requireActual('../../../client/rhs');
    return {
        ...actual,
        getRHSStatuses: jest.fn(),
    };
});

const mockGetRHSStatuses = getRHSStatuses as jest.MockedFunction<typeof getRHSStatuses>;

const mockTheme = baseMockTheme as Theme;

const SETTING_ID = 'PluginSettings.Plugins.jira.rhsstatustabs';
const CLOUD_ID = 'https://cloud.example.atlassian.net';
const OAUTH_ID = 'https://oauth.example.atlassian.net';
const SERVER_ID = 'http://jira.example.com';
const PLUGIN_ROUTE = '/plugins/jira';

const cloudInstance = {instance_id: CLOUD_ID, type: InstanceType.CLOUD};
const oauthInstance = {instance_id: OAUTH_ID, type: InstanceType.CLOUD_OAUTH};
const serverInstance = {instance_id: SERVER_ID, type: InstanceType.SERVER};

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
        {
            id: '10042',
            name: 'In Progress',
            statusCategory: {id: 4, key: 'indeterminate', name: 'In Progress'},
            project: {id: '10000', key: 'PLAY', name: 'Playbooks'},
        },
        {
            id: '10043',
            name: 'In Progress',
            statusCategory: {id: 4, key: 'indeterminate', name: 'In Progress'},
            project: {id: '10001', key: 'DSGN', name: 'Design'},
        },
    ],
};

const baseProps: Props = {
    id: SETTING_ID,
    value: null,
    disabled: false,
    onChange: jest.fn(),
    setSaveNeeded: jest.fn(),
    theme: mockTheme as Theme,
    installedInstances: [cloudInstance, oauthInstance, serverInstance],
    pluginServerRoute: PLUGIN_ROUTE,
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
        mockGetRHSStatuses.mockResolvedValue(statusesData);
    });

    test.each([
        [null],
        [{}],
    ])('unset value renders Assigned and In Progress selected without calling onChange', async (value) => {
        const onChange = jest.fn();
        const setSaveNeeded = jest.fn();
        const getConnected = jest.fn().mockResolvedValue({data: {}});

        renderWithRedux(
            <RHSStatusSetting
                {...baseProps}
                value={value}
                onChange={onChange}
                setSaveNeeded={setSaveNeeded}
                getConnected={getConnected}
            />,
        );

        await waitFor(() => {
            expect(mockGetRHSStatuses).toHaveBeenCalled();
        });

        expect(screen.getByText(ASSIGNED_TAB.name)).toBeInTheDocument();
        expect(screen.getByText('In Progress')).toBeInTheDocument();
        expect(onChange).not.toHaveBeenCalled();
        expect(setSaveNeeded).not.toHaveBeenCalled();
    });

    test('stored empty extras render Assigned without reseeding In Progress', async () => {
        const {props} = await renderSettled({
            value: {[CLOUD_ID]: []},
        });

        await waitFor(() => {
            expect(mockGetRHSStatuses).toHaveBeenCalled();
        });

        expect(screen.getByText(ASSIGNED_TAB.name)).toBeInTheDocument();
        expect(screen.queryByLabelText('Remove In Progress')).not.toBeInTheDocument();
        expect(props.onChange).not.toHaveBeenCalled();
    });

    test('changing to In Progress on an unset instance persists the extras array', async () => {
        const onChange = jest.fn();
        const {props} = await renderSettled({
            value: {[CLOUD_ID]: []},
            onChange,
        });

        await waitFor(() => {
            expect(mockGetRHSStatuses).toHaveBeenCalled();
        });

        fireEvent.keyDown(screen.getByLabelText(STATUS_TABS_LABEL), {key: 'ArrowDown'});
        const inProgressOption = await screen.findByText('In Progress');
        fireEvent.click(inProgressOption);

        await waitFor(() => {
            expect(onChange).toHaveBeenCalledWith(SETTING_ID, {[CLOUD_ID]: [IN_PROGRESS_TAB]});
        });
        expect(props.setSaveNeeded).toHaveBeenCalled();
    });

    test('removing the last extra tab persists Assigned only', async () => {
        const onChange = jest.fn();
        const {props} = await renderSettled({
            onChange,
        });

        await waitFor(() => {
            expect(mockGetRHSStatuses).toHaveBeenCalled();
        });

        fireEvent.click(screen.getByLabelText('Remove In Progress'));

        await waitFor(() => {
            expect(onChange).toHaveBeenCalledWith(SETTING_ID, {[CLOUD_ID]: []});
        });
        expect(props.setSaveNeeded).toHaveBeenCalled();
    });

    test('removing a stored last extra tab persists Assigned only', async () => {
        const onChange = jest.fn();
        const {props} = await renderSettled({
            value: {[CLOUD_ID]: [{kind: 'category', key: 'done', name: 'Done'}]},
            onChange,
        });

        await waitFor(() => {
            expect(mockGetRHSStatuses).toHaveBeenCalled();
        });

        fireEvent.click(screen.getByLabelText('Remove Done'));

        await waitFor(() => {
            expect(onChange).toHaveBeenCalledWith(SETTING_ID, {[CLOUD_ID]: []});
        });
    });

    test('not_connected keeps stored chips disables the status control and does not call onChange', async () => {
        mockGetRHSStatuses.mockRejectedValue(new RHSFetchError('not_connected', 'Jira account is not connected', 401));

        const {props} = await renderSettled({
            value: {[CLOUD_ID]: [{kind: 'category', key: 'new', name: 'To Do'}]},
        });

        await waitFor(() => {
            expect(mockGetRHSStatuses).toHaveBeenCalled();
        });

        expect(screen.getByText(ASSIGNED_TAB.name)).toBeInTheDocument();
        expect(screen.getByText('To Do')).toBeInTheDocument();
        expect(screen.getByText(NOT_CONNECTED_MESSAGE)).toBeInTheDocument();
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
            expect(mockGetRHSStatuses).toHaveBeenCalled();
        });

        expect(screen.getByText(ASSIGNED_TAB.name)).toBeInTheDocument();
        expect(screen.getByText('Submitted')).toBeInTheDocument();
        expect(screen.queryByText('FromConfig')).not.toBeInTheDocument();
        expect(props.onChange).not.toHaveBeenCalled();
    });

    test('a failed fetch with empty options does not call onChange', async () => {
        mockGetRHSStatuses.mockRejectedValue(new Error('network down'));

        const {props} = await renderSettled({
            value: {[CLOUD_ID]: [{kind: 'category', key: 'done', name: 'Done'}]},
        });

        await waitFor(() => {
            expect(mockGetRHSStatuses).toHaveBeenCalled();
        });

        expect(screen.getByText(ASSIGNED_TAB.name)).toBeInTheDocument();
        expect(screen.getByText('Done')).toBeInTheDocument();
        expect(props.onChange).not.toHaveBeenCalled();
        expect(props.setSaveNeeded).not.toHaveBeenCalled();
    });

    test('an empty 200 fetch does not call onChange', async () => {
        mockGetRHSStatuses.mockResolvedValue({
            statuses: [],
            categories: [],
        });

        const {props} = await renderSettled({
            value: {[CLOUD_ID]: [{kind: 'category', key: 'done', name: 'Done'}]},
        });

        await waitFor(() => {
            expect(mockGetRHSStatuses).toHaveBeenCalled();
        });

        expect(screen.getByText(ASSIGNED_TAB.name)).toBeInTheDocument();
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
            expect(mockGetRHSStatuses).toHaveBeenCalled();
        });

        expect(screen.queryByLabelText('Remove ' + ASSIGNED_TAB.name)).not.toBeInTheDocument();

        const removeInProgress = screen.getByLabelText('Remove In Progress');
        fireEvent.click(removeInProgress);

        await waitFor(() => {
            expect(onChange).toHaveBeenCalled();
        });

        expect(screen.queryByLabelText('Remove ' + ASSIGNED_TAB.name)).not.toBeInTheDocument();
    });

    test('selecting a status calls onChange and setSaveNeeded with extras only', async () => {
        const onChange = jest.fn();
        const setSaveNeeded = jest.fn();
        const {props} = await renderSettled({
            onChange,
            setSaveNeeded,
        });

        await waitFor(() => {
            expect(mockGetRHSStatuses).toHaveBeenCalled();
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
        expect(mockGetRHSStatuses).toHaveBeenCalledWith(PLUGIN_ROUTE, CLOUD_ID);
        expect(props.getConnected).toHaveBeenCalled();
    });

    test('only Cloud instances are offered', async () => {
        await renderSettled();

        await waitFor(() => {
            expect(mockGetRHSStatuses).toHaveBeenCalled();
        });

        fireEvent.keyDown(screen.getByLabelText(INSTANCE_LABEL), {key: 'ArrowDown'});

        const optionLabels = (await screen.findAllByRole('option')).map((option) => option.textContent);
        expect(optionLabels).toEqual([CLOUD_ID, OAUTH_ID]);
        expect(screen.queryByText(SERVER_ID)).not.toBeInTheDocument();
        expect(screen.queryByText('jira.example.com')).not.toBeInTheDocument();
    });

    test('status options show the Jira project under the status name', async () => {
        await renderSettled();

        await waitFor(() => {
            expect(mockGetRHSStatuses).toHaveBeenCalled();
        });

        fireEvent.keyDown(screen.getByLabelText(STATUS_TABS_LABEL), {key: 'ArrowDown'});

        expect(await screen.findByText('Playbooks (PLAY)')).toBeInTheDocument();
        expect(screen.getByText('Design (DSGN)')).toBeInTheDocument();
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
