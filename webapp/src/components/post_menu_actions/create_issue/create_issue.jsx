// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';

import PluginId from 'plugin_id';
import {isDesktopApp} from 'utils/user_agent';
import JiraIcon from 'components/icon';

export default class CreateIssuePostMenuAction extends PureComponent {
    static propTypes = {
        isSystemMessage: PropTypes.bool.isRequired,
        open: PropTypes.func.isRequired,
        postId: PropTypes.string,
        userConnected: PropTypes.bool.isRequired,
        installedInstanceType: PropTypes.string.isRequired,
        isInstanceInstalled: PropTypes.bool.isRequired,
        sendEphemeralPost: PropTypes.func.isRequired,
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
        if (this.props.isSystemMessage || !this.props.isInstanceInstalled) {
            return null;
        }

        let content;
        if (this.props.userConnected) {
            content = (
                <button
                    className='style--none'
                    role='presentation'
                    onClick={this.handleClick}
                >
                    <JiraIcon type='menu'/>
                    {this.getLocalizedTitle()}
                </button>
            );
        } else {
            content = (
                <button
                    className='style--none'
                    role='menuitem'
                    onClick={this.connectClick}
                >
                    <JiraIcon type='menu'/>
                    {'Connect to Jira'}
                </button>
            );
        }

        return (
            <React.Fragment>
                <li
                    className='MenuItem'
                    role='menuitem'
                >
                    {content}
                </li>
            </React.Fragment>
        );
    }
}
