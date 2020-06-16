// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import PropTypes from 'prop-types';

import {getCurrentChannelId} from 'mattermost-redux/selectors/entities/common';

import JiraIcon from 'components/icon';

export default class ChannelHeaderMenuAction extends React.PureComponent {
    static propTypes = {
        theme: PropTypes.object.isRequired,
        open: PropTypes.func.isRequired,
        channelId: PropTypes.string.isRequired,
        isInstanceInstalled: PropTypes.bool.isRequired,
    };

    handleClick = (e) => {
        console.log('handleClick');
        const {open} = this.props;
        e.preventDefault();
        open();
    };

    connectClick = () => {
        console.log('connectClick');

        // if (this.props.isInstanceInstalled && this.props.installedInstanceType === 'server' && isDesktopApp()) {
        this.props.open(this.props.channelId);

        // }
        // window.open('/plugins/' + PluginId + '/user/connect', '_blank');
    };

    render() {
        const content = (
            <button
                className='style--none'
                role='menuitem'

                // onClick={this.handleClick}
                onClick={this.connectClick}
            >
                <JiraIcon type='menu'/>
                {'Jira Subscriptions'}
            </button>
        );

        return (
            {content}
        );
    }
}
