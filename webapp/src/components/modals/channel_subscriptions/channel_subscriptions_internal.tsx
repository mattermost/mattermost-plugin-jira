// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import {AllProjectMetadata, ChannelSubscription} from 'types/model';

import EditChannelSubscription from './edit_channel_subscription';
import SelectChannelSubscription, {SubscriptionsTab} from './select_channel_subscription';
import {SharedProps} from './shared_props';

type State = {
    creatingSubscription: boolean;
    selectedSubscription: ChannelSubscription | null;
    creatingSubscriptionTemplate: boolean;
    selectedSubscriptionTemplate: ChannelSubscription | null;
    activeTab: SubscriptionsTab;
    viewChanged: boolean;
}

type Props = SharedProps & {
    allProjectMetadata: AllProjectMetadata | null;
    open: boolean;
}

export default class ChannelSubscriptionsModalInner extends React.PureComponent<Props, State> {
    state: State = {
        creatingSubscription: false,
        selectedSubscription: null,
        creatingSubscriptionTemplate: false,
        selectedSubscriptionTemplate: null,
        activeTab: 'subscriptions',
        viewChanged: false,
    };

    private edit = React.createRef<EditChannelSubscription>();

    requestClose = (): void => {
        if (this.edit.current) {
            this.edit.current.requestClose();
        } else {
            this.props.close();
        }
    };

    showEditChannelSubscription = (subscription: ChannelSubscription): void => {
        this.setState({selectedSubscription: subscription, creatingSubscription: false, viewChanged: true});
    };

    showEditSubscriptionTemplate = (subscription: ChannelSubscription): void => {
        this.setState({selectedSubscriptionTemplate: subscription, creatingSubscriptionTemplate: false, viewChanged: true});
    };

    showCreateChannelSubscription = (): void => {
        this.setState({selectedSubscription: null, creatingSubscription: true, viewChanged: true});
    };

    showCreateSubscriptionTemplate = (): void => {
        this.setState({selectedSubscriptionTemplate: null, creatingSubscriptionTemplate: true, viewChanged: true});
    };

    finishEditSubscription = (): void => {
        this.setState({selectedSubscription: null, creatingSubscription: false, selectedSubscriptionTemplate: null, creatingSubscriptionTemplate: false});
    };

    render(): JSX.Element {
        const {selectedSubscription, creatingSubscription, creatingSubscriptionTemplate, selectedSubscriptionTemplate} = this.state;

        // Only the first view plays the enter transition; later views swap in place.
        let open;
        if (!this.state.viewChanged || !this.props.open) {
            open = this.props.open;
        }

        if (selectedSubscription || creatingSubscription || creatingSubscriptionTemplate || selectedSubscriptionTemplate) {
            return (
                <EditChannelSubscription
                    {...this.props}
                    ref={this.edit}
                    open={open}
                    finishEditSubscription={this.finishEditSubscription}
                    selectedSubscription={selectedSubscription}
                    creatingSubscription={creatingSubscription}
                    creatingSubscriptionTemplate={creatingSubscriptionTemplate}
                    selectedSubscriptionTemplate={selectedSubscriptionTemplate}
                />
            );
        }

        return (
            <SelectChannelSubscription
                {...this.props}
                open={open}
                activeTab={this.state.activeTab}
                onTabChange={(activeTab) => this.setState({activeTab})}
                allProjectMetadata={this.props.allProjectMetadata}
                showEditChannelSubscription={this.showEditChannelSubscription}
                showCreateChannelSubscription={this.showCreateChannelSubscription}
                showEditSubscriptionTemplate={this.showEditSubscriptionTemplate}
                showCreateSubscriptionTemplate={this.showCreateSubscriptionTemplate}
            />
        );
    }
}
