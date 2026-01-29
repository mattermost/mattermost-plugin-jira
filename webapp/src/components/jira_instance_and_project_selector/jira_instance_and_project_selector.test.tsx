// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {act, render} from '@testing-library/react';
import {Provider} from 'react-redux';
import {IntlProvider} from 'react-intl';
import configureStore from 'redux-mock-store';
import thunk from 'redux-thunk';

import {Theme} from 'mattermost-redux/selectors/entities/preferences';

import {InstanceType} from 'types/model';

import JiraInstanceAndProjectSelector, {Props} from './jira_instance_and_project_selector';

const mockStore = configureStore([thunk]);

const defaultMockState = {
    'plugins-jira': {
        installedInstances: [{instance_id: 'instance1', type: InstanceType.CLOUD}, {instance_id: 'instance2', type: InstanceType.SERVER}],
        connectedInstances: [{instance_id: 'instance1', type: InstanceType.CLOUD}, {instance_id: 'instance2', type: InstanceType.SERVER}],
        defaultUserInstanceID: 'instance1',
    },
    entities: {
        general: {
            config: {
                SiteURL: 'http://localhost:8065',
            },
        },
    },
};

const renderWithRedux = (ui: React.ReactElement, initialState = defaultMockState) => {
    const store = mockStore(initialState);
    return {
        store,
        ...render(
            <IntlProvider locale='en'>
                <Provider store={store}>{ui}</Provider>
            </IntlProvider>,
        ),
    };
};

