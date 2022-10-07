// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';

import JiraIcon from 'components/icon';

export default class CreateIssuePostMenuAction extends PureComponent {
    static propTypes = {
        isSystemMessage: PropTypes.bool.isRequired,
        locale: PropTypes.string,
        open: PropTypes.func.isRequired,
        postId: PropTypes.string,
        userConnected: PropTypes.bool.isRequired,
        userCanConnect: PropTypes.bool.isRequired,
        installedInstances: PropTypes.array.isRequired,
        handleConnectFlow: PropTypes.func.isRequired,
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
            return 'Create Jira Issue';
        }
    };

    handleClick = (e) => {
        const {open, postId} = this.props;
        e.preventDefault();
        open(postId);
    };

    connectClick = () => {
        this.props.handleConnectFlow();
    };

    render() {
        if (this.props.isSystemMessage || !this.props.installedInstances.length) {
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
        } else if (this.props.userCanConnect) {
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
        } else {
            return null;
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
