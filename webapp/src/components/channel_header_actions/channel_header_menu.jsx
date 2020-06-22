// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import PropTypes from 'prop-types';

import JiraIcon from 'components/icon';

export default class ChannelHeaderMenuAction extends React.PureComponent {
    static propTypes = {
        open: PropTypes.func.isRequired,
        channelId: PropTypes.string.isRequired,
        isInstanceInstalled: PropTypes.bool.isRequired,
        userConnected: PropTypes.bool.isRequired,
    };

    handleClick = () => {
        const {isInstanceInstalled, userConnected, open, channelId} = this.props;
        if (isInstanceInstalled && userConnected) {
            open(channelId);
        }
    };

    render() {
        return (
            <button
                className='style--none'
                role='presentation'
                onClick={this.handleClick}
            >
                <JiraIcon type='menu'/>
                {'Jira Subscriptions'}
            </button>
        );
    }
}