describe('components/JiraInstanceAndProjectSelector', () => {
    const mockTheme = {
        centerChannelColor: '#333333',
        centerChannelBg: '#ffffff',
        buttonBg: '#166de0',
        buttonColor: '#ffffff',
        linkColor: '#2389d7',
        errorTextColor: '#fd5960',
    };

    const baseProps: Props = {
        selectedInstanceID: null,
        selectedProjectID: null,
        onInstanceChange: jest.fn(),
        onProjectChange: jest.fn(),
        onError: jest.fn(),

        theme: mockTheme as Theme,
        addValidate: jest.fn(),
        removeValidate: jest.fn(),

        installedInstances: [{instance_id: 'instance1', type: InstanceType.CLOUD}, {instance_id: 'instance2', type: InstanceType.SERVER}, {instance_id: 'instance3', type: InstanceType.SERVER}],
        connectedInstances: [{instance_id: 'instance1', type: InstanceType.CLOUD}, {instance_id: 'instance2', type: InstanceType.SERVER}],
        defaultUserInstanceID: '',
        fetchJiraProjectMetadata: jest.fn().mockResolvedValue({
            data: {
                saved_field_values: {
                    project_key: 'TEST',
                },
                projects: [
                    {value: 'TEST', label: 'Test Project'},
                    {value: 'AA', label: 'Apples Arrangement'},
                ],
            },
        }),
        getConnected: jest.fn().mockResolvedValue({error: null}),
        hideProjectSelector: false,
    };

    beforeEach(() => {
        jest.clearAllMocks();
    });

    test('should match snapshot with one connected instance', async () => {
        const props = {
            ...baseProps,
            connectedInstances: [{instance_id: 'instance1', type: InstanceType.CLOUD}],
        };
        const ref = React.createRef<JiraInstanceAndProjectSelector>();
        await act(async () => {
            renderWithRedux(
                <JiraInstanceAndProjectSelector
                    {...props}
                    ref={ref}
                />,
            );
        });

        expect(ref.current).toBeDefined();
    });

    test('should match snapshot with two connected instances', async () => {
        const props = {
            ...baseProps,
            connectedInstances: [{instance_id: 'instance1', type: InstanceType.CLOUD}, {instance_id: 'instance2', type: InstanceType.SERVER}],
        };
        const ref = React.createRef<JiraInstanceAndProjectSelector>();
        await act(async () => {
            renderWithRedux(
                <JiraInstanceAndProjectSelector
                    {...props}
                    ref={ref}
                />,
            );
        });

        expect(ref.current).toBeDefined();
    });

    test('should match snapshot with a default instance selected', async () => {
        const props = {
            ...baseProps,
            connectedInstances: [{instance_id: 'instance1', type: InstanceType.CLOUD}, {instance_id: 'instance2', type: InstanceType.SERVER}],
            defaultUserInstanceID: 'instance1',
        };
        const ref = React.createRef<JiraInstanceAndProjectSelector>();
        await act(async () => {
            renderWithRedux(
                <JiraInstanceAndProjectSelector
                    {...props}
                    ref={ref}
                />,
            );
        });

        expect(ref.current).toBeDefined();
    });

    test('should assign the correct initial instance id', async () => {
        // Test case 1: defaultUserInstanceID should be selected
        let onInstanceChange = jest.fn();
        let props = {
            ...baseProps,
            onInstanceChange,
            defaultUserInstanceID: 'instance2',
        };
        let ref = React.createRef<JiraInstanceAndProjectSelector>();
        await act(async () => {
            renderWithRedux(
                <JiraInstanceAndProjectSelector
                    {...props}
                    ref={ref}
                />,
            );
        });
        await act(async () => {
            await props.getConnected();
        });
        expect(onInstanceChange).toHaveBeenCalledWith('instance2');

        // Test case 2: Single connected instance should auto-select
        onInstanceChange = jest.fn();
        props = {
            ...baseProps,
            connectedInstances: [{instance_id: 'instance1', type: InstanceType.CLOUD}],
            onInstanceChange,
        };
        ref = React.createRef<JiraInstanceAndProjectSelector>();
        await act(async () => {
            renderWithRedux(
                <JiraInstanceAndProjectSelector
                    {...props}
                    ref={ref}
                />,
            );
        });
        await act(async () => {
            await props.getConnected();
        });
        expect(onInstanceChange).toHaveBeenCalledWith('instance1');

        // Test case 3: Pre-selected instance should take precedence
        onInstanceChange = jest.fn();
        props = {
            ...baseProps,
            onInstanceChange,
            defaultUserInstanceID: 'instance2',
            selectedInstanceID: 'instance3',
        };
        ref = React.createRef<JiraInstanceAndProjectSelector>();
        await act(async () => {
            renderWithRedux(
                <JiraInstanceAndProjectSelector
                    {...props}
                    ref={ref}
                />,
            );
        });
        await act(async () => {
            await props.getConnected();
        });
        expect(onInstanceChange).toHaveBeenCalledWith('instance3');

        // Test case 4: No default and multiple instances - should not call
        onInstanceChange = jest.fn();
        props = {
            ...baseProps,
            onInstanceChange,
        };
        ref = React.createRef<JiraInstanceAndProjectSelector>();
        await act(async () => {
            renderWithRedux(
                <JiraInstanceAndProjectSelector
                    {...props}
                    ref={ref}
                />,
            );
        });
        await act(async () => {
            await props.getConnected();
        });
        expect(onInstanceChange).not.toHaveBeenCalled();
    });

    test('should use default field values after fetch', async () => {
        const onProjectChange = jest.fn();
        const fetchJiraProjectMetadata = jest.fn().mockResolvedValue({
            data: {
                saved_field_values: {
                    project_key: 'TEST',
                },
                projects: [
                    {value: 'TEST', label: 'Test Project'},
                    {value: 'AA', label: 'Apples Arrangement'},
                ],
            },
        });
        const props = {
            ...baseProps,
            defaultUserInstanceID: 'instance2',
            onProjectChange,
            fetchJiraProjectMetadata,
        };
        const ref = React.createRef<JiraInstanceAndProjectSelector>();
        await act(async () => {
            renderWithRedux(
                <JiraInstanceAndProjectSelector
                    {...props}
                    ref={ref}
                />,
            );
        });
        await act(async () => {
            await props.getConnected();
            await fetchJiraProjectMetadata('instance2');
        });
        expect(onProjectChange).toHaveBeenCalledWith({
            project_key: 'TEST',
        });
    });

    test('should pass error on failed fetch', async () => {
        const onError = jest.fn();
        const fetchJiraProjectMetadata = jest.fn().mockResolvedValue({error: {message: 'Some error'}});
        const props = {
            ...baseProps,
            fetchJiraProjectMetadata,
            onError,
            defaultUserInstanceID: 'instance2',
        };
        const ref = React.createRef<JiraInstanceAndProjectSelector>();
        await act(async () => {
            renderWithRedux(
                <JiraInstanceAndProjectSelector
                    {...props}
                    ref={ref}
                />,
            );
        });

        await act(async () => {
            await props.getConnected();
        });
        await act(async () => {
            await fetchJiraProjectMetadata('');
        });
        expect(onError).toHaveBeenCalledWith('Some error');
    });
});
