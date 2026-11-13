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
    pendingView: View | null;
}

type View = Pick<State, 'creatingSubscription' | 'selectedSubscription' | 'creatingSubscriptionTemplate' | 'selectedSubscriptionTemplate'>;

const SUBSCRIPTIONS_VIEW: View = {
    creatingSubscription: false,
    selectedSubscription: null,
    creatingSubscriptionTemplate: false,
    selectedSubscriptionTemplate: null,
};

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
        pendingView: null,
    };

    private edit = React.createRef<EditChannelSubscription>();

    requestClose = (): void => {
        if (this.edit.current) {
            this.edit.current.requestClose();
        } else {
            this.props.close();
        }
    };

    // Switching views plays the current modal's exit, then the next view enters,
    // so the change reads as a transition instead of a swap in place.
    private goTo = (view: View): void => {
        this.setState({pendingView: view});
    };

    private handleExited = (): void => {
        if (this.state.pendingView) {
            this.setState({...this.state.pendingView, pendingView: null});
        }
    };

    showEditChannelSubscription = (subscription: ChannelSubscription): void => {
        this.goTo({...SUBSCRIPTIONS_VIEW, selectedSubscription: subscription});
    };

    showEditSubscriptionTemplate = (subscription: ChannelSubscription): void => {
        this.goTo({...SUBSCRIPTIONS_VIEW, selectedSubscriptionTemplate: subscription});
    };

    showCreateChannelSubscription = (): void => {
        this.goTo({...SUBSCRIPTIONS_VIEW, creatingSubscription: true});
    };

    showCreateSubscriptionTemplate = (): void => {
        this.goTo({...SUBSCRIPTIONS_VIEW, creatingSubscriptionTemplate: true});
    };

    finishEditSubscription = (): void => {
        this.goTo(SUBSCRIPTIONS_VIEW);
    };

    render(): JSX.Element {
        const {selectedSubscription, creatingSubscription, creatingSubscriptionTemplate, selectedSubscriptionTemplate} = this.state;

        const open = this.props.open && !this.state.pendingView;

        if (selectedSubscription || creatingSubscription || creatingSubscriptionTemplate || selectedSubscriptionTemplate) {
            return (
                <EditChannelSubscription
                    {...this.props}
                    ref={this.edit}
                    open={open}
                    onExited={this.handleExited}
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
                onExited={this.handleExited}
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
