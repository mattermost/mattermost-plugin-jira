// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import EditChannelSettings from './edit_channel_settings';

export default class ChannelSettingsModalInner extends React.PureComponent {
    render(): JSX.Element {
        return (
            <EditChannelSettings {...this.props}/>
        );
    }
}
