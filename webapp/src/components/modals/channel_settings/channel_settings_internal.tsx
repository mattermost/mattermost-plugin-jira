// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import {ChannelSubscription} from 'types/model';

import EditChannelSettings from './edit_channel_settings';
import SelectChannelSubscription from './select_channel_subscription';
import {SharedProps} from './shared_props';

export default class ChannelSettingsModalInner extends React.PureComponent<SharedProps> {
    state = {
        creatingSubscription: false,
        selectedSubscription: null,
    };

    showEditChannelSubscription = (subscription: ChannelSubscription): void => {
        this.setState({selectedSubscription: subscription, creatingSubscription: false});
    };

    showCreateChannelSubscription = (): void => {
        this.setState({selectedSubscription: null, creatingSubscription: true});
    };

    finishEditSubscription = (): void => {
        this.setState({selectedSubscription: null, creatingSubscription: false});
    };

    render(): JSX.Element {
        const {selectedSubscription, creatingSubscription} = this.state;

        if (selectedSubscription || creatingSubscription) {
            return (
                <EditChannelSettings
                    {...this.props}
                    close={this.finishEditSubscription}
                    selectedSubscription={selectedSubscription}
                />
            );
        }

        return (
            <SelectChannelSubscription
                {...this.props}
                showEditChannelSubscription={this.showEditChannelSubscription}
                showCreateChannelSubscription={this.showCreateChannelSubscription}
            />
        );
    }
}
