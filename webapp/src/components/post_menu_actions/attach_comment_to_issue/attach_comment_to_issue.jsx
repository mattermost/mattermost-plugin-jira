// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';

import PluginId from 'plugin_id';

export default class AttachCommentToIssuePostMenuAction extends PureComponent {
    static propTypes = {
        isSystemMessage: PropTypes.bool,
        locale: PropTypes.string,
        open: PropTypes.func.isRequired,
        postId: PropTypes.string,
        connected: PropTypes.object.isRequired,
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
        window.open('/plugins/' + PluginId + '/user/connect');
    }

    render() {
        if (this.props.isSystemMessage) {
            return null;
        }

        const conn = this.props.connected || {};
        let content;
        if (conn.connected) {
            content = (
                <button
                    className='style--none'
                    role='menuitem'
                    onClick={this.handleClick}
                >
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
                    {'Connect to Jira'}
                </button>
            );
        }

        return (
            <li
                role='presentation'
            >
                {content}
            </li>
        );
    }
}
