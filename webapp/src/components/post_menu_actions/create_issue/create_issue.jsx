// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';

import {FormattedMessage} from 'react-intl';

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
                    <FormattedMessage defaultMessage='Create Jira Issue'/>
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
                    <FormattedMessage defaultMessage='Connect to Jira'/>
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
