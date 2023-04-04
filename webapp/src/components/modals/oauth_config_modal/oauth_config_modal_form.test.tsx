// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {shallow} from 'enzyme';

import {InstanceType} from 'types/model';

import OAuthConfigModalForm from './oauth_config_modal_form';

describe('components/OAuthConfigModalForm', () => {
    const baseActions = {
        closeModal: jest.fn().mockResolvedValue({}),
        configure: jest.fn().mockResolvedValue({}),
        addValidate: jest.fn(),
        removeValidate: jest.fn(),
        installedInstances: [
            {
                instance_id: 'https://something.atlassian.net',
                type: InstanceType.CLOUD_OAUTH,
            },
        ],
    };

    const baseProps = {
        ...baseActions,
        visible: true,
        theme: {},
    };

    test('should match snapshot', () => {
        const props = {...baseProps};
        const wrapper = shallow<OAuthConfigModalForm>(
            <OAuthConfigModalForm {...props}/>
        );
        expect(wrapper).toMatchSnapshot();
    });

    test('should configure on submit', async () => {
        const closeModal = jest.fn().mockResolvedValue({});
        const configure = jest.fn().mockResolvedValue({});
        const props = {
            ...baseProps,
            closeModal,
            configure,
        };
        const wrapper = shallow<OAuthConfigModalForm>(
            <OAuthConfigModalForm {...props}/>
        );

        wrapper.instance().handleInstanceChange('', 'http://localhost:8080');
        expect(wrapper.state().instanceUrl).toEqual('http://localhost:8080');
        expect(wrapper.state().error).toEqual('');

        wrapper.instance().handleClientIdChange('', 'someclientid');
        expect(wrapper.state().clientId).toEqual('someclientid');
        expect(wrapper.state().error).toEqual('');

        wrapper.instance().handleClientSecretChange('', 'someclientsecret');
        expect(wrapper.state().clientSecret).toEqual('someclientsecret');
        expect(wrapper.state().error).toEqual('');
    });

    test('should show error when user is already connected to instance', async () => {
        const props = {...baseProps};
        const wrapper = shallow<OAuthConfigModalForm>(
            <OAuthConfigModalForm {...props}/>
        );

        wrapper.instance().handleInstanceChange('', 'https://something.atlassian.net');
        expect(wrapper.state().instanceUrl).toEqual('https://something.atlassian.net');
        expect(wrapper.state().error).toEqual('You have already installed this Jira instance.');
    });
});
