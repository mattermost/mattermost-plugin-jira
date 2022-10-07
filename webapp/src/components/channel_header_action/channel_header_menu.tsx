// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import JiraIcon from 'components/icon';

export type Props = {
    open: (channelId: string) => void;
    channelId: string;
    isInstanceInstalled: boolean;
    userConnected: boolean;
};

export default class ChannelHeaderMenuAction extends React.PureComponent<Props> {
    handleClick = (): void => {
        const {open, channelId} = this.props;
        open(channelId);
    };

    render(): React.ReactElement {
        const {isInstanceInstalled, userConnected} = this.props;

        if (!isInstanceInstalled || !userConnected) {
            return null;
        }
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
