// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';

import PluginId from 'plugin_id';

import {isDesktopApp} from 'utils/user_agent';
import JiraIcon from 'components/icon';

export default class AttachCommentToIssuePostMenuAction extends PureComponent {
    static propTypes = {
        isSystemMessage: PropTypes.bool.isRequired,
        locale: PropTypes.string,
        open: PropTypes.func.isRequired,
        postId: PropTypes.string,
        userConnected: PropTypes.bool.isRequired,
        isInstanceInstalled: PropTypes.bool.isRequired,
        installedInstanceType: PropTypes.string.isRequired,
        sendEphemeralPost: PropTypes.func.isRequired,
    };

    static defaultTypes = {
        locale: 'en',
    };

    getLocalizedTitle = () => {
        const {locale} = this.props;
        switch (locale) {
        case 'es':
            return 'Crear incidencia en Jira';
        default:
            return 'Attach to Jira Issue';
        }
    };

    handleClick = (e) => {
        const {open, postId} = this.props;
        e.preventDefault();
        open(postId);
    };

    connectClick = () => {
        if (this.props.isInstanceInstalled && this.props.installedInstanceType === 'server' && isDesktopApp()) {
            this.props.sendEphemeralPost('Please use your browser to connect to Jira.');
            return;
        }
        window.open('/plugins/' + PluginId + '/user/connect', '_blank');
    };

    render() {
        if (this.props.isSystemMessage || !this.props.isInstanceInstalled || !this.props.userConnected) {
            return null;
        }

        const content = (
            <button
                className='style--none'
                role='presentation'
                onClick={this.handleClick}
            >
                <JiraIcon type='menu'/>
                {this.getLocalizedTitle()}
            </button>
        );

        return (
            <li
                className='MenuItem'
                role='menuitem'
            >
                {content}
            </li>
        );
    }
}
