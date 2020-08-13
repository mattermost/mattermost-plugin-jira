// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import {ChannelSubscription, ProjectMetadata, AllProjectMetadata} from 'types/model';

import BackIcon from '../full_screen_modal/back_icon';

import Loading from 'components/loading';

import EditChannelSettings from './edit_channel_settings';
import SelectChannelSubscription from './select_channel_subscription';
import {SharedProps} from './shared_props';

type State = {
    creatingSubscription: boolean;
    selectedSubscription: ChannelSubscription | null;
    fetching: boolean;
    allProjectMetadata: AllProjectMetadata | null;
}

export default class ChannelSettingsModalInner extends React.PureComponent<SharedProps, State> {
    state = {
        creatingSubscription: false,
        selectedSubscription: null,
        fetching: false,
        allProjectMetadata: null,
    };

    componentDidMount(): void {
        this.fetchData();
    }

    fetchData = async (): Promise<void> => {
        if (!this.props.channel) {
            return;
        }

        this.setState({fetching: true});
        const subsResponse = await this.props.fetchChannelSubscriptions(this.props.channel.id);
        if (subsResponse.error) {
            this.props.sendEphemeralPost('You do not have permission to edit subscriptions for this channel. Subscribing to Jira events will create notifications in this channel when certain events occur, such as an issue being updated or created with a specific label. Speak to your Mattermost administrator to request access to this functionality.');
            this.props.close();
            return;
        }

        const projectResponses = await this.props.fetchJiraProjectMetadataForAllInstances();
        if (projectResponses.error) {
            this.props.sendEphemeralPost('Failed to fetch project metadata for any projects.');
            this.props.close();
            return;
        }

        this.setState({fetching: false, allProjectMetadata: projectResponses.data});
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
        if (!this.props.channelSubscriptions || this.state.fetching) {
            form = <Loading/>;
        } else if (selectedSubscription || creatingSubscription) {
            form = (
                <EditChannelSettings
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
                    allProjectMetadata={this.state.allProjectMetadata}
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
