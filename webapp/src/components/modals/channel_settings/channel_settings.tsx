// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';

import {ProjectMetadata, ChannelSubscription} from 'types/model';

import FullScreenModal from '../full_screen_modal/full_screen_modal';

import ChannelSettingsModalInner from './channel_settings_internal';
import {SharedProps} from './shared_props';

import './channel_settings_modal.scss';

export type Props = SharedProps & {
    fetchJiraProjectMetadata: () => Promise<{data?: ProjectMetadata; error: Error}>;
    fetchChannelSubscriptions: (channelId: string) => Promise<{data: ChannelSubscription[]; error: Error}>;
    sendEphemeralPost: (message: string) => void;
    channelSubscriptions: ChannelSubscription[] | null;
    jiraProjectMetadata: ProjectMetadata | null;
}

export default class ChannelSettingsModal extends PureComponent<Props> {
    componentDidUpdate(prevProps: Props): void {
        if (this.props.channel && (!prevProps.channel || this.props.channel.id !== prevProps.channel.id)) {
            this.fetchData();
        }
    }

    fetchData = async (): Promise<void> => {
        if (!this.props.channel) {
            return;
        }

        this.props.sendEphemeralPost('Retrieving Subscriptions');

        const projectsPromise = this.props.fetchJiraProjectMetadata();
        const subscriptionsPromise = this.props.fetchChannelSubscriptions(this.props.channel.id);

        const subscriptionsResponse = await subscriptionsPromise;
        if (subscriptionsResponse.error) {
            this.props.sendEphemeralPost('You do not have permission to edit subscriptions for this channel. Subscribing to Jira events will create notifications in this channel when certain events occur, such as an issue being updated or created with a specific label. Speak to your Mattermost administrator to request access to this functionality.');
            this.handleClose();
            return;
        }

        const projectsResponse = await projectsPromise;
        if (projectsResponse.error) {
            this.props.sendEphemeralPost('Failed to get Jira project information. Please contact your Mattermost administrator.');
            this.handleClose();
        }
    };

    handleClose = (e?: Event): void => {
        if (e && e.preventDefault) {
            e.preventDefault();
        }

        this.props.close();
    };

    render(): JSX.Element {
        const isModalOpen = Boolean(this.props.channel && this.props.jiraProjectMetadata && this.props.channelSubscriptions);

        let inner;
        if (isModalOpen) {
            inner = (
                <ChannelSettingsModalInner
                    {...this.props}
                />
            );
        }

        return (
            <FullScreenModal
                show={isModalOpen}
                onClose={this.handleClose}
            >
                <div className='channel-subscriptions-modal'>
                    <div className='channel-subscriptions-modal-body'>
                        {inner}
                    </div>
                </div>
            </FullScreenModal>
        );
    }
}
