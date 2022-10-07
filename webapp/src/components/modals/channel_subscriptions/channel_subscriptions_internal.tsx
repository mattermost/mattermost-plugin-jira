// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import {ChannelSubscription, AllProjectMetadata} from 'types/model';

import BackIcon from '../full_screen_modal/back_icon';

import EditChannelSubscription from './edit_channel_subscription';
import SelectChannelSubscription from './select_channel_subscription';
import {SharedProps} from './shared_props';

type State = {
    creatingSubscription: boolean;
    selectedSubscription: ChannelSubscription | null;
}

type Props = SharedProps & {
    allProjectMetadata: AllProjectMetadata | null;
}

export default class ChannelSubscriptionsModalInner extends React.PureComponent<Props, State> {
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

    handleBack = (): void => {
        this.setState({
            creatingSubscription: false,
            selectedSubscription: null,
        });
    };

    render(): JSX.Element {
        const {selectedSubscription, creatingSubscription} = this.state;

        let form;
        if (selectedSubscription || creatingSubscription) {
            form = (
                <EditChannelSubscription
                    {...this.props}
                    finishEditSubscription={this.finishEditSubscription}
                    selectedSubscription={selectedSubscription}
                    creatingSubscription={creatingSubscription}
                />
            );
        } else {
            form = (
                <SelectChannelSubscription
                    {...this.props}
                    allProjectMetadata={this.props.allProjectMetadata}
                    showEditChannelSubscription={this.showEditChannelSubscription}
                    showCreateChannelSubscription={this.showCreateChannelSubscription}
                />
            );
        }

        let backIcon;
        if (this.state.creatingSubscription || this.state.selectedSubscription) {
            backIcon = (
                <BackIcon
                    className='back'
                    onClick={this.handleBack}
                />
            );
        }

        return (
            <React.Fragment>
                {backIcon}
                {form}
            </React.Fragment>
        );
    }
}
