// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import {AllProjectMetadata, ChannelSubscription} from 'types/model';

import BackIcon from '../full_screen_modal/back_icon';

import EditChannelSubscription from './edit_channel_subscription';
import SelectChannelSubscription from './select_channel_subscription';
import {SharedProps} from './shared_props';

type State = {
    creatingSubscription: boolean;
    selectedSubscription: ChannelSubscription | null;
    creatingSubscriptionTemplate: boolean;
    selectedSubscriptionTemplate: ChannelSubscription | null;

}

type Props = SharedProps & {
    allProjectMetadata: AllProjectMetadata | null;
}

export default class ChannelSubscriptionsModalInner extends React.PureComponent<Props, State> {
    state = {
        creatingSubscription: false,
        selectedSubscription: null,
        creatingSubscriptionTemplate: false,
        selectedSubscriptionTemplate: null,
    };

    showEditChannelSubscription = (subscription: ChannelSubscription): void => {
        this.setState({selectedSubscription: subscription, creatingSubscription: false});
    };

    showEditSubscriptionTemplate = (subscription: ChannelSubscription): void => {
        this.setState({selectedSubscriptionTemplate: subscription, creatingSubscriptionTemplate: false});
    };

    showCreateChannelSubscription = (): void => {
        this.setState({selectedSubscription: null, creatingSubscription: true});
    };

    showCreateSubscriptionTemplate = (): void => {
        this.setState({selectedSubscriptionTemplate: null, creatingSubscriptionTemplate: true});
    };

    finishEditSubscription = (): void => {
        this.setState({selectedSubscription: null, creatingSubscription: false, selectedSubscriptionTemplate: null, creatingSubscriptionTemplate: false});
    };

    handleBack = (): void => {
        this.setState({
            creatingSubscription: false,
            selectedSubscription: null,
            creatingSubscriptionTemplate: false,
            selectedSubscriptionTemplate: null,
        });
    };

    render(): JSX.Element {
        const {selectedSubscription, creatingSubscription, creatingSubscriptionTemplate, selectedSubscriptionTemplate} = this.state;

        let form;
        if (selectedSubscription || creatingSubscription || creatingSubscriptionTemplate || selectedSubscriptionTemplate) {
            form = (
                <EditChannelSubscription
                    {...this.props}
                    finishEditSubscription={this.finishEditSubscription}
                    selectedSubscription={selectedSubscription}
                    creatingSubscription={creatingSubscription}
                    creatingSubscriptionTemplate={creatingSubscriptionTemplate}
                    selectedSubscriptionTemplate={selectedSubscriptionTemplate}
                />
            );
        } else {
            form = (
                <SelectChannelSubscription
                    {...this.props}
                    allProjectMetadata={this.props.allProjectMetadata}
                    showEditChannelSubscription={this.showEditChannelSubscription}
                    showCreateChannelSubscription={this.showCreateChannelSubscription}
                    showEditSubscriptionTemplate={this.showEditSubscriptionTemplate}
                    showCreateSubscriptionTemplate={this.showCreateSubscriptionTemplate}
                />
            );
        }

        let backIcon;
        if (this.state.creatingSubscription || this.state.selectedSubscription || this.state.creatingSubscriptionTemplate || this.state.selectedSubscriptionTemplate) {
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
